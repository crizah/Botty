package ingest

import "context"

// IdentityResolver maps a WhatsApp id (JID or the newer opaque LID) to a phone number.
type IdentityResolver interface {
	// ResolvePhone returns ok=false when the number cannot be determined (yet).
	ResolvePhone(ctx context.Context, groupJID, waID string) (phone string, ok bool, err error)
}

// StubResolver never resolves anything.
// TODO: change - implement with whatsmeow's LID <-> phone mapping store.
type StubResolver struct{}

func (StubResolver) ResolvePhone(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}
