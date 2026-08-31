package model

import "time"

type StorageSettings struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	UseSSL    bool   `json:"use_ssl"`
	PathStyle bool   `json:"path_style"`
	Enabled   bool   `json:"enabled"`
}

type StorageSettingsRecord struct {
	Config    StorageSettings `json:"config"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type StorageAdminView struct {
	Endpoint      string    `json:"endpoint"`
	Bucket        string    `json:"bucket"`
	Region        string    `json:"region"`
	AccessKey     string    `json:"access_key"`
	SecretKeySet  bool      `json:"secret_key_set"`
	SecretKeyHint string    `json:"secret_key_hint,omitempty"`
	UseSSL        bool      `json:"use_ssl"`
	PathStyle     bool      `json:"path_style"`
	Enabled       bool      `json:"enabled"`
	CORSOrigins   []string  `json:"cors_origins"`
	CORSXML       string    `json:"cors_xml"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type StorageAdminUpdateRequest struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key,omitempty"`
	UseSSL    bool   `json:"use_ssl"`
	PathStyle bool   `json:"path_style"`
	Enabled   bool   `json:"enabled"`
}

type StorageTestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func DefaultStorageSettings() StorageSettings {
	return StorageSettings{
		Region:    "ru-central1",
		UseSSL:    true,
		PathStyle: true,
	}
}
