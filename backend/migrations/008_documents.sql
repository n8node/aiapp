-- +goose Up
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    current_version_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_workspace ON documents (workspace_id, created_at DESC);

CREATE TABLE document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    version_n INT NOT NULL,
    disk_file_id UUID REFERENCES workspace_files(id) ON DELETE SET NULL,
    s3_key TEXT NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL CHECK (size > 0),
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    engine VARCHAR(64),
    page_count INT,
    char_count INT,
    confidence DOUBLE PRECISION,
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    extracted_text TEXT,
    error_code VARCHAR(64),
    reviewed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (document_id, version_n)
);

CREATE INDEX idx_document_versions_doc ON document_versions (document_id, version_n DESC);
CREATE INDEX idx_document_versions_file ON document_versions (workspace_id, disk_file_id)
    WHERE disk_file_id IS NOT NULL;
CREATE INDEX idx_document_versions_hash ON document_versions (workspace_id, content_hash)
    WHERE content_hash <> '';

CREATE TABLE ingest_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    document_version_id UUID NOT NULL REFERENCES document_versions(id) ON DELETE CASCADE,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    lease_until TIMESTAMPTZ,
    lease_owner VARCHAR(64),
    last_error VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ingest_jobs_claim ON ingest_jobs (status, lease_until, created_at);

-- +goose Down
DROP TABLE IF EXISTS ingest_jobs;
DROP TABLE IF EXISTS document_versions;
DROP TABLE IF EXISTS documents;
