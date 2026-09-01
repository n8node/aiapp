package service

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

var (
	ErrModelNotFound      = errors.New("model not found")
	ErrModelInvalidState  = errors.New("model invalid state")
	ErrModelSlugTaken     = errors.New("model slug taken")
	ErrGatewayUnavailable = errors.New("gateway unavailable")
	ErrKnowledgeNotFound  = errors.New("knowledge base not found")
	ErrVectorizeBusy      = errors.New("vectorize busy")
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

type ModelService struct {
	repo   *repository.MLModelRepository
	audit  *repository.AuditRepository
	studio string
	gw     *GatewayClient
	http   *http.Client
}

func NewModelService(repo *repository.MLModelRepository, audit *repository.AuditRepository, studioURL string, gw *GatewayClient) *ModelService {
	return &ModelService{
		repo:   repo,
		audit:  audit,
		studio: strings.TrimRight(strings.TrimSpace(studioURL), "/"),
		gw:     gw,
		http:   &http.Client{Timeout: 3 * time.Second},
	}
}

func validPurpose(p string) bool {
	switch p {
	case model.ModelPurposeEmbeddings, model.ModelPurposeChat, model.ModelPurposeOCR, model.ModelPurposeRerank:
		return true
	default:
		return false
	}
}

func validSource(s string) bool {
	switch s {
	case model.ModelSourceHuggingFace, model.ModelSourceUpload, model.ModelSourceStudio:
		return true
	default:
		return false
	}
}

func (s *ModelService) List(ctx context.Context) ([]model.MLModel, error) {
	return s.repo.List(ctx)
}

func (s *ModelService) Create(ctx context.Context, actorID string, req model.MLModelCreateRequest) (*model.MLModel, error) {
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	name := clipName(req.DisplayName, 255)
	purpose := strings.TrimSpace(req.Purpose)
	source := strings.TrimSpace(req.SourceType)
	ref := strings.TrimSpace(req.SourceRef)
	if !slugRe.MatchString(slug) || name == "" || !validPurpose(purpose) || !validSource(source) || ref == "" {
		return nil, ErrInvalidInput
	}
	if source == model.ModelSourceHuggingFace && (strings.Contains(ref, "..") || strings.HasPrefix(ref, "/")) {
		return nil, ErrInvalidInput
	}
	if _, err := s.repo.GetBySlug(ctx, slug); err == nil {
		return nil, ErrModelSlugTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	uid := actorID
	m, err := s.repo.Create(ctx, &model.MLModel{
		Slug:            slug,
		DisplayName:     name,
		Purpose:         purpose,
		SourceType:      source,
		SourceRef:       ref,
		License:         req.License,
		Architecture:    req.Architecture,
		Quantization:    req.Quantization,
		Dimensions:      req.Dimensions,
		ContextLength:   req.ContextLength,
		GPUDevice:       req.GPUDevice,
		Notes:           req.Notes,
		CreatedByUserID: &uid,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, actorID, "model.register", m.ID)
	return m, nil
}

func (s *ModelService) Action(ctx context.Context, actorID, id, action string) (*model.MLModel, error) {
	m, err := s.repo.Get(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrModelNotFound
	}
	if err != nil {
		return nil, err
	}
	switch strings.TrimSpace(action) {
	case "approve":
		if m.Status != model.ModelStatusQuarantine {
			return nil, ErrModelInvalidState
		}
		if err := s.repo.SetStatus(ctx, id, model.ModelStatusQuarantine, model.ModelStatusApproved, actorID); err != nil {
			return nil, err
		}
		s.audit.Write(ctx, actorID, "model.approve", id)
	case "reject":
		if m.Status != model.ModelStatusQuarantine && m.Status != model.ModelStatusApproved {
			return nil, ErrModelInvalidState
		}
		if err := s.repo.SetStatus(ctx, id, m.Status, model.ModelStatusRejected, actorID); err != nil {
			return nil, err
		}
		s.audit.Write(ctx, actorID, "model.reject", id)
	case "deploy":
		if m.Status != model.ModelStatusApproved && m.Status != model.ModelStatusDeployed {
			return nil, ErrModelInvalidState
		}
		if err := s.repo.UndeployPurpose(ctx, m.Purpose, id); err != nil {
			return nil, err
		}
		alias := m.Purpose
		if err := s.repo.SetDeployed(ctx, id, alias); err != nil {
			return nil, err
		}
		s.audit.Write(ctx, actorID, "model.deploy", id)
	case "disable":
		if m.Status != model.ModelStatusDeployed {
			return nil, ErrModelInvalidState
		}
		if err := s.repo.SetStatus(ctx, id, model.ModelStatusDeployed, model.ModelStatusDisabled, actorID); err != nil {
			return nil, err
		}
		s.audit.Write(ctx, actorID, "model.disable", id)
	default:
		return nil, ErrInvalidInput
	}
	return s.repo.Get(ctx, id)
}

func (s *ModelService) StudioStatus(ctx context.Context) model.StudioStatus {
	st := model.StudioStatus{Path: "/hub", Reachable: false}
	if s.studio == "" {
		return st
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.studio+"/", nil)
	if err != nil {
		return st
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return st
	}
	defer resp.Body.Close()
	st.Reachable = resp.StatusCode > 0 && resp.StatusCode < 500
	return st
}

func (s *ModelService) GatewayStatus(ctx context.Context) model.GatewayStatus {
	out := model.GatewayStatus{Model: "intfloat/multilingual-e5-small", Dimensions: 384}
	if s.gw != nil {
		out.Reachable = s.gw.Ready(ctx)
	}
	return out
}
