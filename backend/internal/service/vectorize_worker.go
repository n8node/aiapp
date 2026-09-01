package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

type VectorizeWorker struct {
	kb      *repository.KnowledgeRepository
	disk    *repository.DiskRepository
	store   *ObjectStorage
	extract Extractor
	gw      *GatewayClient
	log     *slog.Logger
	owner   string
}

func NewVectorizeWorker(
	kb *repository.KnowledgeRepository,
	disk *repository.DiskRepository,
	store *ObjectStorage,
	extract Extractor,
	gw *GatewayClient,
	log *slog.Logger,
) *VectorizeWorker {
	host, _ := os.Hostname()
	owner := fmt.Sprintf("%s-%d", host, os.Getpid())
	if len(owner) > 64 {
		owner = owner[:64]
	}
	if log == nil {
		log = slog.Default()
	}
	return &VectorizeWorker{kb: kb, disk: disk, store: store, extract: extract, gw: gw, log: log, owner: owner}
}

func (w *VectorizeWorker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		ok, err := w.ProcessNext(ctx)
		if err != nil {
			w.log.Error("vectorize job failed", "error", publicIngestError(err))
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

func (w *VectorizeWorker) ProcessNext(ctx context.Context) (bool, error) {
	job, err := w.kb.ClaimJob(ctx, w.owner, 10*time.Minute)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	files, err := w.kb.ListFiles(ctx, job.KnowledgeBaseID)
	if err != nil {
		_ = w.kb.FinishJob(ctx, job.ID, model.VectorizeFailed, "extract_failed", 0, 0)
		return true, err
	}
	processed := 0
	total := len(files)
	failCode := ""
	for _, item := range files {
		_ = w.kb.TouchJob(ctx, job.ID, processed, total)
		if err := w.indexFile(ctx, job, item); err != nil {
			code := publicIngestError(err)
			if errors.Is(err, ErrGatewayUnavailable) {
				failCode = "extract_unavailable"
				_ = w.kb.SetFileStatus(ctx, job.KnowledgeBaseID, item.FileID, model.KBFileFailed, item.ContentHash, 0, failCode)
				break
			}
			_ = w.kb.SetFileStatus(ctx, job.KnowledgeBaseID, item.FileID, model.KBFileFailed, item.ContentHash, 0, code)
		}
		processed++
	}
	status := model.VectorizeCompleted
	if failCode != "" {
		status = model.VectorizeFailed
	}
	_ = w.kb.FinishJob(ctx, job.ID, status, failCode, processed, total)
	w.log.Info("vectorize job finished", "job_id", job.ID, "knowledge_base_id", job.KnowledgeBaseID, "processed", processed)
	return true, nil
}

func (w *VectorizeWorker) indexFile(ctx context.Context, job *model.VectorizeJob, item model.KnowledgeBaseFile) error {
	f, err := w.disk.GetFile(ctx, job.WorkspaceID, item.FileID, false)
	if err != nil {
		return err
	}
	if !filesniff.Ingestible(f.Name) {
		return w.kb.SetFileStatus(ctx, job.KnowledgeBaseID, item.FileID, model.KBFileSkipped, "", 0, "not_ingestible")
	}
	max := filesniff.MaxSizeBytes(f.MimeType, f.Name)
	tmp, err := os.CreateTemp("", "rigintel-kb-*")
	if err != nil {
		return err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)
	if err := w.store.DownloadToFile(ctx, f.S3Key, path, max); err != nil {
		return err
	}
	out, err := w.extract.Extract(ctx, path, f.Name, f.MimeType)
	if err != nil {
		return err
	}
	text := truncateRunes(out.Text, maxExtractedRunes)
	hash := hashText(text)
	if item.Status == model.KBFileIndexed && item.ContentHash == hash && item.ChunkCount > 0 {
		return nil
	}
	chunks := chunkText(text, defaultChunkSize, defaultChunkOverlap)
	if kb, err := w.kb.Get(ctx, job.WorkspaceID, job.KnowledgeBaseID); err == nil {
		chunks = chunkText(text, kb.ChunkSize, kb.ChunkOverlap)
	}
	if err := w.kb.DeleteChunksForFile(ctx, job.KnowledgeBaseID, item.FileID); err != nil {
		return err
	}
	if len(chunks) == 0 {
		return w.kb.SetFileStatus(ctx, job.KnowledgeBaseID, item.FileID, model.KBFileIndexed, hash, 0, "")
	}
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}
	const batch = 16
	var vectors [][]float64
	for i := 0; i < len(texts); i += batch {
		end := i + batch
		if end > len(texts) {
			end = len(texts)
		}
		part, err := w.gw.Embed(ctx, texts[i:end], "document")
		if err != nil {
			return err
		}
		vectors = append(vectors, part...)
	}
	for i, c := range chunks {
		if i >= len(vectors) || len(vectors[i]) == 0 {
			continue
		}
		if err := w.kb.InsertChunk(ctx, job.WorkspaceID, job.KnowledgeBaseID, item.FileID, c.Index, c.Text, hash, vectors[i]); err != nil {
			return err
		}
	}
	return w.kb.SetFileStatus(ctx, job.KnowledgeBaseID, item.FileID, model.KBFileIndexed, hash, len(chunks), "")
}
