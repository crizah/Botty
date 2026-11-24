package wa

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	db "server/internal/db/sqlc"
	"server/internal/sarvam"
	"server/internal/services"
)

// TagStore is the database side of tag handling (implemented by services.TagService).
type TagStore interface {
	Authorize(ctx context.Context, groupJID string, senderIDs []string) (services.TagContext, error)
	OpenTasks(ctx context.Context, projectID string, limit int) ([]services.TaskCandidate, error)
	FindReport(ctx context.Context, tc services.TagContext, messageID string) (db.Report, bool)
	Create(ctx context.Context, tc services.TagContext, in services.TagInput) (services.TagResult, error)
	Close(ctx context.Context, tc services.TagContext, in services.TagInput) (services.TagResult, error)
}

// Classifier is the Sarvam side (implemented by sarvam.Client).
type Classifier interface {
	Choose(ctx context.Context, state, instructions string, options []sarvam.Option) (map[string]float64, error)
	Name(ctx context.Context, text string) (sarvam.Naming, error)
}

var (
	mentionTok = regexp.MustCompile(`@\S+`)
	// TODO: change - extend these lists as real chats show what people actually write
	doneRe = regexp.MustCompile(`(?i)\b(done|fixed|resolved|completed?|closed?|sorted|solved|finished|ho\s?gaya|ho\s?gya|hogaya|kar\s?diya|kardiya|(?:thik|theek)\s+ho\s+gaya)\b|✅`)
)

// isDone: replies containing a "done"-style word close work; anything else creates.
func isDone(text string) bool { return doneRe.MatchString(mentionTok.ReplaceAllString(text, " ")) }

const (
	optNone    = "NONE"     // doesn't match any existing task: a new request
	optNotTask = "NOT_TASK" // chit-chat, thanks, greetings: not a request at all
	taskTO     = 40 * time.Second
)

// TagHandler handles "@botty ..." sent as a reply to someone's message:
// create (new task or another report on an existing one) or done.
// Standalone tags (no reply) are ignored for now.
type TagHandler struct {
	Store     TagStore
	AI        Classifier
	Threshold float64
	MaxTasks  int
}

func (h *TagHandler) Handle(ctx context.Context, m Message) {
	if !(m.Mentioned || strings.Contains(strings.ToLower(m.Text), "@botty")) || m.QuotedID == "" {
		return
	}
	if linkCmd.MatchString(m.Text) {
		return // the link handler owns this
	}
	ctx, cancel := context.WithTimeout(ctx, taskTO)
	defer cancel()

	tc, err := h.Store.Authorize(ctx, m.GroupJID, m.SenderIDs)
	var se *services.Error
	switch {
	case err == nil:
	case errors.As(err, &se) && se.Kind == services.KindForbidden:
		_ = m.Reply(ctx, "Only people on the tag allowlist can tag me.")
		return
	case errors.As(err, &se) && se.Kind == services.KindNotFound:
		log.Printf("[wa] tag ignored (%s) sender IDs: %v", se.Msg, m.SenderIDs) // reply stays silent
		return
	default:
		log.Printf("[wa] tag authorize: %v", err)
		return
	}
	if strings.TrimSpace(m.QuotedText) == "" {
		_ = m.Reply(ctx, "I can only read text messages for now.")
		return
	}

	in := services.TagInput{TagMessageID: m.ID, MessageID: m.QuotedID, MessageText: m.QuotedText, ReporterWaID: ids(m.QuotedSender)}
	if isDone(m.Text) {
		h.done(ctx, m, tc, in)
	} else {
		h.create(ctx, m, tc, in)
	}
}

