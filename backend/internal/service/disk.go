package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

var (
	ErrDiskFileNotFound   = errors.New("disk file not found")
	ErrDiskFolderNotFound = errors.New("disk folder not found")
	ErrDiskFileTooLarge   = errors.New("disk file too large")
	ErrDiskEmptyFile      = errors.New("disk empty file")
	ErrDiskTypeRejected   = errors.New("disk type rejected")
	ErrDiskNoWorkspace    = errors.New("disk no workspace")
	ErrDiskInvalidMove    = errors.New("disk invalid move")
	ErrDiskNameTaken      = errors.New("disk name taken")
	ErrDiskUploadSession  = errors.New("disk upload session")
)

type DiskService struct {
	repo     *repository.DiskRepository
	auth     *AuthService
	storage  *ObjectStorage
	sessions *UploadSessionService
	audit    *repository.AuditRepository
}

func NewDiskService(
	repo *repository.DiskRepository,
	auth *AuthService,
	storage *ObjectStorage,
	sessions *UploadSessionService,
	audit *repository.AuditRepository,
) *DiskService {
	return &DiskService{repo: repo, auth: auth, storage: storage, sessions: sessions, audit: audit}
}

func (s *DiskService) resolve(ctx context.Context, userID, sessionID string) (*model.Workspace, error) {
	_, _, active, err := s.auth.Me(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, ErrDiskNoWorkspace
	}
	return active, nil
}

func (s *DiskService) UploadInit(ctx context.Context, userID, sessionID string, req model.FileUploadInitRequest) (*model.FileUploadInitResponse, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if err := validateDiskName(name); err != nil {
		return nil, err
	}
	if req.Size <= 0 {
		return nil, ErrDiskEmptyFile
	}
	if !filesniff.AllowedExtension(name) {
		return nil, ErrDiskTypeRejected
	}
	if !filesniff.MIMEMatchesExtension(req.MimeType, name) {
		return nil, ErrDiskTypeRejected
	}
	if req.Size > filesniff.MaxSizeBytes(req.MimeType, name) {
		return nil, ErrDiskFileTooLarge
	}
	if err := s.repo.FolderExistsActive(ctx, ws.ID, req.FolderID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDiskFolderNotFound
		}
		return nil, err
	}
	s3Key := buildDiskS3Key(ws.ID)
	presigned, err := s.storage.PresignPut(ctx, s3Key, req.MimeType, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	token, err := s.sessions.Create(UploadSessionClaims{
		WorkspaceID: ws.ID,
		UserID:      userID,
		S3Key:       s3Key,
		Name:        name,
		MimeType:    strings.TrimSpace(req.MimeType),
		Size:        req.Size,
		FolderID:    req.FolderID,
	})
	if err != nil {
		return nil, err
	}
	return &model.FileUploadInitResponse{
		UploadURL:          presigned.URL,
		UploadHeaders:      presigned.Headers,
		UploadSessionToken: token,
	}, nil
}

