package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

var (
	ErrDocumentNotFound      = errors.New("document not found")
	ErrDocumentNotIngestible = errors.New("document not ingestible")
	ErrDocumentInvalidState  = errors.New("document invalid state")
	ErrExtractUnavailable    = errors.New("extract unavailable")
	ErrExtractFailed         = errors.New("extract failed")
)

const maxExtractedRunes = 500_000
const maxReviewNoteRunes = 500

type Extractor interface {
	Extract(ctx context.Context, path, originalName, mimeType string) (*model.ExtractResult, error)
}

type DocumentService struct {
	docs      *repository.DocumentRepository
	disk      *DiskService
	storage   *ObjectStorage
	auth      *AuthService
	audit     *repository.AuditRepository
	extractor Extractor
}

func NewDocumentService(
	docs *repository.DocumentRepository,
	disk *DiskService,
	storage *ObjectStorage,
	auth *AuthService,
	audit *repository.AuditRepository,
	extractor Extractor,
) *DocumentService {
	return &DocumentService{docs: docs, disk: disk, storage: storage, auth: auth, audit: audit, extractor: extractor}
}

func versionReusable(status string) bool {
	switch status {
	case model.DocStatusQueued, model.DocStatusRunning, model.DocStatusAwaitingReview, model.DocStatusPublished:
		return true
	default:
		return false
	}
}

func versionRetryable(status string) bool {
	return status == model.DocStatusFailed || status == model.DocStatusRejected
}

func applyReview(status, action string) (string, error) {
	if status != model.DocStatusAwaitingReview {
		return "", ErrDocumentInvalidState
	}
	switch action {
	case "approve":
		return model.DocStatusPublished, nil
	case "reject":
		return model.DocStatusRejected, nil
	default:
		return "", ErrInvalidInput
	}
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func clipTitle(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Документ"
	}
	return truncateRunes(name, 255)
}

func (s *DocumentService) resolve(ctx context.Context, userID, sessionID string) (*model.Workspace, error) {
	_, _, active, err := s.auth.Me(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, ErrDiskNoWorkspace
	}
	return active, nil
}

func (s *DocumentService) attach(ctx context.Context, wsID string, d *model.Document, includeText bool) (*model.Document, error) {
	if d == nil || d.CurrentVersionID == nil || *d.CurrentVersionID == "" {
		return d, nil
	}
	v, err := s.docs.GetVersion(ctx, wsID, *d.CurrentVersionID, includeText)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return d, nil
		}
		return nil, err
	}
	d.CurrentVersion = v
	return d, nil
}

