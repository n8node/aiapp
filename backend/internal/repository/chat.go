package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/model"
)

type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

const threadCols = `id, workspace_id, user_id, model_id, title, web_search, created_at, updated_at`

func scanThread(row pgx.Row) (*model.ChatThread, error) {
	var t model.ChatThread
	err := row.Scan(&t.ID, &t.WorkspaceID, &t.UserID, &t.ModelID, &t.Title, &t.WebSearch, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if t.CollectionIDs == nil {
		t.CollectionIDs = []string{}
	}
	return &t, nil
}

func (r *ChatRepository) CreateThread(ctx context.Context, t *model.ChatThread) (*model.ChatThread, error) {
	out, err := scanThread(r.pool.QueryRow(ctx, `
		INSERT INTO chat_threads (workspace_id, user_id, model_id, title, web_search)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING `+threadCols, t.WorkspaceID, t.UserID, t.ModelID, t.Title, t.WebSearch))
	if err != nil {
		return nil, err
	}
	if err := r.ReplaceCollections(ctx, out.ID, t.CollectionIDs); err != nil {
		return nil, err
	}
	out.CollectionIDs = append([]string{}, t.CollectionIDs...)
	if out.CollectionIDs == nil {
		out.CollectionIDs = []string{}
	}
	return out, nil
}

func (r *ChatRepository) GetThread(ctx context.Context, workspaceID, userID, id string) (*model.ChatThread, error) {
	t, err := scanThread(r.pool.QueryRow(ctx, `
		SELECT `+threadCols+` FROM chat_threads WHERE id = $1 AND workspace_id = $2 AND user_id = $3`, id, workspaceID, userID))
	if err != nil {
		return nil, err
	}
	ids, err := r.collectionIDs(ctx, []string{t.ID})
	if err != nil {
		return nil, err
	}
	t.CollectionIDs = ids[t.ID]
	if t.CollectionIDs == nil {
		t.CollectionIDs = []string{}
	}
	return t, nil
}

func (r *ChatRepository) ListThreads(ctx context.Context, workspaceID, userID string) ([]model.ChatThread, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+threadCols+` FROM chat_threads
		WHERE workspace_id = $1 AND user_id = $2
		ORDER BY updated_at DESC
		LIMIT 100`, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ChatThread
	var ids []string
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	coll, err := r.collectionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].CollectionIDs = coll[out[i].ID]
		if out[i].CollectionIDs == nil {
			out[i].CollectionIDs = []string{}
		}
	}
	if out == nil {
		out = []model.ChatThread{}
	}
	return out, nil
}

func (r *ChatRepository) collectionIDs(ctx context.Context, threadIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(threadIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT thread_id, knowledge_base_id FROM chat_thread_collections
		WHERE thread_id::text = ANY($1)`, threadIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tid, kid string
		if err := rows.Scan(&tid, &kid); err != nil {
			return nil, err
		}
		out[tid] = append(out[tid], kid)
	}
	return out, rows.Err()
}

func (r *ChatRepository) ReplaceCollections(ctx context.Context, threadID string, kbIDs []string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM chat_thread_collections WHERE thread_id = $1`, threadID); err != nil {
		return err
	}
	for _, id := range kbIDs {
		if id == "" {
			continue
		}
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO chat_thread_collections (thread_id, knowledge_base_id) VALUES ($1,$2)
			ON CONFLICT DO NOTHING`, threadID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *ChatRepository) UpdateThread(ctx context.Context, t *model.ChatThread) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE chat_threads SET model_id = $4, title = $5, web_search = $6, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND user_id = $3`,
		t.ID, t.WorkspaceID, t.UserID, t.ModelID, t.Title, t.WebSearch)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return r.ReplaceCollections(ctx, t.ID, t.CollectionIDs)
}

func (r *ChatRepository) TouchThread(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE chat_threads SET updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *ChatRepository) DeleteThread(ctx context.Context, workspaceID, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM chat_threads WHERE id = $1 AND workspace_id = $2 AND user_id = $3`, id, workspaceID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ChatRepository) InsertMessage(ctx context.Context, m *model.ChatMessage) (*model.ChatMessage, error) {
	cites, err := json.Marshal(m.Citations)
	if err != nil {
		cites = []byte("[]")
	}
	if m.Citations == nil {
		cites = []byte("[]")
	}
	var out model.ChatMessage
	var raw []byte
	err = r.pool.QueryRow(ctx, `
		INSERT INTO chat_messages (thread_id, role, content, citations, media_url, media_kind)
		VALUES ($1,$2,$3,$4::jsonb,$5,$6)
		RETURNING id, thread_id, role, content, citations, media_url, media_kind, created_at`,
		m.ThreadID, m.Role, m.Content, cites, m.MediaURL, m.MediaKind).Scan(
		&out.ID, &out.ThreadID, &out.Role, &out.Content, &raw, &out.MediaURL, &out.MediaKind, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(raw, &out.Citations)
	if out.Citations == nil {
		out.Citations = []model.ChatCitation{}
	}
	return &out, nil
}

func (r *ChatRepository) ListMessages(ctx context.Context, threadID string) ([]model.ChatMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, thread_id, role, content, citations, media_url, media_kind, created_at
		FROM chat_messages WHERE thread_id = $1
		ORDER BY created_at ASC
		LIMIT 200`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ChatMessage
	for rows.Next() {
		var m model.ChatMessage
		var raw []byte
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &raw, &m.MediaURL, &m.MediaKind, &m.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &m.Citations)
		if m.Citations == nil {
			m.Citations = []model.ChatCitation{}
		}
		out = append(out, m)
	}
	if out == nil {
		out = []model.ChatMessage{}
	}
	return out, rows.Err()
}