func (h *TagHandler) create(ctx context.Context, m Message, tc services.TagContext, in services.TagInput) {
	if _, dup := h.Store.FindReport(ctx, tc, in.MessageID); dup {
		_ = m.Reply(ctx, "That message is already tracked.")
		return
	}
	tasks, err := h.Store.OpenTasks(ctx, tc.ProjectID, h.MaxTasks)
	if err != nil {
		log.Printf("[wa] tag open tasks: %v", err)
		return
	}

	// classify first; the naming model is only called when a new task will actually be made
	d := h.decide(ctx, in.MessageText, tasks, true)
	switch d.kind {
	case kindNotTask:
		log.Printf("[wa] tag: classified as not a task, skipped")
		_ = m.Reply(ctx, "not a task")
		return
	case kindExisting:
		in.TaskID = d.taskID
	default:
		naming, err := h.AI.Name(ctx, in.MessageText)
		if err != nil || naming.Title == "" {
			if err != nil {
				log.Printf("[wa] naming failed, using fallback title: %v", err)
			}
			naming = sarvam.Naming{Title: fallbackTitle(in.MessageText)}
		}
		in.NewTitle, in.NewDesc = naming.Title, naming.Description
	}
	res, err := h.Store.Create(ctx, tc, in)
	if err != nil {
		log.Printf("[wa] tag create: %v", err)
		_ = m.Reply(ctx, "Something went wrong saving that. Try again.")
		return
	}
	log.Printf("[wa] tag: T-%d %q (new task: %v)", res.Number, res.Title, res.Created)
	_ = m.Reply(ctx, "done")
}

func (h *TagHandler) done(ctx context.Context, m Message, tc services.TagContext, in services.TagInput) {
	if _, ok := h.Store.FindReport(ctx, tc, in.MessageID); !ok { // not already a report: classify it
		tasks, err := h.Store.OpenTasks(ctx, tc.ProjectID, h.MaxTasks)
		if err != nil {
			log.Printf("[wa] tag open tasks: %v", err)
			return
		}
		d := h.decide(ctx, in.MessageText, tasks, false)
		in.TaskID = d.taskID
		if d.kind != kindExisting {
			_ = m.Reply(ctx, "I couldn't tell which task that is. Reply to the original request with @botty done.")
			return
		}
	}
	res, err := h.Store.Close(ctx, tc, in)
	if err != nil {
		log.Printf("[wa] tag close: %v", err)
		_ = m.Reply(ctx, "Something went wrong closing that. Try again.")
		return
	}
	log.Printf("[wa] tag: closed on T-%d %q", res.Number, res.Title)
	_ = m.Reply(ctx, "done")
}

type decisionKind int

const (
	kindNew      decisionKind = iota // make a new task
	kindExisting                     // belongs to an existing task
	kindNotTask                      // chit-chat, ignore
)

type decision struct {
	kind   decisionKind
	taskID string
}

// decide asks Prism where the message belongs: one of the open tasks, a new task, or (when
// allowNotTask) "not a task". A low-confidence or failed answer means a new task.
func (h *TagHandler) decide(ctx context.Context, text string, tasks []services.TaskCandidate, allowNotTask bool) decision {
	opts := make([]sarvam.Option, 0, len(tasks)+2)
	for _, t := range tasks {
		d := t.Title
		if t.Description != "" {
			d += " — " + t.Description
		}
		opts = append(opts, sarvam.Option{ID: t.ID, Text: d})
	}
	opts = append(opts, sarvam.Option{ID: optNone, Text: "None of the above: a different, new request or problem that needs doing"})
	instr := "Which existing task is this message about? Choose " + optNone + " if it is a different problem."
	if allowNotTask {
		opts = append(opts, sarvam.Option{ID: optNotTask, Text: "Not a task: chit-chat, thanks, greetings, jokes or a plain statement with nothing to do"})
		instr += " Choose " + optNotTask + " if it is not a request or problem at all."
	}

	probs, err := h.AI.Choose(ctx, text, instr, opts)
	if err != nil {
		log.Printf("[wa] prism: %v", err) // falls back to a new task
		return decision{kind: kindNew}
	}
	best, bp := "", 0.0
	for id, p := range probs {
		if p > bp {
			best, bp = id, p
		}
	}
	log.Printf("[wa] prism: top=%s p=%.2f (threshold %.2f)", best, bp, h.Threshold)
	switch {
	case bp < h.Threshold || best == optNone:
		return decision{kind: kindNew}
	case best == optNotTask:
		return decision{kind: kindNotTask}
	}
	return decision{kind: kindExisting, taskID: best}
}

// fallbackTitle is used when the naming model fails: the first few words of the message.
func fallbackTitle(text string) string {
	words := strings.Fields(mentionTok.ReplaceAllString(text, " "))
	if len(words) > 6 {
		words = words[:6]
	}
	t := strings.Join(words, " ")
	if t == "" {
		return "New request"
	}
	return strings.ToUpper(t[:1]) + t[1:]
}

func ids(vals ...string) []string {
	var out []string
	for _, v := range vals {
		if v != "" && v != "<nil>" {
			out = append(out, v)
		}
	}
	return out
}
