package model

import "time"

type BitrixStatus struct {
	Configured       bool       `json:"configured"`
	PortalHost       string     `json:"portal_host,omitempty"`
	WebhookMasked    string     `json:"webhook_masked,omitempty"`
	LastSyncAt       *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus   string     `json:"last_sync_status,omitempty"`
	LastSyncError    string     `json:"last_sync_error,omitempty"`
	DepartmentsCount int        `json:"departments_count"`
	UsersCount       int        `json:"users_count"`
}

type BitrixDepartment struct {
	BitrixID           int64   `json:"bitrix_id"`
	ParentBitrixID     *int64  `json:"parent_bitrix_id,omitempty"`
	Name               string  `json:"name"`
	Sort               int     `json:"sort"`
	HeadBitrixID       *int64  `json:"head_bitrix_id,omitempty"`
	WorkspaceID        *string `json:"workspace_id,omitempty"`
	WorkspaceName      *string `json:"workspace_name,omitempty"`
	WorkspaceInherited bool    `json:"workspace_inherited"`
	InheritedFrom      string  `json:"inherited_from,omitempty"`
	IncludeDescendants *bool   `json:"include_descendants,omitempty"`
}

type BitrixUser struct {
	BitrixID      int64   `json:"bitrix_id"`
	Email         string  `json:"email"`
	Name          string  `json:"name"`
	LastName      string  `json:"last_name"`
	Active        bool    `json:"active"`
	DepartmentIDs []int64 `json:"department_ids"`
}

type BitrixSyncResult struct {
	Departments int `json:"departments"`
	Users       int `json:"users"`
}