func (s *DiskService) UploadComplete(ctx context.Context, userID, sessionID, token string) (*model.DiskFile, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	claims, err := s.sessions.Verify(token)
	if err != nil {
		return nil, ErrDiskUploadSession
	}
	if claims.WorkspaceID != ws.ID || claims.UserID != userID {
		return nil, ErrForbidden
	}
	if existing, err := s.repo.GetFileByS3Key(ctx, ws.ID, claims.S3Key); err == nil {
		return existing, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	head, err := s.storage.HeadObject(ctx, claims.S3Key)
	if err != nil {
		return nil, ErrDiskUploadSession
	}
	if head.Size != claims.Size {
		_ = s.storage.DeleteObject(ctx, claims.S3Key)
		return nil, ErrDiskUploadSession
	}
	prefix, err := s.storage.ReadHead(ctx, claims.S3Key, filesniff.MaxHead)
	if err != nil || !filesniff.HeadMatches(claims.Name, prefix) {
		_ = s.storage.DeleteObject(ctx, claims.S3Key)
		return nil, ErrDiskTypeRejected
	}
	mime := claims.MimeType
	if mime == "" {
		mime = "application/octet-stream"
	}
	uid := userID
	created, err := s.repo.CreateFile(ctx, &model.DiskFile{
		WorkspaceID:     ws.ID,
		FolderID:        claims.FolderID,
		CreatedByUserID: &uid,
		Name:            claims.Name,
		MimeType:        mime,
		Size:            claims.Size,
		S3Key:           claims.S3Key,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, userID, "disk.file.upload", created.ID)
	return created, nil
}

func (s *DiskService) ListFiles(ctx context.Context, userID, sessionID, section string, folderID *string) ([]model.DiskFile, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	filter := repository.DiskFileFilter{WorkspaceID: ws.ID, FolderID: folderID}
	switch section {
	case "recent":
		filter.ScopeAll = true
		filter.RecentOnly = true
		filter.Limit = 100
	case "photos":
		filter.ScopeAll = true
		filter.TypeFilter = "image"
		filter.Limit = 200
	case "videos":
		filter.ScopeAll = true
		filter.TypeFilter = "video"
		filter.Limit = 200
	default:
		if err := s.repo.FolderExistsActive(ctx, ws.ID, folderID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrDiskFolderNotFound
			}
			return nil, err
		}
	}
	items, err := s.repo.ListFiles(ctx, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.DiskFile{}
	}
	return items, nil
}

func (s *DiskService) GetFile(ctx context.Context, userID, sessionID, fileID string) (*model.DiskFile, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	f, err := s.repo.GetFile(ctx, ws.ID, fileID, false)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrDiskFileNotFound
	}
	return f, err
}

func (s *DiskService) DownloadURL(ctx context.Context, userID, sessionID, fileID string, inline bool) (string, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	f, err := s.repo.GetFile(ctx, ws.ID, fileID, false)
	if errors.Is(err, repository.ErrNotFound) {
		return "", ErrDiskFileNotFound
	}
	if err != nil {
		return "", err
	}
	return s.storage.PresignGet(ctx, f.S3Key, f.Name, inline, 5*time.Minute)
}

func (s *DiskService) PatchFile(ctx context.Context, userID, sessionID, fileID string, req model.DiskFilePatchRequest) (*model.DiskFile, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if req.Name == nil {
		return nil, ErrInvalidInput
	}
	name := strings.TrimSpace(*req.Name)
	if err := validateDiskName(name); err != nil {
		return nil, err
	}
	f, err := s.repo.UpdateFileName(ctx, ws.ID, fileID, name)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrDiskFileNotFound
	}
	return f, err
}

func (s *DiskService) MoveFile(ctx context.Context, userID, sessionID, fileID string, folderID *string) (*model.DiskFile, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.FolderExistsActive(ctx, ws.ID, folderID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDiskFolderNotFound
		}
		return nil, err
	}
	f, err := s.repo.UpdateFileFolder(ctx, ws.ID, fileID, folderID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrDiskFileNotFound
	}
	return f, err
}

func (s *DiskService) CopyFile(ctx context.Context, userID, sessionID, fileID string, folderID *string) (*model.DiskFile, error) {
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
	if err := s.repo.FolderExistsActive(ctx, ws.ID, folderID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDiskFolderNotFound
		}
		return nil, err
	}
	dstKey := buildDiskS3Key(ws.ID)
	if err := s.storage.CopyObject(ctx, src.S3Key, dstKey); err != nil {
		return nil, err
	}
	uid := userID
	created, err := s.repo.CreateFile(ctx, &model.DiskFile{
		WorkspaceID:     ws.ID,
		FolderID:        folderID,
		CreatedByUserID: &uid,
		Name:            duplicateName(src.Name),
		MimeType:        src.MimeType,
		Size:            src.Size,
		S3Key:           dstKey,
	})
	if err != nil {
		_ = s.storage.DeleteObject(ctx, dstKey)
		return nil, err
	}
	s.audit.Write(ctx, userID, "disk.file.copy", created.ID)
	return created, nil
}

