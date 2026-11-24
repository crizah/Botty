// Package ingest is the (stubbed) WhatsApp side of the system.
//
// Nothing here talks to WhatsApp yet. It only marks where the real pieces plug in.
package ingest

import (
	"context"
	"log"
	"time"
)

// Worker discovers groups the bot has been added to and keeps their members up to date.
//
// TODO: change - replace the polling stub with whatsmeow events (group join, group
// participant updates, new messages). If the library gives us an "added to group" event,
// the ticker goes away entirely.
type Worker struct {
	Interval time.Duration
	Resolver IdentityResolver
}

func (w *Worker) Run(ctx context.Context) {
	interval := w.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}
	log.Printf("[ingest] STUB worker running (every %s) - no WhatsApp connection yet", interval)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// TODO: change - poll / drain WhatsApp events, upsert wa_groups + wa_group_members,
			// then call w.Resolver for members without a phone number.
		}
	}
}
