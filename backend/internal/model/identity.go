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
	PasswordHash         string
	TotpSecretEncrypted  string
	TotpEnabledAt        *time.Time
}

type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	OwnerID   string    `json:"owner_id"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Invite struct {
	ID            string     `json:"id"`
	CodePrefix    string     `json:"code_prefix"`
	Status        string     `json:"status"`
	CreatedBy     *string    `json:"created_by_user_id,omitempty"`
	UsedBy        *string    `json:"used_by_user_id,omitempty"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type IssuedInvite struct {
	Invite
	Code string `json:"code"`
}
