package wa

import (
	"context"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"hash/fnv"
	"log"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Message is the trimmed-down group message the rest of the app deals with.
type Message struct {
	ID           string
	GroupJID     string
	SenderJID    string   // may be a LID
	SenderAlt    string   // the phone-number JID when SenderJID is a LID (and vice versa), if known
	SenderIDs    []string // every form of the sender ID we can work out (LID and phone JID)
	PushName     string
	Text         string
	QuotedID     string    // message this one replies to
	QuotedText   string    // text of that message (empty for media without a caption)
	QuotedSender string    // who wrote it (JID or LID)
	Mentioned    bool      // the bot itself is @mentioned
	Image        *ImageRef // the image this message carries (its caption is Text)
	QuotedImage  *ImageRef // the image in the message this one replies to
	Time         time.Time

	// set by the router so handlers can answer without knowing about whatsmeow
	Reply     func(ctx context.Context, text string) error
	GroupName func(ctx context.Context) string
	// Participants lists the group's members (excluding the bot) with phone numbers resolved where possible.
	Participants func(ctx context.Context) ([]Participant, error)
}

// ImageRef is an image in a message; the bytes are only fetched when someone calls Download.
type ImageRef struct {
	Mime     string
	Size     uint64 // as declared by the sender's client
	Download func(ctx context.Context) ([]byte, error)
}

func imageRef(cli *whatsmeow.Client, img *waE2E.ImageMessage) *ImageRef {
	return &ImageRef{Mime: img.GetMimetype(), Size: img.GetFileLength(), Download: func(ctx context.Context) ([]byte, error) { return cli.Download(ctx, img) }}
}

// Participant is one group member as WhatsApp reports it.
type Participant struct {
	WaID  string // primary JID or LID
	Phone string // "+<number>" when known
	Name  string // push/contact name when known
}

// Handler is where message processing plugs in.
type Handler interface {
	Handle(ctx context.Context, m Message)
}

// Router fans messages out to N workers. A group always maps to the same worker, so one
// group's messages are handled in order while different groups run in parallel.
type Router struct {
	handler Handler
	queues  []chan func(context.Context)

	mu      sync.Mutex
	seenIDs map[string]struct{}
}

// seen reports whether a message ID was already dispatched (WhatsApp can redeliver).
// The set is bounded: it resets when it grows large, which is fine for catching close duplicates.
func (r *Router) seen(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.seenIDs) > 10000 {
		r.seenIDs = map[string]struct{}{}
	}
	if _, ok := r.seenIDs[id]; ok {
		return true
	}
	r.seenIDs[id] = struct{}{}
	return false
}

func NewRouter(h Handler, workers int) *Router {
	if workers < 1 {
		workers = 8
	}
	r := &Router{handler: h, queues: make([]chan func(context.Context), workers), seenIDs: map[string]struct{}{}}
	for i := range r.queues {
		r.queues[i] = make(chan func(context.Context), 256)
	}
	return r
}

func (r *Router) Start(ctx context.Context) {
	for _, q := range r.queues {
		go func(q chan func(context.Context)) {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-q:
					job(ctx)
				}
			}
		}(q)
	}
}

// Dispatch is called from the whatsmeow event handler, so it must return fast.
func (r *Router) Dispatch(cli *whatsmeow.Client, e *events.Message) {
	log.Printf("[wa] event: msg=%s chat=%s sender=%s fromMe=%v group=%v kind=%s fields=[%s] edit=%v ephemeral=%v viewonce=%v ts=%s",
		e.Info.ID, e.Info.Chat, e.Info.Sender, e.Info.IsFromMe, e.Info.IsGroup, e.Info.Type, fieldNames(e.Message), e.IsEdit, e.IsEphemeral, e.IsViewOnce, e.Info.Timestamp.UTC().Format("15:04:05"))
	if !e.Info.IsGroup {
		log.Printf("[wa] skip %s: not a group message", e.Info.ID)
		return
	}
	if e.Info.IsFromMe {
		log.Printf("[wa] skip %s: sent from the bot's own number", e.Info.ID)
		return
	}
	m := Message{
		ID:        e.Info.ID,
		GroupJID:  e.Info.Chat.String(),
		SenderJID: e.Info.Sender.String(),
		SenderAlt: e.Info.SenderAlt.String(),
		PushName:  e.Info.PushName,
		Time:      e.Info.Timestamp,
	}
	m.SenderIDs = senderIDs(cli, e)
	if e.Message.GetConversation() != "" {
		m.Text = e.Message.GetConversation()
	}
	var ci *waE2E.ContextInfo
	if x := e.Message.GetExtendedTextMessage(); x != nil {
		m.Text = x.GetText()
		ci = x.GetContextInfo()
	}
	if img := e.Message.GetImageMessage(); img != nil { // a photo: its caption is the text
		m.Text = img.GetCaption()
		ci = img.GetContextInfo()
		m.Image = imageRef(cli, img)
	}
	if ci != nil {
		m.QuotedID = ci.GetStanzaID()
		m.QuotedSender = ci.GetParticipant()
		m.QuotedText = quotedText(ci.GetQuotedMessage())
		if qi := ci.GetQuotedMessage().GetImageMessage(); qi != nil {
			m.QuotedImage = imageRef(cli, qi)
		}
		for _, jid := range ci.GetMentionedJID() {
			if isBot(cli, jid) {
				m.Mentioned = true
			}
		}
	}
	chat, callID, callSender, callMsg := e.Info.Chat, e.Info.ID, e.Info.Sender.ToNonAD().String(), e.Message
	m.Reply = func(ctx context.Context, text string) error {
		// a real WhatsApp reply: quotes the message that called the bot
		_, err := cli.SendMessage(ctx, chat, &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(callID),
				Participant:   proto.String(callSender),
				QuotedMessage: callMsg,
			},
		}})
		return err
	}
	m.GroupName = func(ctx context.Context) string {
		if gi, err := cli.GetGroupInfo(ctx, chat); err == nil && gi.Name != "" {
			return gi.Name
		}
		return "WhatsApp group"
	}
	m.Participants = func(ctx context.Context) ([]Participant, error) { return participants(ctx, cli, chat) }
	log.Printf("[wa] parsed %s: textLen=%d quotedID=%q quotedTextLen=%d mentioned=%v image=%v quotedImage=%v", m.ID, len(m.Text), m.QuotedID, len(m.QuotedText), m.Mentioned, m.Image != nil, m.QuotedImage != nil)
	if r.seen(m.ID) {
		log.Printf("[wa] skip %s: already dispatched (duplicate delivery)", m.ID)
		return
	}
	r.Submit(m.GroupJID, func(ctx context.Context) { r.handler.Handle(ctx, m) })
}

