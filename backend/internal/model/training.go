package model

import "time"

const (
	TrainingDraft     = "draft"
	TrainingSubmitted = "submitted"
	TrainingApproved  = "approved"
	TrainingRejected  = "rejected"
	TrainingCancelled = "cancelled"

	TrainingPurposeBehavior       = "behavior"
	TrainingPurposeFormat         = "format"
	TrainingPurposeExtraction     = "extraction"
	TrainingPurposeClassification = "classification"
)

type TrainingRequest struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspace_id"`
	RequestedByUserID string     `json:"requested_by_user_id"`
	RequesterEmail    string     `json:"requester_email,omitempty"`
	Title             string     `json:"title"`
	Purpose           string     `json:"purpose"`
	DatasetNote       string     `json:"dataset_note"`
	BaseModel         string     `json:"base_model"`
	Status            string     `json:"status"`
	ReviewNote        string     `json:"review_note,omitempty"`
	ReviewedByUserID  *string    `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type TrainingRequestCreate struct {
	Title       string `json:"title"`
	Purpose     string `json:"purpose"`
	DatasetNote string `json:"dataset_note"`
	BaseModel   string `json:"base_model"`
}

type TrainingRequestAction struct {
	Note string `json:"note"`
}
