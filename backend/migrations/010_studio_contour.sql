-- +goose Up
ALTER TABLE users
    ADD COLUMN studio_access BOOLEAN NOT NULL DEFAULT false;

INSERT INTO app_settings (key, value)
VALUES ('ui.locale', 'ru')
ON CONFLICT (key) DO NOTHING;

CREATE TABLE training_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    requested_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    purpose VARCHAR(32) NOT NULL DEFAULT 'behavior',
    dataset_note VARCHAR(2000) NOT NULL DEFAULT '',
    base_model VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    review_note VARCHAR(1000) NOT NULL DEFAULT '',
    reviewed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_training_requests_workspace ON training_requests (workspace_id, created_at DESC);
CREATE INDEX idx_training_requests_status ON training_requests (status, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS training_requests;
DELETE FROM app_settings WHERE key = 'ui.locale';
ALTER TABLE users DROP COLUMN IF EXISTS studio_access;
