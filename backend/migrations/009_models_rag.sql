-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE ml_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    source_type VARCHAR(32) NOT NULL,
    source_ref TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    artifact_hash VARCHAR(64) NOT NULL DEFAULT '',
    license VARCHAR(128),
    architecture VARCHAR(128),
    quantization VARCHAR(64),
    dimensions INT,
    context_length INT,
    gpu_device INT,
    gateway_alias VARCHAR(64),
    notes VARCHAR(500),
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ml_models_purpose_status ON ml_models (purpose, status);

INSERT INTO ml_models (slug, display_name, purpose, source_type, source_ref, status, license, architecture, dimensions, gateway_alias, notes)
VALUES (
    'e5-small',
    'Multilingual E5 small',
    'embeddings',
    'huggingface',
    'intfloat/multilingual-e5-small',
    'deployed',
    'MIT',
    'e5',
    384,
    'embeddings',
    'Default embedding model served by the internal gateway on CPU so GPU stays free for chat and Studio.'
);

CREATE TABLE knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    chunk_size INT NOT NULL DEFAULT 500,
    chunk_overlap INT NOT NULL DEFAULT 50,
    embedding_model_id UUID REFERENCES ml_models(id) ON DELETE SET NULL,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_knowledge_bases_workspace ON knowledge_bases (workspace_id, created_at DESC);

CREATE TABLE knowledge_base_files (
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES workspace_files(id) ON DELETE CASCADE,
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    chunk_count INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    error_code VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (knowledge_base_id, file_id)
);

CREATE TABLE knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    file_id UUID REFERENCES workspace_files(id) ON DELETE SET NULL,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    embedding vector(384),
    token_count INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_knowledge_chunks_kb ON knowledge_chunks (workspace_id, knowledge_base_id);
CREATE INDEX idx_knowledge_chunks_file ON knowledge_chunks (knowledge_base_id, file_id);
CREATE INDEX idx_knowledge_chunks_embedding ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);

CREATE TABLE vectorize_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    lease_until TIMESTAMPTZ,
    lease_owner VARCHAR(64),
    last_error VARCHAR(64),
    processed INT NOT NULL DEFAULT 0,
    total INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vectorize_jobs_claim ON vectorize_jobs (status, lease_until, created_at);

-- +goose Down
DROP TABLE IF EXISTS vectorize_jobs;
DROP TABLE IF EXISTS knowledge_chunks;
DROP TABLE IF EXISTS knowledge_base_files;
DROP TABLE IF EXISTS knowledge_bases;
DROP TABLE IF EXISTS ml_models;
DROP EXTENSION IF EXISTS vector;
