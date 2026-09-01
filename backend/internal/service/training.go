package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

var (
	ErrTrainingNotFound     = errors.New("training not found")
	ErrTrainingInvalidState = errors.New("training invalid state")
)

type TrainingService struct {
	repo  *repository.TrainingRepository
	auth  *AuthService
	audit *repository.AuditRepository
}

func NewTrainingService(repo *repository.TrainingRepository, auth *AuthService, audit *repository.AuditRepository) *TrainingService {
	return &TrainingService{repo: repo, auth: auth, audit: audit}
}

func (s *TrainingService) resolve(ctx context.Context, userID, sessionID string) (*model.User, *model.Workspace, error) {
	user, _, active, err := s.auth.Me(ctx, userID, sessionID)
	if err != nil {
		return nil, nil, err
	}
	if active == nil {
		return nil, nil, ErrDiskNoWorkspace
	}
	return user, active, nil
}

func (s *TrainingService) Create(ctx context.Context, userID, sessionID string, req model.TrainingRequestCreate) (*model.TrainingRequest, error) {
	user, ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" || utf8.RuneCountInString(title) > 255 {
		return nil, ErrInvalidInput
	}
	purpose := strings.TrimSpace(req.Purpose)
	if purpose == "" {
		purpose = model.TrainingPurposeBehavior
	}
	if !ValidTrainingPurpose(purpose) {
		return nil, ErrInvalidInput
	}
	note := strings.TrimSpace(req.DatasetNote)
	if utf8.RuneCountInString(note) > 2000 {
		return nil, ErrInvalidInput
	}
	base := strings.TrimSpace(req.BaseModel)
	if utf8.RuneCountInString(base) > 255 {
		return nil, ErrInvalidInput
	}
	created, err := s.repo.Create(ctx, &model.TrainingRequest{
		WorkspaceID:       ws.ID,
		RequestedByUserID: user.ID,
		Title:             title,
		Purpose:           purpose,
		DatasetNote:       note,
		BaseModel:         base,
		Status:            model.TrainingDraft,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, userID, "training.create", created.ID)
	return created, nil
}

func (s *TrainingService) List(ctx context.Context, userID, sessionID string) ([]model.TrainingRequest, int, error) {
	_, ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListWorkspace(ctx, ws.ID, 50, 0)
}

func (s *TrainingService) Get(ctx context.Context, userID, sessionID, id string) (*model.TrainingRequest, error) {
	user, ws, err := s.resolve(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTrainingNotFound
		}
		return nil, err
	}
	if !user.IsPlatformAdmin && item.WorkspaceID != ws.ID {
		return nil, ErrForbidden
	}
	return item, nil
}

func (s *TrainingService) Submit(ctx context.Context, userID, sessionID, id string) (*model.TrainingRequest, error) {
	return s.transition(ctx, userID, sessionID, id, "submit", "", false)
}

func (s *TrainingService) Cancel(ctx context.Context, userID, sessionID, id string) (*model.TrainingRequest, error) {
	return s.transition(ctx, userID, sessionID, id, "cancel", "", false)
}

func (s *TrainingService) AdminList(ctx context.Context, status string) ([]model.TrainingRequest, int, error) {
	return s.repo.ListAll(ctx, status, 100, 0)
}

func (s *TrainingService) Approve(ctx context.Context, actorID, id, note string) (*model.TrainingRequest, error) {
	return s.adminTransition(ctx, actorID, id, "approve", note)
}

func (s *TrainingService) Reject(ctx context.Context, actorID, id, note string) (*model.TrainingRequest, error) {
	return s.adminTransition(ctx, actorID, id, "reject", note)
}

func (s *TrainingService) transition(ctx context.Context, userID, sessionID, id, action, note string, admin bool) (*model.TrainingRequest, error) {
	item, err := s.Get(ctx, userID, sessionID, id)
	if err != nil {
		return nil, err
	}
	if !admin && item.RequestedByUserID != userID {
		user, err := s.auth.RequireUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !user.IsPlatformAdmin {
			return nil, ErrForbidden
		}
	}
	next, err := NextTrainingStatus(item.Status, action)
	if err != nil {
		return nil, err
	}
	reviewer := (*string)(nil)
	reviewNote := item.ReviewNote
	if admin {
		reviewer = &userID
		if strings.TrimSpace(note) != "" {
			reviewNote = strings.TrimSpace(note)
			if utf8.RuneCountInString(reviewNote) > 1000 {
				return nil, ErrInvalidInput
			}
		}
	}
	if err := s.repo.UpdateStatus(ctx, id, next, reviewNote, reviewer); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, userID, "training."+action, id)
	return s.repo.Get(ctx, id)
}

func (s *TrainingService) adminTransition(ctx context.Context, actorID, id, action, note string) (*model.TrainingRequest, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTrainingNotFound
		}
		return nil, err
	}
	next, err := NextTrainingStatus(item.Status, action)
	if err != nil {
		return nil, err
	}
	reviewNote := item.ReviewNote
	if strings.TrimSpace(note) != "" {
		reviewNote = strings.TrimSpace(note)
		if utf8.RuneCountInString(reviewNote) > 1000 {
			return nil, ErrInvalidInput
		}
	}
	if err := s.repo.UpdateStatus(ctx, id, next, reviewNote, &actorID); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, actorID, "training."+action, id)
	return s.repo.Get(ctx, id)
}

func NextTrainingStatus(current, action string) (string, error) {
	switch action {
	case "submit":
		if current == model.TrainingDraft {
			return model.TrainingSubmitted, nil
		}
	case "cancel":
		if current == model.TrainingDraft || current == model.TrainingSubmitted {
			return model.TrainingCancelled, nil
		}
	case "approve":
		if current == model.TrainingSubmitted {
			return model.TrainingApproved, nil
		}
	case "reject":
		if current == model.TrainingSubmitted {
			return model.TrainingRejected, nil
		}
	}
	return "", ErrTrainingInvalidState
}
