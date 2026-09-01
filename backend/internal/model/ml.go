package model

import "time"

const (
	ModelPurposeEmbeddings = "embeddings"
	ModelPurposeChat       = "chat"
	ModelPurposeOCR        = "ocr"
	ModelPurposeRerank     = "rerank"
	ModelPurposeImage      = "image"
	ModelPurposeVideo      = "video"

	ModelSourceHuggingFace = "huggingface"
	ModelSourceUpload      = "upload"
	ModelSourceStudio      = "studio_export"

	ModelStatusQuarantine = "quarantine"
	ModelStatusApproved   = "approved"
	ModelStatusRejected   = "rejected"
	ModelStatusDeployed   = "deployed"
	ModelStatusDisabled   = "disabled"

	KBFilePending = "pending"
	KBFileIndexed = "indexed"
	KBFileFailed  = "failed"
	KBFileSkipped = "skipped"

	VectorizeQueued    = "queued"
	VectorizeRunning   = "running"
	VectorizeCompleted = "completed"
	VectorizeFailed    = "failed"
)

type MLModel struct {
	ID               string     `json:"id"`
	Slug             string     `json:"slug"`
	DisplayName      string     `json:"display_name"`
	Purpose          string     `json:"purpose"`
	SourceType       string     `json:"source_type"`
	SourceRef        string     `json:"source_ref"`
	Status           string     `json:"status"`
	ArtifactHash     string     `json:"artifact_hash,omitempty"`
	License          *string    `json:"license,omitempty"`
	Architecture     *string    `json:"architecture,omitempty"`
	Quantization     *string    `json:"quantization,omitempty"`
	Dimensions       *int       `json:"dimensions,omitempty"`
	ContextLength    *int       `json:"context_length,omitempty"`
	GPUDevice        *int       `json:"gpu_device,omitempty"`
	GatewayAlias     *string    `json:"gateway_alias,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedByUserID  *string    `json:"created_by_user_id,omitempty"`
	ApprovedByUserID *string    `json:"approved_by_user_id,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type MLModelCreateRequest struct {
	Slug          string  `json:"slug"`
	DisplayName   string  `json:"display_name"`
	Purpose       string  `json:"purpose"`
	SourceType    string  `json:"source_type"`
	SourceRef     string  `json:"source_ref"`
	License       *string `json:"license"`
	Architecture  *string `json:"architecture"`
	Quantization  *string `json:"quantization"`
	Dimensions    *int    `json:"dimensions"`
	ContextLength *int    `json:"context_length"`
	GPUDevice     *int    `json:"gpu_device"`
	Notes         *string `json:"notes"`
}

type MLModelActionRequest struct {
	Action string `json:"action"`
}

type StudioStatus struct {
	Reachable bool   `json:"reachable"`
	Path      string `json:"path"`
}

type GatewayStatus struct {
	Reachable  bool   `json:"reachable"`
	Model      string `json:"embedding_model"`
	Dimensions int    `json:"dimensions"`
}

type KnowledgeBase struct {
	ID                   string              `json:"id"`
	WorkspaceID          string              `json:"workspace_id"`
	Name                 string              `json:"name"`
	ChunkSize            int                 `json:"chunk_size"`
	ChunkOverlap         int                 `json:"chunk_overlap"`
	FolderID             *string             `json:"folder_id,omitempty"`
	SimilarityThreshold  float64             `json:"similarity_threshold"`
	TopK                 int                 `json:"top_k"`
	EmbeddingModelID     *string             `json:"embedding_model_id,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
	FileCount            int                 `json:"file_count"`
	IndexedCount         int                 `json:"indexed_count"`
	Files                []KnowledgeBaseFile `json:"files,omitempty"`
	LatestJob            *VectorizeJob       `json:"latest_job,omitempty"`
}

type KnowledgeBaseFile struct {
	FileID      string  `json:"file_id"`
	Name        string  `json:"name,omitempty"`
	ContentHash string  `json:"content_hash"`
	ChunkCount  int     `json:"chunk_count"`
	Status      string  `json:"status"`
	ErrorCode   *string `json:"error_code,omitempty"`
}

type KnowledgeBaseCreateRequest struct {
	Name     string   `json:"name"`
	FolderID *string  `json:"folder_id"`
	FileIDs  []string `json:"file_ids"`
}

type KnowledgeBasePatchRequest struct {
	Name                *string  `json:"name"`
	FileIDs             []string `json:"file_ids"`
	RemoveFileIDs       []string `json:"remove_file_ids"`
	FolderID            *string  `json:"folder_id"`
	ChunkSize           *int     `json:"chunk_size"`
	ChunkOverlap        *int     `json:"chunk_overlap"`
	SimilarityThreshold *float64 `json:"similarity_threshold"`
	TopK                *int     `json:"top_k"`
	ClearVectors        bool     `json:"clear_vectors"`
}

type VectorizeResult struct {
	KnowledgeBase *KnowledgeBase `json:"knowledge_base"`
	JobID         string         `json:"job_id"`
}

type KnowledgeSearchRequest struct {
	Query     string  `json:"query"`
	Limit     int     `json:"limit"`
	Threshold float64 `json:"threshold"`
}

type KnowledgeHit struct {
	FileID     *string `json:"file_id,omitempty"`
	FileName   string  `json:"file_name,omitempty"`
	ChunkIndex int     `json:"chunk_index"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

type VectorizeJob struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspace_id,omitempty"`
	KnowledgeBaseID string     `json:"knowledge_base_id"`
	CreatedByUserID *string    `json:"-"`
	Status          string     `json:"status"`
	Attempts        int        `json:"-"`
	MaxAttempts     int        `json:"-"`
	LeaseUntil      *time.Time `json:"-"`
	LeaseOwner      *string    `json:"-"`
	LastError       *string    `json:"last_error,omitempty"`
	Processed       int        `json:"processed"`
	Total           int        `json:"total"`
}
