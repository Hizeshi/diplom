package chatrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/chat"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ─── Session ─────────────────────────────────────────────────────────────────

// EnsureSession guarantees the session exists and enforces ownership.
//
// trusted=true skips token enforcement (internal/telegram callers).
//
// For anonymous web callers (trusted=false):
//   - New session: created with tokenHash stored.
//   - Existing session owned by same Supabase user: allowed.
//   - Existing session with matching tokenHash: allowed.
//   - Existing session without tokenHash (created before this feature): allowed (migration path).
//   - All other cases: returns chat.ErrSessionForbidden.
//
// Returns (isNew bool, err error).
func (r *Repository) EnsureSession(
	ctx context.Context,
	sessionID, userID, platform string,
	authUserID *string,
	tokenHash string,
	trusted bool,
) (bool, error) {
	// Check whether session already exists.
	var stored struct {
		tokenHash  string
		authUserID *string
	}
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(session_token_hash,''), auth_user_id
		   FROM chat_sessions WHERE session_id = $1`,
		sessionID,
	).Scan(&stored.tokenHash, &stored.authUserID)

	if err != nil && err != pgx.ErrNoRows {
		return false, fmt.Errorf("chatrepo: ensure session lookup: %w", err)
	}

	if err == pgx.ErrNoRows {
		// Create new session.
		_, insertErr := r.db.Exec(ctx,
			`INSERT INTO chat_sessions
			   (session_id, user_id, platform, auth_user_id, session_token_hash, is_human_mode)
			 VALUES ($1, $2, $3, $4, $5, false)`,
			sessionID, userID, platform, authUserID, tokenHash,
		)
		if insertErr != nil {
			return false, fmt.Errorf("chatrepo: ensure session insert: %w", insertErr)
		}
		return true, nil
	}

	// Session exists — enforce ownership unless caller is trusted.
	if !trusted {
		ownerOK := false
		switch {
		case authUserID != nil && stored.authUserID != nil && *authUserID == *stored.authUserID:
			ownerOK = true // authenticated user owns this session
		case tokenHash != "" && stored.tokenHash == tokenHash:
			ownerOK = true // anonymous client presents valid token
		case stored.tokenHash == "":
			ownerOK = true // old session without token (migration path)
		}
		if !ownerOK {
			return false, chat.ErrSessionForbidden
		}
	}

	_, err = r.db.Exec(ctx,
		`UPDATE chat_sessions SET updated_at = NOW() WHERE session_id = $1`, sessionID)
	if err != nil {
		return false, fmt.Errorf("chatrepo: ensure session update: %w", err)
	}
	return false, nil
}

// GetSession returns the session or nil if not found.
func (r *Repository) GetSession(ctx context.Context, sessionID string) (*chat.Session, error) {
	var s chat.Session
	err := r.db.QueryRow(ctx,
		`SELECT session_id, auth_user_id, COALESCE(user_id,''), platform, is_human_mode
		   FROM chat_sessions WHERE session_id = $1`, sessionID).
		Scan(&s.SessionID, &s.AuthUserID, &s.UserID, &s.Platform, &s.IsHumanMode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("chatrepo: get session: %w", err)
	}
	return &s, nil
}

// ─── Messages ────────────────────────────────────────────────────────────────

// GetHistory returns the last N messages for a session (oldest first).
func (r *Repository) GetHistory(ctx context.Context, sessionID string, limit int) ([]chat.Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT role, content, COALESCE(sender_type,''), COALESCE(message_type,'text'),
		       COALESCE(meta_data, '{}'::jsonb), COALESCE(file_path,''), created_at
		FROM (
			SELECT * FROM chat_messages
			WHERE session_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		) sub
		ORDER BY created_at ASC`,
		sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("chatrepo: get history: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetPublicHistory returns messages for the public chat history endpoint.
// Access is granted when either:
//   - authUserID is provided and matches the session owner, OR
//   - tokenHash matches the stored session token hash.
//
// Returns chat.ErrSessionForbidden when neither condition is met.
func (r *Repository) GetPublicHistory(ctx context.Context, sessionID, tokenHash string, authUserID *string) ([]chat.PublicMessage, error) {
	// Load session ownership info.
	var stored struct {
		tokenHash  string
		authUserID *string
	}
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(session_token_hash,''), auth_user_id
		   FROM chat_sessions WHERE session_id = $1`,
		sessionID,
	).Scan(&stored.tokenHash, &stored.authUserID)
	if err == pgx.ErrNoRows {
		return nil, chat.ErrSessionForbidden
	}
	if err != nil {
		return nil, fmt.Errorf("chatrepo: get public history lookup: %w", err)
	}

	// Verify access.
	accessOK := false
	switch {
	case authUserID != nil && stored.authUserID != nil && *authUserID == *stored.authUserID:
		accessOK = true
	case tokenHash != "" && stored.tokenHash == tokenHash:
		accessOK = true
	case stored.tokenHash == "":
		// Old session without token — allow (migration path).
		accessOK = tokenHash != "" || authUserID != nil
	}
	if !accessOK {
		return nil, chat.ErrSessionForbidden
	}

	rows, err := r.db.Query(ctx, `
		SELECT role, content, COALESCE(sender_type,''), created_at
		FROM chat_messages
		WHERE session_id = $1
		ORDER BY created_at ASC`,
		sessionID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo: get public history: %w", err)
	}
	defer rows.Close()

	var msgs []chat.PublicMessage
	for rows.Next() {
		var (
			m         chat.PublicMessage
			createdAt time.Time
		)
		if err := rows.Scan(&m.Role, &m.Content, &m.SenderType, &createdAt); err != nil {
			return nil, fmt.Errorf("chatrepo: scan public history: %w", err)
		}
		m.Time = createdAt.Format("15:04")
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// SaveMessage persists a single chat message.
func (r *Repository) SaveMessage(ctx context.Context, sessionID, role, content, senderType, msgType string, meta map[string]any, filePath string) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		metaJSON = []byte("{}")
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO chat_messages
		  (session_id, role, content, sender_type, message_type, meta_data, file_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sessionID, role, content, senderType, msgType, metaJSON, filePath)
	if err != nil {
		return fmt.Errorf("chatrepo: save message: %w", err)
	}
	// Bump session updated_at.
	_, _ = r.db.Exec(ctx, `UPDATE chat_sessions SET updated_at = NOW() WHERE session_id = $1`, sessionID)
	return nil
}

// ─── Knowledge Search ────────────────────────────────────────────────────────

// SearchKnowledge calls match_sales_knowledge and returns relevant entries.
func (r *Repository) SearchKnowledge(ctx context.Context, embedding []float32, threshold float64, limit int, topicFilter string) ([]chat.KnowledgeMatch, error) {
	if len(embedding) == 0 {
		return nil, nil
	}

	vecStr := formatVector(embedding)

	filter := "{}"
	if topicFilter != "" {
		b, _ := json.Marshal(map[string]string{"topic": topicFilter})
		filter = string(b)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, content, COALESCE(metadata, '{}'::jsonb), similarity
		FROM match_sales_knowledge($1::vector, $2, $3, $4::jsonb)`,
		vecStr, threshold, limit, filter)
	if err != nil {
		return nil, fmt.Errorf("chatrepo: search knowledge: %w", err)
	}
	defer rows.Close()

	var matches []chat.KnowledgeMatch
	for rows.Next() {
		var (
			m       chat.KnowledgeMatch
			metaRaw []byte
		)
		if err := rows.Scan(&m.ID, &m.Content, &metaRaw, &m.Similarity); err != nil {
			return nil, fmt.Errorf("chatrepo: scan knowledge: %w", err)
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &m.Metadata)
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func scanMessages(rows pgx.Rows) ([]chat.Message, error) {
	var msgs []chat.Message
	for rows.Next() {
		var (
			m       chat.Message
			metaRaw []byte
		)
		if err := rows.Scan(
			&m.Role, &m.Content, &m.SenderType, &m.MessageType,
			&metaRaw, &m.FilePath, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("chatrepo: scan message: %w", err)
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &m.MetaData)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func formatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
