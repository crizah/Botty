package wa

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"

	"server/internal/services"
)

// MemberStore is what the keeper needs from the database (implemented by services.GroupService).
type MemberStore interface {
	GroupIDByJID(ctx context.Context, jid string) (string, bool)
	SyncMembers(ctx context.Context, groupID string, members []services.MemberSync) error
	HasMember(ctx context.Context, groupID string, waIDs ...string) (bool, error)
	MarkLeftExcept(ctx context.Context, groupID string, keep []string) (int64, error)
}

// resyncCooldown stops a sender we can't match (e.g. an ID format mismatch) from re-syncing on every message.
const resyncCooldown = 2 * time.Minute

// Keeper keeps a linked group's member list current: after linking, when people join or
// leave, and when a message arrives from someone we don't have a row for.
type Keeper struct {
	Store MemberStore

	mu     sync.Mutex
	lastAt map[string]time.Time // group JID -> last reconcile
}

func NewKeeper(s MemberStore) *Keeper { return &Keeper{Store: s, lastAt: map[string]time.Time{}} }

// Reconcile loads the group's real member list and makes our table match it:
// new people are added (IDs mapped to phone/name), people who left are flagged.
func (k *Keeper) Reconcile(ctx context.Context, groupJID, groupID string, fetch func(context.Context) ([]Participant, error)) {
	k.mu.Lock()
	k.lastAt[groupJID] = time.Now()
	k.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	ps, err := fetch(ctx)
	if err != nil {
		log.Printf("[wa] member sync (group %s): %v", groupID, err)
		return
	}
	members := make([]services.MemberSync, 0, len(ps))
	ids := make([]string, 0, len(ps))
	withPhone := 0
	for _, p := range ps {
		if p.Phone != "" {
			withPhone++
		}
		ids = append(ids, p.WaID)
		members = append(members, services.MemberSync{WaID: p.WaID, Phone: p.Phone, PushName: p.Name})
	}
	if err := k.Store.SyncMembers(ctx, groupID, members); err != nil {
		log.Printf("[wa] member sync (group %s): %v", groupID, err)
		return
	}
	left, err := k.Store.MarkLeftExcept(ctx, groupID, ids)
	if err != nil {
		log.Printf("[wa] member sync (group %s): marking leavers: %v", groupID, err)
	}
	log.Printf("[wa] group %s: %d members (%d with phone numbers), %d left", groupID, len(members), withPhone, left)
}

// OnGroupInfo handles WhatsApp's group-change events. Only joins and leaves of linked groups matter.
func (k *Keeper) OnGroupInfo(ctx context.Context, e *events.GroupInfo, fetch func(context.Context) ([]Participant, error)) {
	if len(e.Join) == 0 && len(e.Leave) == 0 {
		return
	}
	jid := e.JID.String()
	id, ok := k.Store.GroupIDByJID(ctx, jid)
	if !ok {
		return // not linked: ignore
	}
	k.Reconcile(ctx, jid, id, fetch)
}

// Handle is a message Handler: if a linked group's message comes from someone we don't have, re-sync.
func (k *Keeper) Handle(ctx context.Context, m Message) {
	id, ok := k.Store.GroupIDByJID(ctx, m.GroupJID)
	if !ok {
		return
	}
	ids := []string{m.SenderJID}
	if m.SenderAlt != "" && m.SenderAlt != "<nil>" {
		ids = append(ids, m.SenderAlt)
	}
	if known, err := k.Store.HasMember(ctx, id, ids...); err != nil || known {
		return
	}
	k.mu.Lock()
	recent := time.Since(k.lastAt[m.GroupJID]) < resyncCooldown
	k.mu.Unlock()
	if recent {
		return
	}
	k.Reconcile(ctx, m.GroupJID, id, m.Participants)
}

// Chain runs several handlers on every message, in order.
type Chain []Handler

func (c Chain) Handle(ctx context.Context, m Message) {
	for _, h := range c {
		h.Handle(ctx, m)
	}
}
