-- +goose Up
ALTER TABLE invites
    ADD COLUMN IF NOT EXISTS code_encrypted TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_invites_created_at ON invites (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invites_used_at ON invites (used_at DESC) WHERE used_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_invites_used_at;
DROP INDEX IF EXISTS idx_invites_created_at;
ALTER TABLE invites DROP COLUMN IF EXISTS code_encrypted;
