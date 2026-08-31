package model

import "time"

type User struct {
	ID                    string     `json:"id"`
	Email                 string     `json:"email"`
	Name                  string     `json:"name"`
	Locale                string     `json:"locale"`
	Timezone              string     `json:"timezone"`
	IsPlatformAdmin       bool       `json:"is_platform_admin"`
	IsBlocked             bool       `json:"is_blocked"`
	EmailVerifiedAt       *time.Time `json:"email_verified_at,omitempty"`
	TotpEnabled           bool       `json:"totp_enabled"`
	RegisteredViaInviteID *string    `json:"registered_via_invite_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type UserRecord struct {
	User
	PasswordHash        string
	TotpSecretEncrypted string
	TotpEnabledAt       *time.Time
}

type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	OwnerID   string    `json:"owner_id"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkspaceBitrixLink struct {
	WorkspaceID        string `json:"workspace_id"`
	WorkspaceName      string `json:"workspace_name,omitempty"`
	BitrixDepartmentID int64  `json:"bitrix_department_id"`
	DepartmentName     string `json:"department_name,omitempty"`
	IncludeDescendants bool   `json:"include_descendants"`
}

type WorkspaceAdmin struct {
	Workspace
	MemberCount       int                   `json:"member_count"`
	BitrixDepartments []WorkspaceBitrixLink `json:"bitrix_departments"`
}

type MembershipApplyResult struct {
	Workspaces int `json:"workspaces"`
	Matched    int `json:"matched"`
	Added      int `json:"added"`
	Updated    int `json:"updated"`
	Removed    int `json:"removed"`
	Unmatched  int `json:"unmatched"`
}

type Invite struct {
	ID            string     `json:"id"`
	CodePrefix    string     `json:"code_prefix"`
	Code          string     `json:"code,omitempty"`
	Status        string     `json:"status"`
	CreatedBy     *string    `json:"created_by_user_id,omitempty"`
	UsedBy        *string    `json:"used_by_user_id,omitempty"`
	UsedByEmail   *string    `json:"used_by_email,omitempty"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CodeEncrypted string     `json:"-"`
}

type IssuedInvite struct {
	Invite
	Code string `json:"code"`
}
