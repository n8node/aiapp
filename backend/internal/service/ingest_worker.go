package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
	"unicode/utf8"

	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

const ingestLease = 5 * time.Minute

type IngestWorker struct {
	docs    *repository.DocumentRepository
	store   *ObjectStorage
	extract Extractor
	log     *slog.Logger
	owner   string
}

func NewIngestWorker(docs *repository.DocumentRepository, store *ObjectStorage, extract Extractor, log *slog.Logger) *IngestWorker {
	host, _ := os.Hostname()
	owner := fmt.Sprintf("%s-%d", host, os.Getpid())
	if len(owner) > 64 {
		owner = owner[:64]
	}
	if log == nil {
		log = slog.Default()
	}
	return &IngestWorker{docs: docs, store: store, extract: extract, log: log, owner: owner}
}

func (w *IngestWorker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		ok, err := w.ProcessNext(ctx)
		if err != nil {
			w.log.Error("ingest job failed", "error", publicIngestError(err))
		}
		if ok {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (w *IngestWorker) ProcessNext(ctx context.Context) (bool, error) {
	job, err := w.docs.ClaimJob(ctx, w.owner, ingestLease)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	v, err := w.docs.GetVersion(ctx, job.WorkspaceID, job.DocumentVersionID, false)
	if err != nil {
		_ = w.docs.FinishJob(ctx, job.ID, model.IngestJobFailed, "extract_failed")
		return true, err
	}
	_ = w.docs.SetVersionStatus(ctx, job.WorkspaceID, v.ID, model.DocStatusQueued, model.DocStatusRunning)

	code := w.runExtract(ctx, v)
	jobStatus := model.IngestJobCompleted
	if code != "" {
		jobStatus = model.IngestJobFailed
	}
	if err := w.docs.FinishJob(ctx, job.ID, jobStatus, code); err != nil {
		return true, err
	}
	w.log.Info("ingest job finished", "job_id", job.ID, "version_id", v.ID, "workspace_id", job.WorkspaceID, "outcome", jobStatus)
	return true, nil
}

func (w *IngestWorker) runExtract(ctx context.Context, v *model.DocumentVersion) string {
	fail := func(code string) string {
		_ = w.docs.UpdateVersionExtract(ctx, v.WorkspaceID, v.ID, model.DocStatusFailed, "", 0, 0, 0, nil, "", code)
		return code
	}
	if w.extract == nil {
		return fail("extract_unavailable")
	}
	max := filesniff.MaxSizeBytes(v.MimeType, v.OriginalName)
	tmp, err := os.CreateTemp("", "rigintel-ingest-*")
	if err != nil {
		return fail("extract_failed")
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)

	if err := w.store.DownloadToFile(ctx, v.S3Key, path, max); err != nil {
		return fail(publicIngestError(err))
	}

	extractCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	out, err := w.extract.Extract(extractCtx, path, v.OriginalName, v.MimeType)
	if err != nil {
		return fail(publicIngestError(err))
	}

	text := truncateRunes(out.Text, maxExtractedRunes)
	chars := utf8.RuneCountInString(text)
	pages := out.PageCount
	conf := out.Confidence
	engine := out.Engine
	warn := out.Warnings
	if warn == nil {
		warn = []string{}
	}
	if err := w.docs.UpdateVersionExtract(ctx, v.WorkspaceID, v.ID, model.DocStatusAwaitingReview, engine, pages, chars, conf, warn, text, ""); err != nil {
		return fail("extract_failed")
	}
	return ""
}

func publicIngestError(err error) string {
	switch {
	case errors.Is(err, ErrExtractUnavailable):
		return "extract_unavailable"
	case errors.Is(err, ErrExtractFailed):
		return "extract_failed"
	case errors.Is(err, ErrObjectTooLarge):
		return "object_too_large"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return "extract_timeout"
	default:
		return "extract_failed"
	}
}