func (s *DiskService) DeleteFile(ctx context.Context, userID, sessionID, fileID string) error {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if _, err := s.repo.GetFile(ctx, ws.ID, fileID, false); errors.Is(err, repository.ErrNotFound) {
		return ErrDiskFileNotFound
	} else if err != nil {
		return err
	}
	if err := s.repo.SoftDeleteFiles(ctx, ws.ID, []string{fileID}, newTrashBatch(), time.Now().UTC()); err != nil {
		return err
	}
	s.audit.Write(ctx, userID, "disk.file.trash", fileID)
	return nil
}

func (s *DiskService) BulkFiles(ctx context.Context, userID, sessionID string, req model.DiskBulkRequest) (*model.DiskBulkResult, error) {
	res := &model.DiskBulkResult{}
	if len(req.IDs) > 50 {
		return nil, ErrInvalidInput
	}
	for _, id := range req.IDs {
		var err error
		switch req.Action {
		case "delete":
			err = s.DeleteFile(ctx, userID, sessionID, id)
		case "move":
			_, err = s.MoveFile(ctx, userID, sessionID, id, req.FolderID)
		case "copy":
			_, err = s.CopyFile(ctx, userID, sessionID, id, req.FolderID)
		default:
			return nil, ErrInvalidInput
		}
		if err != nil {
			res.Errors = append(res.Errors, model.DiskBulkItemError{ID: id, Message: "не удалось"})
			continue
		}
		res.OK++
	}
	return res, nil
}

func (s *DiskService) ListFolders(ctx context.Context, userID, sessionID string, parentID *string, scopeAll bool) ([]model.DiskFolder, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if !scopeAll {
		if err := s.repo.FolderExistsActive(ctx, ws.ID, parentID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrDiskFolderNotFound
			}
			return nil, err
		}
	}
	items, err := s.repo.ListFolders(ctx, ws.ID, parentID, scopeAll)
	if err != nil {
		return nil, err
	}
	for i := range items {
		n, _ := s.repo.CountFilesInFolder(ctx, items[i].ID)
		items[i].FilesCount = n
	}
	if items == nil {
		items = []model.DiskFolder{}
	}
	return items, nil
}

func (s *DiskService) CreateFolder(ctx context.Context, userID, sessionID, name string, parentID *string) (*model.DiskFolder, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if err := validateDiskName(name); err != nil {
		return nil, err
	}
	if err := s.repo.FolderExistsActive(ctx, ws.ID, parentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDiskFolderNotFound
		}
		return nil, err
	}
	taken, err := s.repo.SiblingNameTaken(ctx, ws.ID, parentID, name, "")
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrDiskNameTaken
	}
	uid := userID
	created, err := s.repo.CreateFolder(ctx, &model.DiskFolder{
		WorkspaceID:     ws.ID,
		ParentID:        parentID,
		Name:            name,
		CreatedByUserID: &uid,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, userID, "disk.folder.create", created.ID)
	return created, nil
}

func (s *DiskService) PatchFolder(ctx context.Context, userID, sessionID, folderID string, req model.DiskFolderPatchRequest) (*model.DiskFolder, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if req.Name == nil {
		return nil, ErrInvalidInput
	}
	name := strings.TrimSpace(*req.Name)
	if err := validateDiskName(name); err != nil {
		return nil, err
	}
	cur, err := s.repo.GetFolder(ctx, ws.ID, folderID, false)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrDiskFolderNotFound
	}
	if err != nil {
		return nil, err
	}
	taken, err := s.repo.SiblingNameTaken(ctx, ws.ID, cur.ParentID, name, folderID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrDiskNameTaken
	}
	return s.repo.UpdateFolderName(ctx, ws.ID, folderID, name)
}

