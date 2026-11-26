package wa

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strconv"
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
	OpenReportsBy(ctx context.Context, tc services.TagContext, taskID string, reporterWaIDs []string, limit int) ([]services.ReportCandidate, error)
	AddDuplicate(ctx context.Context, tc services.TagContext, reportID string, in services.TagInput) error
	Thread(ctx context.Context, groupID, startID string, max int) []services.ThreadMsg
}

// Classifier is the Sarvam side (implemented by sarvam.Client).
type Classifier interface {
	Choose(ctx context.Context, state, instructions string, options []sarvam.Option) (map[string]float64, error)
	Name(ctx context.Context, text string) (sarvam.Naming, error)
}

var (
	mentionTok = regexp.MustCompile(`@\S+`)
)

var botMention = regexp.MustCompile(`(?i)@botty\b|@\d{7,}`)

// stripBotMention removes "@botty" and mention-by-number tokens, leaving the actual request.
func stripBotMention(text string) string {
	return strings.Join(strings.Fields(botMention.ReplaceAllString(text, " ")), " ")
}

const (
	optNone     = "NONE"      // doesn't match any existing task: a new request
	optNotTask  = "NOT_TASK"  // chit-chat, thanks, greetings: not a request at all
	optDiff     = "DIFFERENT" // not the exact same message as any earlier one
	optCreate   = "CREATE"    // intent: make a task / report from the quoted message
	optFinished = "DONE"      // intent: the work is finished
	optInvalid  = "INVALID"   // intent: not an instruction for the bot at all
	taskTO      = 40 * time.Second
)

// TagHandler handles "@botty ...":
//   - as a reply to someone's message: create (new task or another report on an existing one) or done
//   - standalone ("@botty my water broke"): create only, from the message's own text. Standalone
//     tags never complete anything.
type TagHandler struct {
	Store        TagStore
	AI           Classifier
	Threshold    float64 // min Prism probability to file under an existing task
	DupThreshold float64 // min probability to call two messages from the same person the exact same
	MaxTasks     int
	ChainDepth   int // most messages of a reply chain sent to the classifier (1 = just the replied-to message)
}

