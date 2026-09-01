package model

import "time"

const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
	MediaKindImage    = "image"
	MediaKindVideo    = "video"
)

type ChatThread struct {
	ID             string          `json:"id"`
	WorkspaceID    string          `json:"workspace_id"`
	UserID         string          `json:"user_id"`
	ModelID        *string         `json:"model_id,omitempty"`
	Title          string          `json:"title"`
	WebSearch      bool            `json:"web_search"`
	CollectionIDs  []string        `json:"collection_ids"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Model          *CatalogModel   `json:"model,omitempty"`
}

type ChatMessage struct {
	ID        string           `json:"id"`
	ThreadID  string           `json:"thread_id"`
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	Citations []ChatCitation   `json:"citations,omitempty"`
	MediaURL  *string          `json:"media_url,omitempty"`
	MediaKind *string          `json:"media_kind,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

type ChatCitation struct {
	Source     string  `json:"source"`
	FileName   string  `json:"file_name,omitempty"`
	Content    string  `json:"content,omitempty"`
	Similarity float64 `json:"similarity,omitempty"`
	URL        string  `json:"url,omitempty"`
}

type CatalogModel struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Purpose     string `json:"purpose"`
}

type ChatThreadCreateRequest struct {
	ModelID       string   `json:"model_id"`
	CollectionIDs []string `json:"collection_ids"`
	WebSearch     bool     `json:"web_search"`
}

type ChatThreadPatchRequest struct {
	ModelID       *string  `json:"model_id"`
	CollectionIDs []string `json:"collection_ids"`
	WebSearch     *bool    `json:"web_search"`
	Title         *string  `json:"title"`
}

type ChatMessageCreateRequest struct {
	Content string `json:"content"`
}

type ChatMediaRequest struct {
	Prompt string `json:"prompt"`
	Kind   string `json:"kind"`
}

type ChatStreamEvent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Message   *ChatMessage    `json:"message,omitempty"`
	Citations []ChatCitation  `json:"citations,omitempty"`
}
