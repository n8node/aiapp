package service

import (
	"bytes"
	"context"
	"errors"
	"path"
	"strings"

	"github.com/n8node/aiapp/internal/archivekit"
	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

func (s *DiskService) ExtractArchive(ctx context.Context, userID, sessionID, fileID string) (*model.DiskExtractResult, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	src, err := s.repo.GetFile(ctx, ws.ID, fileID, false)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrDiskFileNotFound
	}
	if err != nil {
		return nil, err
	}
	if !archivekit.IsArchiveName(src.Name) {
		return nil, ErrDiskNotArchive
	}
	max := filesniff.MaxSizeBytes(src.MimeType, src.Name)
	data, err := s.storage.ReadObjectLimited(ctx, src.S3Key, max)
	if err != nil {
		return nil, ErrDiskNotArchive
	}
	entries, err := archivekit.Walk(src.Name, data)
	if errors.Is(err, archivekit.ErrEncrypted) {
		return nil, ErrDiskArchiveEncrypt
	}
	if errors.Is(err, archivekit.ErrTooLarge) || errors.Is(err, archivekit.ErrTooMany) {
		return nil, ErrDiskArchiveLimit
	}
	if err != nil {
		return nil, ErrDiskNotArchive
	}

	base := archivekit.ArchiveBaseName(src.Name)
	if err := validateDiskName(base); err != nil {
		base = "архив"
	}
	folderName, err := s.nextFolderName(ctx, ws.ID, src.FolderID, base)
	if err != nil {
		return nil, err
	}
	uid := userID
	root, err := s.repo.CreateFolder(ctx, &model.DiskFolder{
		WorkspaceID:     ws.ID,
		ParentID:        src.FolderID,
		Name:            folderName,
		CreatedByUserID: &uid,
	})
	if err != nil {
		return nil, err
	}

	createdKeys := []string{}
	createdFileIDs := []string{}
	rollback := func() {
		_ = s.storage.DeleteObjects(ctx, createdKeys)
		_ = s.repo.DeleteFilesByIDs(ctx, ws.ID, createdFileIDs)
		_ = s.purgeFolderTree(ctx, ws.ID, root.ID, false)
	}

	folders := map[string]string{"": root.ID}
	filesCreated := 0
	skipped := 0

	for _, e := range entries {
		if e.IsDir {
			if _, err := s.ensureExtractFolder(ctx, ws.ID, uid, root.ID, e.RelPath, folders); err != nil {
				skipped++
			}
			continue
		}
		if len(e.Data) == 0 {
			skipped++
			continue
		}
		dir := path.Dir(e.RelPath)
		if dir == "." {
			dir = ""
		}
		parentID, err := s.ensureExtractFolder(ctx, ws.ID, uid, root.ID, dir, folders)
		if err != nil {
			skipped++
			continue
		}
		name := path.Base(e.RelPath)
		if err := validateDiskName(name); err != nil {
			skipped++
			continue
		}
		key := buildDiskS3Key(ws.ID)
		mime := filesniff.MIMEFromName(name)
		if err := s.storage.PutObject(ctx, key, mime, bytes.NewReader(e.Data), int64(len(e.Data))); err != nil {
			rollback()
			return nil, ErrDiskStorageWrite
		}
		createdKeys = append(createdKeys, key)
		created, err := s.repo.CreateFile(ctx, &model.DiskFile{
			WorkspaceID:     ws.ID,
			FolderID:        &parentID,
			CreatedByUserID: &uid,
			Name:            name,
			MimeType:        mime,
			Size:            int64(len(e.Data)),
			S3Key:           key,
		})
		if err != nil {
			rollback()
			return nil, err
		}
		createdFileIDs = append(createdFileIDs, created.ID)
		filesCreated++
	}

	s.audit.Write(ctx, userID, "disk.archive.extract", src.ID)
	return &model.DiskExtractResult{Folder: *root, Files: filesCreated, Skipped: skipped}, nil
}

func (s *DiskService) nextFolderName(ctx context.Context, workspaceID string, parentID *string, base string) (string, error) {
	taken := func(name string) bool {
		ok, err := s.repo.SiblingNameTaken(ctx, workspaceID, parentID, name, "")
		if err != nil {
			return true
		}
		return ok
	}
	name, err := archivekit.NextAvailableName(base, taken)
	if err != nil {
		return "", ErrDiskNameTaken
	}
	if err := validateDiskName(name); err != nil {
		return "", err
	}
	return name, nil
}

func (s *DiskService) ensureExtractFolder(ctx context.Context, workspaceID, userID, rootID, rel string, folders map[string]string) (string, error) {
	rel = strings.Trim(rel, "/")
	if rel == "" || rel == "." {
		return rootID, nil
	}
	if id, ok := folders[rel]; ok {
		return id, nil
	}
	parentRel := path.Dir(rel)
	if parentRel == "." {
		parentRel = ""
	}
	parentID, err := s.ensureExtractFolder(ctx, workspaceID, userID, rootID, parentRel, folders)
	if err != nil {
		return "", err
	}
	name := path.Base(rel)
	if err := validateDiskName(name); err != nil {
		return "", err
	}
	uid := userID
	created, err := s.repo.CreateFolder(ctx, &model.DiskFolder{
		WorkspaceID:     workspaceID,
		ParentID:        &parentID,
		Name:            name,
		CreatedByUserID: &uid,
	})
	if err != nil {
		return "", err
	}
	folders[rel] = created.ID
	return created.ID, nil
}
