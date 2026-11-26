package wa

import (
	"context"
	"log"
	"strings"

	"server/internal/services"
)

// MessageLogger saves text messages from linked groups so tags can follow reply chains.
// Link commands are skipped so claim codes are never stored.
type MessageLogger struct {
	Groups *services.GroupService
	Log    *services.MessageLog
}

func (h *MessageLogger) Handle(ctx context.Context, m Message) {
	if strings.TrimSpace(m.Text) == "" || linkCmd.MatchString(m.Text) {
		return
	}
	gid, ok := h.Groups.GroupIDByJID(ctx, m.GroupJID)
	if !ok {
		return
	}
	sender := ""
	if len(m.SenderIDs) > 0 {
		sender = m.SenderIDs[0]
	}
	if err := h.Log.Save(ctx, services.SaveMessage{GroupID: gid, MessageID: m.ID, SenderID: sender, SenderName: m.PushName, Text: m.Text, QuotedID: m.QuotedID}); err != nil {
		log.Printf("[wa] message log: %v", err)
	}
}
