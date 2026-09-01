-- +goose Up
ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS folder_id UUID REFERENCES workspace_folders(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS similarity_threshold DOUBLE PRECISION NOT NULL DEFAULT 0.3,
    ADD COLUMN IF NOT EXISTS top_k INT NOT NULL DEFAULT 8;

CREATE TABLE chat_threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id UUID REFERENCES ml_models(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL DEFAULT 'Новый чат',
    web_search BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_threads_user ON chat_threads (workspace_id, user_id, updated_at DESC);

CREATE TABLE chat_thread_collections (
    thread_id UUID NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    PRIMARY KEY (thread_id, knowledge_base_id)
);

CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
    role VARCHAR(16) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    citations JSONB NOT NULL DEFAULT '[]'::jsonb,
    media_url TEXT,
    media_kind VARCHAR(16),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_messages_thread ON chat_messages (thread_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_thread_collections;
DROP TABLE IF EXISTS chat_threads;
ALTER TABLE knowledge_bases
    DROP COLUMN IF EXISTS folder_id,
    DROP COLUMN IF EXISTS similarity_threshold,
    DROP COLUMN IF EXISTS top_k;
