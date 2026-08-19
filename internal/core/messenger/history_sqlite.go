package messenger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"hop.top/kit/go/storage/sqldb"
)

// ConversationStoreOptions tunes the sqlite-backed conversation store.
type ConversationStoreOptions struct {
	// MaxTurnsPerConversation caps retained turns per conversation; older
	// turns are pruned on append. <= 0 uses DefaultConversationRetention.
	MaxTurnsPerConversation int
}

func (o ConversationStoreOptions) retention() int {
	if o.MaxTurnsPerConversation <= 0 {
		return DefaultConversationRetention
	}
	return o.MaxTurnsPerConversation
}

// SQLiteConversationStore is the ConversationStore backed by the shared kit
// sqlite connection (WAL, busy timeout) — the same primitive the session
// registry uses, so no new database dependency is introduced.
type SQLiteConversationStore struct {
	db   *sql.DB
	opts ConversationStoreOptions
}

var _ ConversationStore = (*SQLiteConversationStore)(nil)

const conversationSchema = `
CREATE TABLE IF NOT EXISTS message_turns (
	seq             INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id TEXT NOT NULL,
	session_id      TEXT NOT NULL,
	service_id      TEXT NOT NULL DEFAULT '',
	platform        TEXT NOT NULL DEFAULT '',
	profile_id      TEXT NOT NULL DEFAULT '',
	action_name     TEXT NOT NULL DEFAULT '',
	direction       TEXT NOT NULL,
	message_id      TEXT NOT NULL DEFAULT '',
	channel_id      TEXT NOT NULL DEFAULT '',
	sender_id       TEXT NOT NULL DEFAULT '',
	sender_name     TEXT NOT NULL DEFAULT '',
	text            TEXT NOT NULL DEFAULT '',
	attachments     TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_message_turns_conversation ON message_turns(conversation_id, seq);
CREATE INDEX IF NOT EXISTS idx_message_turns_session ON message_turns(session_id, seq);
`

const turnColumns = `seq, conversation_id, session_id, service_id, platform, profile_id, action_name,
	direction, message_id, channel_id, sender_id, sender_name, text, attachments, created_at`

// OpenConversationStore opens (or creates) the sqlite store at path. The
// parent directory is created when missing.
func OpenConversationStore(path string, opts ConversationStoreOptions) (*SQLiteConversationStore, error) {
	db, err := sqldb.Open(sqldb.Options{Path: path})
	if err != nil {
		return nil, fmt.Errorf("open conversation store: %w", err)
	}
	if _, err := db.ExecContext(context.Background(), conversationSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate conversation store: %w", err)
	}
	return &SQLiteConversationStore{db: db, opts: opts}, nil
}

// OpenDefaultConversationStore opens the store at DefaultConversationStorePath.
func OpenDefaultConversationStore(opts ConversationStoreOptions) (*SQLiteConversationStore, error) {
	path, err := DefaultConversationStorePath()
	if err != nil {
		return nil, err
	}
	return OpenConversationStore(path, opts)
}

