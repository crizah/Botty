package services

import (
	"context"
	"database/sql"
	"fmt"

	db "server/internal/db/sqlc"
)

// MessageLog keeps recent group messages so a tag can follow a reply chain.
type MessageLog struct {
	Queries *db.Queries
}

func NewMessageLog(dbConn *sql.DB) *MessageLog { return &MessageLog{Queries: db.New(dbConn)} }

// ThreadMsg is one message in a reply chain.
type ThreadMsg struct {
	ID       string
	Sender   string // display name, may be empty
	Text     string
	QuotedID string
}

type SaveMessage struct {
	GroupID, MessageID, SenderID, SenderName, Text, QuotedID string
}

func (l *MessageLog) Save(ctx context.Context, m SaveMessage) error {
	return l.Queries.CreateMessage(ctx, db.CreateMessageParams{
		ID: newID(), GroupID: m.GroupID, MessageID: m.MessageID, SenderWaID: m.SenderID,
		SenderName: m.SenderName, Text: m.Text, QuotedID: m.QuotedID,
	})
}

// Thread returns the reply chain that ends at startID, oldest first, at most max messages long.
// It is empty when startID was never logged; it stops early at the first message not in the log.
func (l *MessageLog) Thread(ctx context.Context, groupID, startID string, max int) []ThreadMsg {
	var rev []ThreadMsg
	seen := map[string]bool{}
	for id := startID; id != "" && len(rev) < max && !seen[id]; {
		seen[id] = true
		m, err := l.Queries.GetMessage(ctx, db.GetMessageParams{GroupID: groupID, MessageID: id})
		if err != nil {
			break
		}
		rev = append(rev, ThreadMsg{ID: m.MessageID, Sender: m.SenderName, Text: m.Text, QuotedID: m.QuotedID})
		id = m.QuotedID
	}
	out := make([]ThreadMsg, len(rev))
	for i, m := range rev {
		out[len(rev)-1-i] = m
	}
	return out
}

// Purge deletes messages older than the given number of days.
func (l *MessageLog) Purge(ctx context.Context, days int) (int64, error) {
	return l.Queries.PurgeMessages(ctx, fmt.Sprintf("-%d days", days))
}