func (h *TagHandler) Handle(ctx context.Context, m Message) {
	if !(m.Mentioned || strings.Contains(strings.ToLower(m.Text), "@botty")) {
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
	if m.QuotedID == "" { // standalone: the sender reports their own issue
		text := stripBotMention(m.Text)
		if text == "" {
			return
		}
		h.create(ctx, m, tc, services.TagInput{MessageID: m.ID, MessageText: text, ReporterWaID: m.SenderIDs}, "")
		return
	}
	if strings.TrimSpace(m.QuotedText) == "" {
		_ = m.Reply(ctx, "I can only read text messages for now.")
		return
	}

	in := services.TagInput{TagMessageID: m.ID, MessageID: m.QuotedID, MessageText: m.QuotedText, ReporterWaID: ids(m.QuotedSender)}
	thread := h.thread(ctx, tc, m)
	it, err := h.intent(ctx, m.Text)
	if err != nil {
		log.Printf("[wa] prism (intent): %v", err)
		_ = m.Reply(ctx, "Something went wrong. Try again.")
		return
	}
	switch it {
	case intentInvalid:
		_ = m.Reply(ctx, "that's not a valid instruction")
	case intentDone:
		h.done(ctx, m, tc, in, thread)
	default:
		h.create(ctx, m, tc, in, thread)
	}
}

type intentKind int

const (
	intentCreate intentKind = iota
	intentDone
	intentInvalid
)

// intent asks Prism what a reply tag means: create a task, finish one, or nothing the bot can act on.
// Only the tag message's own text is sent, never the message it replies to. Standalone tags don't call this.
func (h *TagHandler) intent(ctx context.Context, tagText string) (intentKind, error) {
	text := stripBotMention(tagText)
	if text == "" {
		return intentCreate, nil // a bare "@botty" reply: nothing to read, so it creates
	}
	probs, err := h.AI.Choose(ctx, text,
		"This is a short message sent to a task-tracking bot as a reply to someone else's message. What is the sender asking the bot to do?",
		[]sarvam.Option{
			{ID: optCreate, Text: "Create or track a task from that message: add this, on it, noted, will do, I'll look into it, please handle this"},
			{ID: optFinished, Text: "Say the work is finished: done, fixed, resolved, completed, sorted, closed, ho gaya, kar diya"},
			{ID: optInvalid, Text: "Not a valid instruction for a task bot: a joke, a random request, chit-chat, or something unrelated to tasks"},
		})
	if err != nil {
		return intentCreate, err
	}
	log.Printf("[wa] prism (intent): create=%.2f done=%.2f invalid=%.2f", probs[optCreate], probs[optFinished], probs[optInvalid])
	switch {
	case probs[optInvalid] > probs[optCreate] && probs[optInvalid] > probs[optFinished]:
		return intentInvalid, nil
	case probs[optFinished] > probs[optCreate]:
		return intentDone, nil
	}
	return intentCreate, nil
}

func (h *TagHandler) create(ctx context.Context, m Message, tc services.TagContext, in services.TagInput, thread string) {
	classify := in.MessageText // what the classifier reads: the message, or its whole reply chain
	if thread != "" {
		classify = thread
	}
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
	d := h.decide(ctx, classify, tasks, true, thread != "")
	switch d.kind {
	case kindNotTask:
		log.Printf("[wa] tag: classified as not a task, skipped")
		_ = m.Reply(ctx, "not a task")
		return
	case kindExisting:
		in.TaskID = d.taskID
		// same person, same task: is this the exact same message they already sent?
		if cands, err := h.Store.OpenReportsBy(ctx, tc, d.taskID, in.ReporterWaID, 10); err != nil {
			log.Printf("[wa] tag open reports: %v", err)
		} else if len(cands) > 0 {
			if repID := h.sameReport(ctx, in.MessageText, cands); repID != "" {
				if err := h.Store.AddDuplicate(ctx, tc, repID, in); err != nil {
					log.Printf("[wa] tag duplicate: %v", err)
					_ = m.Reply(ctx, "Something went wrong saving that. Try again.")
					return
				}
				log.Printf("[wa] tag: counted as a repeat of an existing report")
				_ = m.Reply(ctx, "done")
				return
			}
		}
	default:
		naming, err := h.AI.Name(ctx, classify)
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

func (h *TagHandler) done(ctx context.Context, m Message, tc services.TagContext, in services.TagInput, thread string) {
	classify := in.MessageText
	if thread != "" {
		classify = thread
	}
	if _, ok := h.Store.FindReport(ctx, tc, in.MessageID); !ok { // not already a report: classify it
		tasks, err := h.Store.OpenTasks(ctx, tc.ProjectID, h.MaxTasks)
		if err != nil {
			log.Printf("[wa] tag open tasks: %v", err)
			return
		}
		d := h.decide(ctx, classify, tasks, false, thread != "")
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
func (h *TagHandler) decide(ctx context.Context, text string, tasks []services.TaskCandidate, allowNotTask, isThread bool) decision {
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
	if isThread {
		instr = "This is a reply chain, oldest message first. Which existing task is the LAST message about? Use the earlier messages for context. Choose " + optNone + " if it is a different problem."
	}
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

// sameReport asks Prism whether the new message says EXACTLY the same thing as one of the
// person's earlier open reports. It returns that report's ID, or "" when it is different.
func (h *TagHandler) sameReport(ctx context.Context, text string, cands []services.ReportCandidate) string {
	opts := make([]sarvam.Option, 0, len(cands)+1)
	for _, c := range cands {
		opts = append(opts, sarvam.Option{ID: c.ID, Text: c.Text})
	}
	opts = append(opts, sarvam.Option{ID: optDiff, Text: "A different message: it reports a different problem or adds something new"})
	probs, err := h.AI.Choose(ctx, text, "Is this message saying EXACTLY the same thing as one of these earlier messages from the same person? Choose "+optDiff+" unless it clearly repeats one of them.", opts)
	if err != nil {
		log.Printf("[wa] prism (duplicate check): %v", err) // treat as different
		return ""
	}
	best, bp := "", 0.0
	for id, p := range probs {
		if p > bp {
			best, bp = id, p
		}
	}
	log.Printf("[wa] prism duplicate check: top=%s p=%.2f (threshold %.2f)", best, bp, h.DupThreshold)
	if best == optDiff || bp < h.DupThreshold {
		return ""
	}
	return best
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

// thread builds the text the classifier reads for a reply tag: the replied-to message plus the
// messages it was replying to, oldest first. It is "" when there is no chain to show (just the
// one message), so those cases keep sending the plain message.
func (h *TagHandler) thread(ctx context.Context, tc services.TagContext, m Message) string {
	if h.ChainDepth < 2 {
		log.Printf("[wa] thread: off (TAG_CHAIN_DEPTH=%d)", h.ChainDepth)
		return ""
	}
	msgs := h.Store.Thread(ctx, tc.GroupID, m.QuotedID, h.ChainDepth)
	switch {
	case len(msgs) == 0:
		log.Printf("[wa] thread: replied-to message %s is not in the log (sent before logging, or text-less)", m.QuotedID)
		return ""
	case msgs[len(msgs)-1].ID != m.QuotedID:
		log.Printf("[wa] thread: walk ended on %s, not on the replied-to message %s", msgs[len(msgs)-1].ID, m.QuotedID)
		return ""
	case len(msgs) == 1:
		log.Printf("[wa] thread: %s is logged but has no earlier message in the log (its parent %q is missing or it isn't a reply)", m.QuotedID, msgs[0].QuotedID)
		return ""
	}
	msgs[len(msgs)-1].Text = m.QuotedText // WhatsApp's own copy of the replied-to message
	out := formatThread(msgs)
	log.Printf("[wa] thread: %d messages sent to the classifier:\n%s", len(msgs), out)
	return out
}

// formatThread labels a reply chain for the classifier.
func formatThread(msgs []services.ThreadMsg) string {
	var b strings.Builder
	b.WriteString("Thread (oldest first):")
	for i, t := range msgs {
		name := t.Sender
		if name == "" {
			name = "Someone"
		}
		b.WriteString("\n" + strconv.Itoa(i+1) + ". " + name)
		if i > 0 {
			b.WriteString(" (replying to " + strconv.Itoa(i) + ")")
		}
		b.WriteString(": " + t.Text)
	}
	return b.String()
}