// Submit queues a job on the worker that owns this group, so jobs for one group run in order.
func (r *Router) Submit(groupJID string, job func(context.Context)) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(groupJID))
	q := r.queues[int(h.Sum32())%len(r.queues)]
	select {
	case q <- job:
	default:
		log.Printf("[wa] queue full, dropping a job for %s", groupJID) // TODO: change - decide on backpressure
	}
}

// fieldNames lists which fields of a message are set (conversation, imageMessage, ...), to see what kind it was.
func fieldNames(m *waE2E.Message) string {
	var out []string
	m.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		out = append(out, string(fd.Name()))
		return true
	})
	return strings.Join(out, ",")
}

func isBot(cli *whatsmeow.Client, jid string) bool {
	if cli.Store.ID != nil && jid == cli.Store.ID.ToNonAD().String() {
		return true
	}
	return !cli.Store.LID.IsEmpty() && jid == cli.Store.LID.ToNonAD().String()
}

// participants fetches the group's member list and maps each WhatsApp ID to a phone number and name.
func participants(ctx context.Context, cli *whatsmeow.Client, chat types.JID) ([]Participant, error) {
	gi, err := cli.GetGroupInfo(ctx, chat)
	if err != nil {
		return nil, err
	}
	out := make([]Participant, 0, len(gi.Participants))
	for _, p := range gi.Participants {
		if isBot(cli, p.JID.String()) || isBot(cli, p.PhoneNumber.String()) || isBot(cli, p.LID.String()) {
			continue
		}
		pn := p.PhoneNumber
		if pn.IsEmpty() {
			if p.JID.Server == types.DefaultUserServer {
				pn = p.JID
			} else if lid := firstNonEmpty(p.LID, p.JID); lid.Server == types.HiddenUserServer {
				if got, err := cli.Store.LIDs.GetPNForLID(ctx, lid); err == nil {
					pn = got
				}
			}
		}
		part := Participant{WaID: p.JID.String()}
		if !pn.IsEmpty() {
			part.Phone = "+" + pn.User
		}
		for _, j := range []types.JID{pn, p.JID, p.LID} {
			if j.IsEmpty() {
				continue
			}
			if c, err := cli.Store.Contacts.GetContact(ctx, j); err == nil {
				if name := firstNonEmptyStr(c.FullName, c.PushName, c.BusinessName); name != "" {
					part.Name = name
					break
				}
			}
		}
		out = append(out, part)
	}
	return out, nil
}

func firstNonEmpty(js ...types.JID) types.JID {
	for _, j := range js {
		if !j.IsEmpty() {
			return j
		}
	}
	return types.EmptyJID
}

func firstNonEmptyStr(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func quotedText(q *waE2E.Message) string {
	switch {
	case q.GetConversation() != "":
		return q.GetConversation()
	case q.GetExtendedTextMessage().GetText() != "":
		return q.GetExtendedTextMessage().GetText()
	case q.GetImageMessage().GetCaption() != "":
		return q.GetImageMessage().GetCaption()
	case q.GetVideoMessage().GetCaption() != "":
		return q.GetVideoMessage().GetCaption()
	}
	return ""
}

// senderIDs returns every form of the sender's ID we can find (LID and phone JID), because a
// client can send in either form and group members are stored under whichever WhatsApp gave us.
func senderIDs(cli *whatsmeow.Client, e *events.Message) []string {
	seen := map[string]bool{}
	var out []string
	add := func(j types.JID) {
		if s := j.ToNonAD().String(); !j.IsEmpty() && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	add(e.Info.Sender)
	add(e.Info.SenderAlt)
	ctx := context.Background()
	for _, j := range []types.JID{e.Info.Sender, e.Info.SenderAlt} {
		switch j.Server {
		case types.HiddenUserServer:
			if pn, err := cli.Store.LIDs.GetPNForLID(ctx, j.ToNonAD()); err == nil {
				add(pn)
			}
		case types.DefaultUserServer:
			if lid, err := cli.Store.LIDs.GetLIDForPN(ctx, j.ToNonAD()); err == nil {
				add(lid)
			}
		}
	}
	return out
}
