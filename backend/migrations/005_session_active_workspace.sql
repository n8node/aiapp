-- +goose Up
ALTER TABLE sessions
    ADD COLUMN active_workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX idx_sessions_active_workspace_id ON sessions (active_workspace_id)
    WHERE active_workspace_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_active_workspace_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS active_workspace_id;
