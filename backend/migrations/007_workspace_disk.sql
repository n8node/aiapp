-- +goose Up
CREATE TABLE workspace_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES workspace_folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ,
    trash_batch_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspace_folders_ws_parent ON workspace_folders (workspace_id, parent_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_folders_ws_deleted ON workspace_folders (workspace_id, deleted_at);

CREATE TABLE workspace_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES workspace_folders(id) ON DELETE SET NULL,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    size BIGINT NOT NULL CHECK (size > 0),
    s3_key TEXT NOT NULL UNIQUE,
    deleted_at TIMESTAMPTZ,
    trash_batch_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspace_files_ws_folder ON workspace_files (workspace_id, folder_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_files_ws_deleted ON workspace_files (workspace_id, deleted_at);
CREATE INDEX idx_workspace_files_ws_mime ON workspace_files (workspace_id, mime_type, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS workspace_files;
DROP TABLE IF EXISTS workspace_folders;
