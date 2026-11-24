package wa

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"server/internal/services"
)

// linkCmd matches "link <code>" anywhere after the bot is addressed.
var linkCmd = regexp.MustCompile(`(?i)\blink\s+((?:botty[-\s]?)?[a-z0-9-]{8,})`)

const (
	maxFailures   = 5
	failureWindow = 10 * time.Minute
)

// LinkHandler handles "@botty link <code>" (a real mention or the plain text "@botty") and
// ignores everything else without logging or storing it.
// TODO: change - extend with tag handling (task creation) once that design is settled.
type LinkHandler struct {
	Claims *services.ClaimService
	Keeper *Keeper

	mu       sync.Mutex
	failures map[string][]time.Time // sender -> recent failed attempts
}

func NewLinkHandler(c *services.ClaimService, k *Keeper) *LinkHandler {
	return &LinkHandler{Claims: c, Keeper: k, failures: map[string][]time.Time{}}
}

func (h *LinkHandler) Handle(ctx context.Context, m Message) {
	addressed := m.Mentioned || strings.Contains(strings.ToLower(m.Text), "@botty")
	if !addressed {
		return
	}
	match := linkCmd.FindStringSubmatch(m.Text)
	log.Printf("[wa] bot addressed in %s (mention=%v, link command=%v)", m.GroupJID, m.Mentioned, match != nil)
	if match == nil {
		return // addressed but not a link command; tag handling comes later
	}
	if h.blocked(m.SenderJID) {
		return
	}

	g, err := h.Claims.Redeem(ctx, match[1], m.GroupJID, m.GroupName(ctx))
	var se *services.Error
	switch {
	case err == nil:
		log.Printf("[wa] linked group %s to its owner", g.ID)
		go h.Keeper.Reconcile(ctx, m.GroupJID, g.ID, m.Participants) // build the member list in the background
		_ = m.Reply(ctx, "yipee yayay linked!!")
	case errors.As(err, &se) && se.Kind == services.KindNotFound:
		h.fail(m.SenderJID)
		_ = m.Reply(ctx, "That code is invalid or expired. Generate a new one in the dashboard.")
	case errors.As(err, &se) && se.Kind == services.KindConflict:
		_ = m.Reply(ctx, "This group is already linked.")
	default:
		log.Printf("[wa] link failed: %v", err)
	}
}

func (h *LinkHandler) recent(sender string) []time.Time {
	cut := time.Now().Add(-failureWindow)
	var keep []time.Time
	for _, t := range h.failures[sender] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	h.failures[sender] = keep
	return keep
}

func (h *LinkHandler) blocked(sender string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.recent(sender)) >= maxFailures
}

func (h *LinkHandler) fail(sender string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failures[sender] = append(h.recent(sender), time.Now())
}
