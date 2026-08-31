-- +goose Up
ALTER TABLE bitrix_departments
    ADD COLUMN head_bitrix_id BIGINT;

ALTER TABLE workspace_members
    ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT 'local';

CREATE TABLE workspace_bitrix_departments (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    bitrix_department_id BIGINT NOT NULL,
    include_descendants BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, bitrix_department_id),
    UNIQUE (bitrix_department_id)
);

CREATE INDEX idx_workspace_bitrix_departments_workspace
    ON workspace_bitrix_departments (workspace_id);

-- +goose Down
DROP TABLE IF EXISTS workspace_bitrix_departments;
ALTER TABLE workspace_members DROP COLUMN IF EXISTS source;
ALTER TABLE bitrix_departments DROP COLUMN IF EXISTS head_bitrix_id;