func (s *DocumentService) resultForVersion(ctx context.Context, wsID string, v *model.DocumentVersion, jobID *string, created bool) (*model.DocumentFromFileResult, error) {
	d, err := s.docs.Get(ctx, wsID, v.DocumentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	d, err = s.attach(ctx, wsID, d, false)
	if err != nil {
		return nil, err
	}
	return &model.DocumentFromFileResult{Document: d, JobID: jobID, Created: created}, nil
}

func (s *DocumentService) FromFile(ctx context.Context, userID, sessionID string, req model.DocumentFromFileRequest) (*model.DocumentFromFileResult, error) {
	fileID := strings.TrimSpace(req.FileID)
	if fileID == "" {
		return nil, ErrInvalidInput
	}
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	file, err := s.disk.GetFile(ctx, userID, sessionID, fileID)
	if err != nil {
		return nil, err
	}
	if !filesniff.Ingestible(file.Name) {
		return nil, ErrDocumentNotIngestible
	}

	existing, err := s.docs.FindByDiskFile(ctx, ws.ID, file.ID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		if versionReusable(existing.Status) {
			return s.resultForVersion(ctx, ws.ID, existing, nil, false)
		}
		if versionRetryable(existing.Status) {
			return s.Retry(ctx, userID, sessionID, existing.DocumentID, existing.ID)
		}
	}

	max := filesniff.MaxSizeBytes(file.MimeType, file.Name)
	if file.Size > max {
		return nil, ErrObjectTooLarge
	}
	hash, size, err := s.storage.HashAndSize(ctx, file.S3Key, max)
	if err != nil {
		return nil, err
	}

	if dup, err := s.docs.FindByHash(ctx, ws.ID, hash); err == nil {
		return s.resultForVersion(ctx, ws.ID, dup, nil, false)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	uid := userID
	doc, err := s.docs.Create(ctx, &model.Document{
		WorkspaceID:     ws.ID,
		Title:           clipTitle(file.Name),
		CreatedByUserID: &uid,
	})
	if err != nil {
		return nil, err
	}
	n, err := s.docs.NextVersionN(ctx, doc.ID)
	if err != nil {
		return nil, err
	}
	ver, err := s.docs.CreateVersion(ctx, &model.DocumentVersion{
		DocumentID:   doc.ID,
		WorkspaceID:  ws.ID,
		VersionN:     n,
		DiskFileID:   &file.ID,
		S3Key:        file.S3Key,
		OriginalName: file.Name,
		MimeType:     file.MimeType,
		Size:         size,
		ContentHash:  hash,
		Status:       model.DocStatusQueued,
		Warnings:     []string{},
	})
	if err != nil {
		return nil, err
	}
	if err := s.docs.SetCurrentVersion(ctx, ws.ID, doc.ID, ver.ID); err != nil {
		return nil, err
	}
	job, err := s.docs.EnqueueJob(ctx, &model.IngestJob{
		WorkspaceID:       ws.ID,
		DocumentVersionID: ver.ID,
		CreatedByUserID:   &uid,
	})
	if err != nil {
		_ = s.docs.UpdateVersionExtract(ctx, ws.ID, ver.ID, model.DocStatusFailed, "", 0, 0, 0, nil, "", "extract_unavailable")
		return nil, err
	}
	s.audit.Write(ctx, userID, "document.ingest", doc.ID)
	return s.resultForVersion(ctx, ws.ID, ver, &job.ID, true)
}

func (s *DocumentService) List(ctx context.Context, userID, sessionID string) ([]model.Document, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	items, err := s.docs.List(ctx, ws.ID, 50)
	if err != nil {
		return nil, err
	}
	for i := range items {
		d, err := s.attach(ctx, ws.ID, &items[i], false)
		if err != nil {
			return nil, err
		}
		items[i] = *d
	}
	return items, nil
}

func (s *DocumentService) Get(ctx context.Context, userID, sessionID, documentID string) (*model.Document, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	d, err := s.docs.Get(ctx, ws.ID, documentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrDocumentNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.attach(ctx, ws.ID, d, true)
}

func (s *DocumentService) Review(ctx context.Context, userID, sessionID, documentID, versionID string, req model.DocumentReviewRequest) (*model.Document, error) {
	action := strings.TrimSpace(strings.ToLower(req.Action))
	note := ""
	if req.Note != nil {
		note = strings.TrimSpace(*req.Note)
		if utf8.RuneCountInString(note) > maxReviewNoteRunes {
			return nil, ErrInvalidInput
		}
	}
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	v, err := s.docs.GetVersion(ctx, ws.ID, versionID, false)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	if v.DocumentID != documentID {
		return nil, ErrDocumentNotFound
	}
	to, err := applyReview(v.Status, action)
	if err != nil {
		return nil, err
	}
	if err := s.docs.UpdateVersionStatus(ctx, ws.ID, versionID, v.Status, to, userID, note); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentInvalidState
		}
		return nil, err
	}
	s.audit.Write(ctx, userID, "document.review", documentID)
	return s.Get(ctx, userID, sessionID, documentID)
}

func (s *DocumentService) Retry(ctx context.Context, userID, sessionID, documentID, versionID string) (*model.DocumentFromFileResult, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	v, err := s.docs.GetVersion(ctx, ws.ID, versionID, false)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	if v.DocumentID != documentID {
		return nil, ErrDocumentNotFound
	}
	if !versionRetryable(v.Status) {
		return nil, ErrDocumentInvalidState
	}
	if err := s.docs.ResetVersionForRetry(ctx, ws.ID, versionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentInvalidState
		}
		return nil, err
	}
	uid := userID
	job, err := s.docs.EnqueueJob(ctx, &model.IngestJob{
		WorkspaceID:       ws.ID,
		DocumentVersionID: versionID,
		CreatedByUserID:   &uid,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, userID, "document.retry", documentID)
	v.Status = model.DocStatusQueued
	return s.resultForVersion(ctx, ws.ID, v, &job.ID, false)
}
