package model

import "time"

type DiskFile struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspace_id"`
	FolderID        *string    `json:"folder_id"`
	CreatedByUserID *string    `json:"created_by_user_id,omitempty"`
	Name            string     `json:"name"`
	MimeType        string     `json:"mime_type"`
	Size            int64      `json:"size"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	S3Key           string     `json:"-"`
	TrashBatchID    *string    `json:"-"`
}

type DiskFolder struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspace_id"`
	ParentID        *string    `json:"parent_id"`
	Name            string     `json:"name"`
	FilesCount      int        `json:"files_count,omitempty"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedByUserID *string    `json:"-"`
	TrashBatchID    *string    `json:"-"`
}

type FolderBreadcrumb struct {
	ID   *string `json:"id"`
	Name string  `json:"name"`
}

type FileUploadInitRequest struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	MimeType string  `json:"mime_type"`
	FolderID *string `json:"folder_id"`
}

type FileUploadInitResponse struct {
	UploadURL          string            `json:"upload_url"`
	UploadHeaders      map[string]string `json:"upload_headers"`
	UploadSessionToken string            `json:"upload_session_token"`
}

type FileUploadCompleteRequest struct {
	UploadSessionToken string `json:"upload_session_token"`
}

type DiskFilePatchRequest struct {
	Name     *string `json:"name"`
	FolderID *string `json:"folder_id"`
}

type DiskFolderWriteRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

type DiskFolderPatchRequest struct {
	Name     *string `json:"name"`
	ParentID *string `json:"parent_id"`
}

type DiskBulkRequest struct {
	IDs      []string `json:"ids"`
	Action   string   `json:"action"`
	FolderID *string  `json:"folder_id"`
}

type DiskTrashRestoreRequest struct {
	FileIDs   []string `json:"file_ids"`
	FolderIDs []string `json:"folder_ids"`
}

type DiskBulkResult struct {
	OK     int                 `json:"ok"`
	Errors []DiskBulkItemError `json:"errors"`
}

type DiskBulkItemError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type DiskExtractResult struct {
	Folder  DiskFolder `json:"folder"`
	Files   int        `json:"files"`
	Skipped int        `json:"skipped"`
}
