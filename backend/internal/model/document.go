package model

import "time"

const (
	DocStatusQueued         = "queued"
	DocStatusRunning        = "running"
	DocStatusAwaitingReview = "awaiting_review"
	DocStatusPublished      = "published"
	DocStatusFailed         = "failed"
	DocStatusRejected       = "rejected"
	IngestJobQueued         = "queued"
	IngestJobRunning        = "running"
	IngestJobCompleted      = "completed"
	IngestJobFailed         = "failed"
)

type Document struct {
	ID               string           `json:"id"`
	WorkspaceID      string           `json:"workspace_id"`
	Title            string           `json:"title"`
	CreatedByUserID  *string          `json:"created_by_user_id,omitempty"`
	CurrentVersionID *string          `json:"current_version_id,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	CurrentVersion   *DocumentVersion `json:"current_version,omitempty"`
}

type DocumentVersion struct {
	ID               string     `json:"id"`
	DocumentID       string     `json:"document_id"`
	WorkspaceID      string     `json:"workspace_id"`
	VersionN         int        `json:"version_n"`
	DiskFileID       *string    `json:"disk_file_id,omitempty"`
	OriginalName     string     `json:"original_name"`
	MimeType         string     `json:"mime_type"`
	Size             int64      `json:"size"`
	ContentHash      string     `json:"content_hash"`
	Status           string     `json:"status"`
	Engine           *string    `json:"engine,omitempty"`
	PageCount        *int       `json:"page_count,omitempty"`
	CharCount        *int       `json:"char_count,omitempty"`
	Confidence       *float64   `json:"confidence,omitempty"`
	Warnings         []string   `json:"warnings"`
	ExtractedText    *string    `json:"extracted_text,omitempty"`
	ErrorCode        *string    `json:"error_code,omitempty"`
	ReviewedByUserID *string    `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote       *string    `json:"review_note,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	S3Key            string     `json:"-"`
}

type DocumentFromFileRequest struct {
	FileID string `json:"file_id"`
}

type DocumentReviewRequest struct {
	Action string  `json:"action"`
	Note   *string `json:"note"`
}

type DocumentFromFileResult struct {
	Document *Document `json:"document"`
	JobID    *string   `json:"job_id,omitempty"`
	Created  bool      `json:"created"`
}

type IngestJob struct {
	ID                string
	WorkspaceID       string
	DocumentVersionID string
	CreatedByUserID   *string
	Status            string
	Attempts          int
	MaxAttempts       int
	LeaseUntil        *time.Time
	LeaseOwner        *string
	LastError         *string
}

type ExtractResult struct {
	Engine     string   `json:"engine"`
	Text       string   `json:"text"`
	PageCount  int      `json:"page_count"`
	Confidence float64  `json:"confidence"`
	Warnings   []string `json:"warnings"`
}