func (s *DiskService) MoveFolder(ctx context.Context, userID, sessionID, folderID string, parentID *string) (*model.DiskFolder, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.GetFolder(ctx, ws.ID, folderID, false); errors.Is(err, repository.ErrNotFound) {
		return nil, ErrDiskFolderNotFound
	} else if err != nil {
		return nil, err
	}
	if parentID != nil && strings.TrimSpace(*parentID) == "" {
		parentID = nil
	}
	if parentID != nil {
		if *parentID == folderID {
			return nil, ErrDiskInvalidMove
		}
		if err := s.repo.FolderExistsActive(ctx, ws.ID, parentID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrDiskFolderNotFound
			}
			return nil, err
		}
		desc, err := s.repo.IsDescendantOf(ctx, *parentID, folderID)
		if err != nil {
			return nil, err
		}
		if desc {
			return nil, ErrDiskInvalidMove
		}
	}
	return s.repo.UpdateFolderParent(ctx, ws.ID, folderID, parentID)
}

func (s *DiskService) DeleteFolder(ctx context.Context, userID, sessionID, folderID string) error {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if _, err := s.repo.GetFolder(ctx, ws.ID, folderID, false); errors.Is(err, repository.ErrNotFound) {
		return ErrDiskFolderNotFound
	} else if err != nil {
		return err
	}
	folderIDs, err := s.repo.CollectSubtreeIDs(ctx, folderID, true)
	if err != nil {
		return err
	}
	var fileIDs []string
	for _, fid := range folderIDs {
		ids, err := s.repo.CollectFileIDsInFolder(ctx, fid, true)
		if err != nil {
			return err
		}
		fileIDs = append(fileIDs, ids...)
	}
	batch := newTrashBatch()
	now := time.Now().UTC()
	if err := s.repo.SoftDeleteFiles(ctx, ws.ID, fileIDs, batch, now); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteFolders(ctx, ws.ID, folderIDs, batch, now); err != nil {
		return err
	}
	s.audit.Write(ctx, userID, "disk.folder.trash", folderID)
	return nil
}

func (s *DiskService) BulkFolders(ctx context.Context, userID, sessionID string, req model.DiskBulkRequest) (*model.DiskBulkResult, error) {
	res := &model.DiskBulkResult{}
	if len(req.IDs) > 50 {
		return nil, ErrInvalidInput
	}
	for _, id := range req.IDs {
		var err error
		switch req.Action {
		case "delete":
			err = s.DeleteFolder(ctx, userID, sessionID, id)
		case "move":
			_, err = s.MoveFolder(ctx, userID, sessionID, id, req.FolderID)
		default:
			return nil, ErrInvalidInput
		}
		if err != nil {
			res.Errors = append(res.Errors, model.DiskBulkItemError{ID: id, Message: "не удалось"})
			continue
		}
		res.OK++
	}
	return res, nil
}

func (s *DiskService) Breadcrumbs(ctx context.Context, userID, sessionID, folderID string) ([]model.FolderBreadcrumb, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	rootID := (*string)(nil)
	crumbs := []model.FolderBreadcrumb{{ID: rootID, Name: "Файлы"}}
	more, err := s.repo.Breadcrumbs(ctx, ws.ID, folderID)
	if err != nil {
		return nil, err
	}
	if len(more) == 0 {
		return nil, ErrDiskFolderNotFound
	}
	return append(crumbs, more...), nil
}

func (s *DiskService) ListTrash(ctx context.Context, userID, sessionID string) ([]model.DiskFile, []model.DiskFolder, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, nil, err
	}
	files, err := s.repo.ListTrashedFilesTop(ctx, ws.ID)
	if err != nil {
		return nil, nil, err
	}
	folders, err := s.repo.ListTrashedFoldersTop(ctx, ws.ID)
	if err != nil {
		return nil, nil, err
	}
	if files == nil {
		files = []model.DiskFile{}
	}
	if folders == nil {
		folders = []model.DiskFolder{}
	}
	return files, folders, nil
}

