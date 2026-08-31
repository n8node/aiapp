-- +goose Up
CREATE TABLE storage_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    config JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO storage_settings (id, config) VALUES (1, '{
  "endpoint": "",
  "bucket": "",
  "region": "ru-central1",
  "access_key": "",
  "secret_key": "",
  "use_ssl": true,
  "path_style": true,
  "enabled": false
}'::jsonb);

-- +goose Down
DROP TABLE IF EXISTS storage_settings;
