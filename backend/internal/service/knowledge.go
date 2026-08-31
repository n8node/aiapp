package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

type KnowledgeService struct {
	kb     *repository.KnowledgeRepository
	models *repository.MLModelRepository
	disk   *DiskService
	auth   *AuthService
	audit  *repository.AuditRepository
	gw     *GatewayClient
}

func NewKnowledgeService(
	kb *repository.KnowledgeRepository,
	models *repository.MLModelRepository,
	disk *DiskService,
	auth *AuthService,
	audit *repository.AuditRepository,
	gw *GatewayClient,
) *KnowledgeService {
	return &KnowledgeService{kb: kb, models: models, disk: disk, auth: auth, audit: audit, gw: gw}
}

func (s *KnowledgeService) resolve(ctx context.Context, userID, sessionID string) (*model.Workspace, error) {
	_, _, active, err := s.auth.Me(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, ErrDiskNoWorkspace
	}
	return active, nil
}

func (s *KnowledgeService) withFiles(ctx context.Context, k *model.KnowledgeBase) (*model.KnowledgeBase, error) {
	files, err := s.kb.ListFiles(ctx, k.ID)
	if err != nil {
		return nil, err
	}
	k.Files = files
	k.FileCount = len(files)
	indexed := 0
	for _, f := range files {
		if f.Status == model.KBFileIndexed {
			indexed++
		}
	}
	k.IndexedCount = indexed
	return k, nil
}

func (s *KnowledgeService) Create(ctx context.Context, userID, sessionID string, req model.KnowledgeBaseCreateRequest) (*model.KnowledgeBase, error) {
	name := clipName(req.Name, 255)
	if name == "" {
		return nil, ErrInvalidInput
	}
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	var embedID *string
	if m, err := s.models.DeployedByPurpose(ctx, model.ModelPurposeEmbeddings); err == nil {
		embedID = &m.ID
	}
	uid := userID
	k, err := s.kb.Create(ctx, &model.KnowledgeBase{
		WorkspaceID:      ws.ID,
		Name:             name,
		ChunkSize:        defaultChunkSize,
		ChunkOverlap:     defaultChunkOverlap,
		EmbeddingModelID: embedID,
	}, &uid)
	if err != nil {
		return nil, err
	}
	ids := uniqueIDs(req.FileIDs)
	if req.FolderID != nil && strings.TrimSpace(*req.FolderID) != "" {
		listed, err := s.disk.ListFiles(ctx, userID, sessionID, "my-files", req.FolderID)
		if err != nil {
			return nil, err
		}
		for _, f := range listed {
			if filesniff.Ingestible(f.Name) {
				ids = append(ids, f.ID)
			}
		}
		ids = uniqueIDs(ids)
	}
	if err := s.attach(ctx, userID, sessionID, ws.ID, k.ID, ids); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, userID, "knowledge.create", k.ID)
	return s.withFiles(ctx, k)
}

func (s *KnowledgeService) List(ctx context.Context, userID, sessionID string) ([]model.KnowledgeBase, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	return s.kb.List(ctx, ws.ID)
}

func (s *KnowledgeService) Get(ctx context.Context, userID, sessionID, id string) (*model.KnowledgeBase, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	k, err := s.kb.Get(ctx, ws.ID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrKnowledgeNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.withFiles(ctx, k)
}

func (s *KnowledgeService) Delete(ctx context.Context, userID, sessionID, id string) error {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if err := s.kb.Delete(ctx, ws.ID, id); errors.Is(err, repository.ErrNotFound) {
		return ErrKnowledgeNotFound
	} else if err != nil {
		return err
	}
	s.audit.Write(ctx, userID, "knowledge.delete", id)
	return nil
}

func (s *KnowledgeService) Patch(ctx context.Context, userID, sessionID, id string, req model.KnowledgeBasePatchRequest) (*model.KnowledgeBase, error) {
	ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.kb.Get(ctx, ws.ID, id); errors.Is(err, repository.ErrNotFound) {
		return nil, ErrKnowledgeNotFound
	} else if err != nil {
		return nil, err
	}
	if req.Name != nil {
		name := clipName(*req.Name, 255)
		if name == "" {
			return nil, ErrInvalidInput
		}
		if err := s.kb.Rename(ctx, ws.ID, id, name); err != nil {
			return nil, err
		}
	}
	if req.FileIDs != nil {
		if err := s.attach(ctx, userID, sessionID, ws.ID, id, uniqueIDs(req.FileIDs)); err != nil {
			return nil, err
		}
	}
	k, err := s.kb.Get(ctx, ws.ID, id)
	if err != nil {
		return nil, err
	}
	return s.withFiles(ctx, k)
}

func (s *KnowledgeService) attach(ctx context.Context, userID, sessionID, workspaceID, kbID string, fileIDs []string) error {
	var okIDs []string
	for _, id := range fileIDs {
		f, err := s.disk.GetFile(ctx, userID, sessionID, id)
		if err != nil {
			return err
		}
		if f.WorkspaceID != workspaceID {
			return ErrDiskFileNotFound
		}
		if !filesniff.Ingestible(f.Name) {
			continue
		}
		okIDs = append(okIDs, f.ID)
	}
	return s.kb.AttachFiles(ctx, kbID, okIDs)
}

func (s *KnowledgeService) Vectorize(ctx context.Context, userID, sessionID, id string) (*model.VectorizeResult, error) {
	k, err := s.Get(ctx, userID, sessionID, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.kb.RunningJobForKB(ctx, k.ID); err == nil {
		return nil, ErrVectorizeBusy
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	uid := userID
	job, err := s.kb.EnqueueJob(ctx, &model.VectorizeJob{
		WorkspaceID:     k.WorkspaceID,
		KnowledgeBaseID: k.ID,
		CreatedByUserID: &uid,
		Total:           k.FileCount,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, userID, "knowledge.vectorize", k.ID)
	return &model.VectorizeResult{KnowledgeBase: k, JobID: job.ID}, nil
}

func (s *KnowledgeService) Search(ctx context.Context, userID, sessionID, id string, req model.KnowledgeSearchRequest) ([]model.KnowledgeHit, error) {
	q := strings.TrimSpace(req.Query)
	if utf8CountRunes(q) < 2 {
		return nil, ErrInvalidInput
	}
	k, err := s.Get(ctx, userID, sessionID, id)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	th := req.Threshold
	if th <= 0 {
		th = 0.3
	}
	vecs, err := s.gw.Embed(ctx, []string{q}, "query")
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, ErrGatewayUnavailable
	}
	return s.kb.SearchVector(ctx, k.WorkspaceID, k.ID, vecs[0], limit, th)
}

func uniqueIDs(ids []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func utf8CountRunes(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