func (s *DiskService) RestoreTrash(ctx context.Context, userID, sessionID string, fileIDs, folderIDs []string) error {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	for _, id := range folderIDs {
		f, err := s.repo.GetFolder(ctx, ws.ID, id, true)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrDiskFolderNotFound
		}
		if err != nil {
			return err
		}
		if f.TrashBatchID != nil && *f.TrashBatchID != "" {
			if err := s.repo.RestoreFoldersByBatch(ctx, ws.ID, *f.TrashBatchID); err != nil {
				return err
			}
			if err := s.repo.RestoreFilesByBatch(ctx, ws.ID, *f.TrashBatchID); err != nil {
				return err
			}
			continue
		}
		parent := f.ParentID
		if parent != nil {
			if _, err := s.repo.GetFolder(ctx, ws.ID, *parent, false); err != nil {
				parent = nil
			}
		}
		if err := s.repo.RestoreFolder(ctx, ws.ID, id, parent); err != nil {
			return err
		}
	}
	for _, id := range fileIDs {
		f, err := s.repo.GetFile(ctx, ws.ID, id, true)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrDiskFileNotFound
		}
		if err != nil {
			return err
		}
		folderID := f.FolderID
		if folderID != nil {
			if _, err := s.repo.GetFolder(ctx, ws.ID, *folderID, false); err != nil {
				folderID = nil
			}
		}
		if err := s.repo.RestoreFile(ctx, ws.ID, id, folderID); err != nil {
			return err
		}
	}
	s.audit.Write(ctx, userID, "disk.trash.restore", ws.ID)
	return nil
}

func (s *DiskService) PermanentDelete(ctx context.Context, userID, sessionID, id, kind string) error {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if kind == "folder" {
		f, err := s.repo.GetFolder(ctx, ws.ID, id, true)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrDiskFolderNotFound
		}
		if err != nil {
			return err
		}
		folderIDs, err := s.repo.CollectSubtreeIDs(ctx, f.ID, false)
		if err != nil {
			return err
		}
		var fileIDs []string
		for _, fid := range folderIDs {
			ids, err := s.repo.CollectFileIDsInFolder(ctx, fid, false)
			if err != nil {
				return err
			}
			fileIDs = append(fileIDs, ids...)
		}
		files, err := s.repo.ListFilesByIDs(ctx, ws.ID, fileIDs, true)
		if err != nil {
			return err
		}
		for _, file := range files {
			_ = s.storage.DeleteObject(ctx, file.S3Key)
			_, _ = s.repo.DeleteFilePermanent(ctx, ws.ID, file.ID)
		}
		_ = s.repo.DeleteFoldersByIDs(ctx, ws.ID, folderIDs)
		s.audit.Write(ctx, userID, "disk.folder.purge", id)
		return nil
	}
	f, err := s.repo.GetFile(ctx, ws.ID, id, true)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrDiskFileNotFound
	}
	if err != nil {
		return err
	}
	_ = s.storage.DeleteObject(ctx, f.S3Key)
	if _, err := s.repo.DeleteFilePermanent(ctx, ws.ID, id); err != nil {
		return err
	}
	s.audit.Write(ctx, userID, "disk.file.purge", id)
	return nil
}

func (s *DiskService) EmptyTrash(ctx context.Context, userID, sessionID string) error {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	files, err := s.repo.ListAllTrashedFiles(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, f := range files {
		_ = s.storage.DeleteObject(ctx, f.S3Key)
		_, _ = s.repo.DeleteFilePermanent(ctx, ws.ID, f.ID)
	}
	folders, err := s.repo.ListTrashedFoldersTop(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, folder := range folders {
		ids, err := s.repo.CollectSubtreeIDs(ctx, folder.ID, false)
		if err != nil {
			return err
		}
		_ = s.repo.DeleteFoldersByIDs(ctx, ws.ID, ids)
	}
	s.audit.Write(ctx, userID, "disk.trash.empty", ws.ID)
	return nil
}

func validateDiskName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return ErrInvalidInput
	}
	if strings.ContainsAny(name, "/\\\x00") {
		return ErrInvalidInput
	}
	return nil
}

func buildDiskS3Key(workspaceID string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("workspaces/%s/files/%s", workspaceID, hex.EncodeToString(b))
}

func duplicateName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "файл (копия)"
	}
	if i := strings.LastIndex(name, "."); i > 0 && i < len(name)-1 {
		return name[:i] + " (копия)" + name[i:]
	}
	return name + " (копия)"
}

func newTrashBatch() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