// AppendTurn persists the turn, prunes turns beyond the retention cap for
// that conversation, and returns the turn with Seq assigned.
func (s *SQLiteConversationStore) AppendTurn(ctx context.Context, turn ConversationTurn) (ConversationTurn, error) {
	if err := turn.Validate(); err != nil {
		return ConversationTurn{}, fmt.Errorf("append conversation turn: %w", err)
	}
	if turn.Timestamp.IsZero() {
		turn.Timestamp = time.Now().UTC()
	}
	attachments := ""
	if len(turn.Attachments) > 0 {
		encoded, err := json.Marshal(turn.Attachments)
		if err != nil {
			return ConversationTurn{}, fmt.Errorf("encode turn attachments: %w", err)
		}
		attachments = string(encoded)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ConversationTurn{}, fmt.Errorf("begin conversation append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `INSERT INTO message_turns (
		conversation_id, session_id, service_id, platform, profile_id, action_name,
		direction, message_id, channel_id, sender_id, sender_name, text, attachments, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		turn.ConversationID, turn.SessionID, turn.ServiceID, turn.Platform, turn.ProfileID, turn.ActionName,
		turn.Direction, turn.MessageID, turn.ChannelID, turn.SenderID, turn.SenderName, turn.Text, attachments,
		turn.Timestamp.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return ConversationTurn{}, fmt.Errorf("insert conversation turn: %w", err)
	}
	seq, err := result.LastInsertId()
	if err != nil {
		return ConversationTurn{}, fmt.Errorf("read conversation turn seq: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM message_turns
		WHERE conversation_id = ? AND seq NOT IN (
			SELECT seq FROM message_turns WHERE conversation_id = ? ORDER BY seq DESC LIMIT ?
		)`, turn.ConversationID, turn.ConversationID, s.opts.retention()); err != nil {
		return ConversationTurn{}, fmt.Errorf("prune conversation turns: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ConversationTurn{}, fmt.Errorf("commit conversation append: %w", err)
	}
	turn.Seq = seq
	turn.Timestamp = turn.Timestamp.UTC()
	return turn, nil
}

// RecentTurns returns the newest turns for the query, oldest first.
func (s *SQLiteConversationStore) RecentTurns(ctx context.Context, query ConversationQuery) ([]ConversationTurn, error) {
	if query.ConversationID == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = DefaultPriorTurnLimit
	}
	// Newest N by seq, then reversed so callers see chronological order. An
	// empty session filter matches every session in the conversation.
	rows, err := s.db.QueryContext(ctx, `SELECT `+turnColumns+` FROM (
		SELECT `+turnColumns+` FROM message_turns
		WHERE conversation_id = ? AND (? = '' OR session_id = ?)
		ORDER BY seq DESC LIMIT ?
	) ORDER BY seq ASC`, query.ConversationID, query.SessionID, query.SessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("query conversation turns: %w", err)
	}
	defer func() { _ = rows.Close() }()
	turns := make([]ConversationTurn, 0, limit)
	for rows.Next() {
		turn, err := scanTurn(rows)
		if err != nil {
			return nil, err
		}
		turns = append(turns, turn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation turns: %w", err)
	}
	return turns, nil
}

// ListConversations summarises conversations, most recently appended first.
func (s *SQLiteConversationStore) ListConversations(ctx context.Context, filter ConversationFilter) ([]ConversationSummary, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = -1 // sqlite: negative LIMIT means no limit
	}
	rows, err := s.db.QueryContext(ctx, `SELECT last.conversation_id, last.service_id, last.platform, last.channel_id,
		agg.turn_count, agg.first_at, last.created_at, last.direction, last.text
	FROM message_turns last
	JOIN (
		SELECT conversation_id, COUNT(*) AS turn_count, MIN(created_at) AS first_at, MAX(seq) AS last_seq
		FROM message_turns GROUP BY conversation_id
	) agg ON agg.conversation_id = last.conversation_id AND agg.last_seq = last.seq
	WHERE (? = '' OR last.service_id = ?) AND (? = '' OR last.platform = ?)
	ORDER BY last.seq DESC LIMIT ?`,
		filter.ServiceID, filter.ServiceID, filter.Platform, filter.Platform, limit)
	if err != nil {
		return nil, fmt.Errorf("query conversations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	summaries := []ConversationSummary{}
	for rows.Next() {
		var summary ConversationSummary
		var firstAt, lastAt string
		if err := rows.Scan(&summary.ConversationID, &summary.ServiceID, &summary.Platform, &summary.ChannelID,
			&summary.TurnCount, &firstAt, &lastAt, &summary.LastDirection, &summary.LastText); err != nil {
			return nil, fmt.Errorf("scan conversation summary: %w", err)
		}
		summary.FirstAt = parseTurnTime(firstAt)
		summary.LastAt = parseTurnTime(lastAt)
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	return summaries, nil
}

// Close releases the sqlite connection.
func (s *SQLiteConversationStore) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close conversation store: %w", err)
	}
	return nil
}

func scanTurn(rows *sql.Rows) (ConversationTurn, error) {
	var turn ConversationTurn
	var attachments, createdAt string
	if err := rows.Scan(&turn.Seq, &turn.ConversationID, &turn.SessionID, &turn.ServiceID, &turn.Platform,
		&turn.ProfileID, &turn.ActionName, &turn.Direction, &turn.MessageID, &turn.ChannelID,
		&turn.SenderID, &turn.SenderName, &turn.Text, &attachments, &createdAt); err != nil {
		return ConversationTurn{}, fmt.Errorf("scan conversation turn: %w", err)
	}
	if attachments != "" {
		if err := json.Unmarshal([]byte(attachments), &turn.Attachments); err != nil {
			return ConversationTurn{}, fmt.Errorf("decode turn attachments: %w", err)
		}
	}
	turn.Timestamp = parseTurnTime(createdAt)
	return turn, nil
}

func parseTurnTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
