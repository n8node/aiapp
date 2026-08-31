-- +goose Up
CREATE TABLE bitrix_departments (
    bitrix_id BIGINT PRIMARY KEY,
    parent_bitrix_id BIGINT,
    name TEXT NOT NULL,
    sort INT NOT NULL DEFAULT 0,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bitrix_departments_parent ON bitrix_departments (parent_bitrix_id);

CREATE TABLE bitrix_users (
    bitrix_id BIGINT PRIMARY KEY,
    email TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT true,
    department_ids BIGINT[] NOT NULL DEFAULT '{}',
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bitrix_users_email ON bitrix_users (email);

CREATE TABLE bitrix_sync_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(16) NOT NULL,
    error_public TEXT NOT NULL DEFAULT '',
    departments_count INT NOT NULL DEFAULT 0,
    users_count INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_bitrix_sync_runs_started_at ON bitrix_sync_runs (started_at DESC);

-- +goose Down
DROP TABLE IF EXISTS bitrix_sync_runs;
DROP TABLE IF EXISTS bitrix_users;
DROP TABLE IF EXISTS bitrix_departments;
