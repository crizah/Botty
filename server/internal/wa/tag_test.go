package wa

import (
	"context"
	"errors"
	"strings"
	"testing"

	db "server/internal/db/sqlc"
	"server/internal/sarvam"
	"server/internal/services"
)

type fakeStore struct {
	authErr  error
	tasks    []services.TaskCandidate
	existing bool
	rep      db.Report // returned when existing
	created  *services.TagInput
	closed   *services.TagInput
	openReps []services.ReportCandidate
	dupOf    string
	thread   []services.ThreadMsg
	reportID string
}

func (f *fakeStore) Authorize(context.Context, string, []string) (services.TagContext, error) {
	return services.TagContext{ProjectID: "p", GroupID: "g", MemberID: "m"}, f.authErr
}
func (f *fakeStore) OpenTasks(context.Context, string, int) ([]services.TaskCandidate, error) {
	return f.tasks, nil
}
func (f *fakeStore) FindReport(context.Context, services.TagContext, string) (db.Report, bool) {
	return f.rep, f.existing
}
func (f *fakeStore) Create(_ context.Context, _ services.TagContext, in services.TagInput) (services.TagResult, error) {
	f.created = &in
	return services.TagResult{Number: 7, Title: "X", Created: in.TaskID == "", ReportID: "r-new"}, nil
}
func (f *fakeStore) Thread(context.Context, string, string, int) []services.ThreadMsg {
	return f.thread
}
func (f *fakeStore) Close(_ context.Context, _ services.TagContext, in services.TagInput) (services.TagResult, error) {
	f.closed = &in
	return services.TagResult{Number: 7, Title: "X"}, nil
}

type fakeAI struct {
	probs       map[string]float64
	seq         []map[string]float64 // if set, successive Choose calls return these in order
	calls       *int
	finished    bool // the intent call answers DONE
	report      bool // the intent call answers REPORT
	invalid     bool // the intent call answers INVALID
	intentErr   error
	intentState *string // records what text the intent call was given
	err         error
	nameCalls   *int
}

func (f fakeAI) Choose(_ context.Context, state, _ string, opts []sarvam.Option) (map[string]float64, error) {
	if len(opts) == 4 && opts[0].ID == optCreate { // the intent call
		if f.intentState != nil {
			*f.intentState = state
		}
		if f.intentErr != nil {
			return nil, f.intentErr
		}
		switch {
		case f.report:
			return map[string]float64{optCreate: 0.1, optFinished: 0.1, optReport: 0.7, optInvalid: 0.1}, nil
		case f.invalid:
			return map[string]float64{optCreate: 0.1, optFinished: 0.1, optInvalid: 0.8}, nil
		case f.finished:
			return map[string]float64{optCreate: 0.1, optFinished: 0.8, optInvalid: 0.1}, nil
		}
		return map[string]float64{optCreate: 0.8, optFinished: 0.1, optInvalid: 0.1}, nil
	}
	if f.seq != nil {
		i := *f.calls
		*f.calls++
		return f.seq[i], f.err
	}
	return f.probs, f.err
}
func (f fakeAI) Name(context.Context, string) (sarvam.Naming, error) {
	if f.nameCalls != nil {
		*f.nameCalls++
	}
	return sarvam.Naming{Title: "Water leak", Description: "A leak."}, nil
}

func run(h *TagHandler, text string) string {
	var reply string
	h.Handle(context.Background(), Message{
		SenderIDs: []string{"s"}, Text: text, Mentioned: true, QuotedID: "q1", QuotedText: "paani leak ho raha hai",
		Reply: func(_ context.Context, t string) error { reply = t; return nil },
	})
	return reply
}

var tasks = []services.TaskCandidate{{ID: "t1", Number: 1, Title: "Water leak"}, {ID: "t2", Number: 2, Title: "Wifi down"}}

func TestCreateNewWhenLowConfidence(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{"t1": 0.4, "t2": 0.1, optNone: 0.5}}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty add"); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.created.TaskID != "" || st.created.NewTitle != "Water leak" {
		t.Fatalf("%+v", st.created)
	}
}

func TestCreateAddsToExistingWhenConfident(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{"t1": 0.8, "t2": 0.1, optNone: 0.1}}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty on it"); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.created.TaskID != "t1" {
		t.Fatalf("%+v", st.created)
	}
}

func TestPrismErrorFallsBackToNewTask(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{err: errors.New("boom")}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty add"); r != "done" {
		t.Fatalf("reply %q", r)
	}
}

func TestDoneLowConfidenceDoesNotCreate(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{finished: true, probs: map[string]float64{"t1": 0.3, "t2": 0.2, optNone: 0.5}}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty done"); !strings.Contains(r, "couldn't tell") {
		t.Fatalf("reply %q", r)
	}
	if st.closed != nil || st.created != nil {
		t.Fatal("must not write anything")
	}
}

func TestDoneOnExistingReportSkipsTaskClassifier(t *testing.T) {
	st := &fakeStore{existing: true}
	h := &TagHandler{Store: st, AI: fakeAI{finished: true, err: errors.New("task classifier must not be called")}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty done"); r != "done" {
		t.Fatalf("reply %q", r)
	}
}

func TestNotAllowedAndStandalone(t *testing.T) {
	st := &fakeStore{authErr: services.Forbidden("x")}
	h := &TagHandler{Store: st, AI: fakeAI{}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty add"); !strings.Contains(r, "allowlist") {
		t.Fatalf("reply %q", r)
	}

}

type panicAI struct{}

func (panicAI) Choose(context.Context, string, string, []sarvam.Option) (map[string]float64, error) {
	panic("the model must not see untagged messages")
}
func (panicAI) Name(context.Context, string) (sarvam.Naming, error) {
	panic("the model must not see untagged messages")
}

// Only messages that tag the bot (real mention or "@botty") may reach Sarvam.
func TestUntaggedMessagesNeverReachTheModel(t *testing.T) {
	h := &TagHandler{Store: &fakeStore{tasks: tasks}, AI: panicAI{}, Threshold: 0.6, MaxTasks: 50}
	reply := func(context.Context, string) error { t.Fatal("must not reply"); return nil }
	for _, text := range []string{"paani leak ho raha hai", "done", "fixed ✅", "add this please", "botty done"} {
		h.Handle(context.Background(), Message{SenderIDs: []string{"s"}, Text: text, QuotedID: "q1", QuotedText: "leak", Reply: reply})
	}
	// tagged but not allowed: still must not reach the model
	deny := &TagHandler{Store: &fakeStore{authErr: services.Forbidden("x")}, AI: panicAI{}, Threshold: 0.6, MaxTasks: 50}
	deny.Handle(context.Background(), Message{SenderIDs: []string{"s"}, Text: "@botty add", Mentioned: true, QuotedID: "q1", QuotedText: "leak", Reply: func(context.Context, string) error { return nil }})
}

func TestNotATaskSkipsNamingAndCreatesNothing(t *testing.T) {
	st, calls := &fakeStore{tasks: tasks}, 0
	h := &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{"t1": 0.05, "t2": 0.05, optNone: 0.1, optNotTask: 0.8}, nameCalls: &calls}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty add"); !strings.Contains(r, "not a task") {
		t.Fatalf("reply %q", r)
	}
	if st.created != nil || calls != 0 {
		t.Fatalf("created=%v nameCalls=%d, want nothing", st.created, calls)
	}
}

func TestNamingOnlyWhenNewTaskIsMade(t *testing.T) {
	// matches an existing task: no naming call
	st, calls := &fakeStore{tasks: tasks}, 0
	h := &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{"t1": 0.9, "t2": 0.05, optNone: 0.03, optNotTask: 0.02}, nameCalls: &calls}, Threshold: 0.6, MaxTasks: 50}
	run(h, "@botty add")
	if calls != 0 {
		t.Fatalf("naming called %d times for an existing-task match", calls)
	}
	// new task: exactly one naming call
	h.AI = fakeAI{probs: map[string]float64{"t1": 0.1, "t2": 0.1, optNone: 0.7, optNotTask: 0.1}, nameCalls: &calls}
	run(h, "@botty add")
	if calls != 1 {
		t.Fatalf("naming called %d times for a new task, want 1", calls)
	}
}

func (f *fakeStore) OpenReportsBy(context.Context, services.TagContext, string, []string, int) ([]services.ReportCandidate, error) {
	return f.openReps, nil
}
func (f *fakeStore) AddDuplicate(_ context.Context, _ services.TagContext, reportID string, _ services.TagInput) error {
	f.dupOf = reportID
	return nil
}

func dupHandler(st *fakeStore, second map[string]float64) *TagHandler {
	n := 0
	first := map[string]float64{"t1": 0.9, "t2": 0.05, optNone: 0.03, optNotTask: 0.02}
	return &TagHandler{Store: st, AI: fakeAI{seq: []map[string]float64{first, second}, calls: &n}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
}

func TestSamePersonNoEarlierReportAddsAsIs(t *testing.T) {
	n := 0
	st := &fakeStore{tasks: tasks} // no open reports from this person
	h := &TagHandler{Store: st, AI: fakeAI{seq: []map[string]float64{{"t1": 0.9, optNone: 0.1}}, calls: &n}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
	if r := run(h, "@botty add"); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.created == nil || st.created.TaskID != "t1" || st.dupOf != "" || n != 1 {
		t.Fatalf("created=%v dupOf=%q classifier calls=%d (want 1)", st.created, st.dupOf, n)
	}
}

func TestExactRepeatIsCountedNotAddedAsNewReport(t *testing.T) {
	st := &fakeStore{tasks: tasks, openReps: []services.ReportCandidate{{ID: "r1", Text: "paani leak ho raha hai"}}}
	h := dupHandler(st, map[string]float64{"r1": 0.95, optDiff: 0.05})
	if r := run(h, "@botty add"); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.dupOf != "r1" || st.created != nil {
		t.Fatalf("dupOf=%q created=%v, want a repeat of r1 and no new report", st.dupOf, st.created)
	}
}

func TestDifferentMessageFromSamePersonMakesAnotherReport(t *testing.T) {
	st := &fakeStore{tasks: tasks, openReps: []services.ReportCandidate{{ID: "r1", Text: "tap leaking"}}}
	h := dupHandler(st, map[string]float64{"r1": 0.2, optDiff: 0.8})
	run(h, "@botty add")
	if st.dupOf != "" || st.created == nil || st.created.TaskID != "t1" {
		t.Fatalf("dupOf=%q created=%v, want a new report on t1", st.dupOf, st.created)
	}
	// a borderline "same" below the duplicate threshold also counts as different
	st2 := &fakeStore{tasks: tasks, openReps: []services.ReportCandidate{{ID: "r1", Text: "tap leaking"}}}
	h2 := dupHandler(st2, map[string]float64{"r1": 0.6, optDiff: 0.4})
	run(h2, "@botty add")
	if st2.dupOf != "" || st2.created == nil {
		t.Fatal("0.6 is under the 0.7 duplicate threshold, must be a new report")
	}
}

func standalone(h *TagHandler, text string) (reply string) {
	h.Handle(context.Background(), Message{
		ID: "B9", SenderIDs: []string{"s"}, Text: text, Mentioned: true,
		Reply: func(_ context.Context, t string) error { reply = t; return nil },
	})
	return reply
}

func TestStandaloneCreatesFromOwnText(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{"t1": 0.1, "t2": 0.1, optNone: 0.7, optNotTask: 0.1}}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
	if r := standalone(h, "@botty my water broke"); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.created == nil || st.created.MessageText != "my water broke" || st.created.MessageID != "B9" || len(st.created.ReporterWaID) != 1 {
		t.Fatalf("%+v", st.created)
	}
	if st.closed != nil {
		t.Fatal("standalone must never close anything")
	}
}

func TestStandaloneNeverCompletes(t *testing.T) {
	st := &fakeStore{tasks: tasks, existing: true}
	h := &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{"t1": 0.9, optNone: 0.1}}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
	standalone(h, "@botty done")
	standalone(h, "@botty fixed the tap")
	if st.closed != nil {
		t.Fatalf("a standalone tag closed something: %+v", st.closed)
	}
}

func TestStandaloneNeedsAllowlistAndText(t *testing.T) {
	deny := &TagHandler{Store: &fakeStore{authErr: services.Forbidden("x")}, AI: panicAI{}, Threshold: 0.6, MaxTasks: 50}
	if r := standalone(deny, "@botty my water broke"); !strings.Contains(r, "allowlist") {
		t.Fatalf("reply %q", r)
	}
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: panicAI{}, Threshold: 0.6, MaxTasks: 50}
	if r := standalone(h, "@botty"); r != "" || st.created != nil {
		t.Fatal("a bare tag must be ignored")
	}
}

func TestStripBotMention(t *testing.T) {
	for in, want := range map[string]string{
		"@botty my water broke":            "my water broke",
		"@Botty   room 4 fan\nnot working": "room 4 fan not working",
		"@919876543210 tap leaking":        "tap leaking",
		"tap leaking @botty":               "tap leaking",
		"@botty":                           "",
	} {
		if got := stripBotMention(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestIntentSeesOnlyTheReplyText(t *testing.T) {
	var seen string
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{intentState: &seen, probs: map[string]float64{optNone: 1}}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
	run(h, "@botty on it") // the quoted message is "paani leak ho raha hai"
	if seen != "on it" {
		t.Fatalf("intent call was given %q, want only the reply text without the mention or quoted message", seen)
	}
}

func TestIntentCanSayFinishedForPhrasesNoKeywordListHas(t *testing.T) {
	st := &fakeStore{existing: true}
	h := &TagHandler{Store: st, AI: fakeAI{finished: true}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty all good now"); r != "done" || st.closed == nil || st.created != nil {
		t.Fatalf("reply=%q closed=%v created=%v", r, st.closed, st.created)
	}
}

func TestIntentFailureDoesNothing(t *testing.T) {
	st := &fakeStore{tasks: tasks, existing: true}
	h := &TagHandler{Store: st, AI: fakeAI{intentErr: errors.New("boom")}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty done"); !strings.Contains(r, "went wrong") || st.created != nil || st.closed != nil {
		t.Fatalf("reply=%q created=%v closed=%v", r, st.created, st.closed)
	}
}

func TestBareReplyTagCreatesWithoutAnIntentCall(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{intentErr: errors.New("must not be called"), probs: map[string]float64{optNone: 1}}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
	if r := run(h, "@botty"); r != "done" || st.created == nil {
		t.Fatalf("reply=%q created=%v", r, st.created)
	}
}

func TestInvalidInstructionIsRefusedAndNothingIsSaved(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{invalid: true, probs: map[string]float64{optNone: 1}}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
	if r := run(h, "@botty make me a sandwich"); !strings.Contains(r, "not a valid instruction") {
		t.Fatalf("reply %q", r)
	}
	if st.created != nil || st.closed != nil {
		t.Fatal("an invalid instruction must not create or close anything")
	}
}

func TestStandaloneTagsMakeNoIntentCall(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{intentErr: errors.New("intent must not be called"), probs: map[string]float64{optNone: 1}}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
	if r := standalone(h, "@botty make me a sandwich"); r != "done" || st.created == nil {
		t.Fatalf("reply=%q created=%v", r, st.created)
	}
}

func TestFormatThread(t *testing.T) {
	got := formatThread([]services.ThreadMsg{{Sender: "Asha", Text: "didnt get access on mail"}, {Text: "mine too"}})
	want := "Thread (oldest first):\n1. Asha: didnt get access on mail\n2. Someone (replying to 1): mine too"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestOwnReportJoinsTheRepliedMessagesTaskWithoutClassifying(t *testing.T) {
	calls := 0
	st := &fakeStore{existing: true, rep: db.Report{TaskID: "t9"}}
	h := &TagHandler{Store: st, AI: fakeAI{report: true, seq: []map[string]float64{{}}, calls: &calls}, Threshold: 0.6, MaxTasks: 50}
	var reply string
	h.Handle(context.Background(), Message{
		ID: "tag1", SenderIDs: []string{"b"}, Text: "@botty yeah for me neither", Mentioned: true, QuotedID: "q1", QuotedText: "water isnt working",
		Reply: func(_ context.Context, t string) error { reply = t; return nil },
	})
	if reply != "done" || st.created == nil {
		t.Fatalf("reply %q created %+v", reply, st.created)
	}
	c := st.created
	if c.TaskID != "t9" || c.MessageID != "tag1" || c.MessageText != "yeah for me neither" || len(c.ReporterWaID) != 1 || c.ReporterWaID[0] != "b" {
		t.Fatalf("%+v", c)
	}
	if calls != 0 {
		t.Fatalf("task classifier was called %d times", calls)
	}
}

func TestOwnReportOnUntrackedMessageUsesNormalCreate(t *testing.T) {
	st := &fakeStore{tasks: tasks}
	h := &TagHandler{Store: st, AI: fakeAI{report: true, probs: map[string]float64{optNone: 0.9}}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty me too"); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.created == nil || st.created.MessageID != "q1" {
		t.Fatalf("%+v", st.created)
	}
}

func ownReportHandler(st *fakeStore, probs map[string]float64) *TagHandler {
	return &TagHandler{Store: st, AI: fakeAI{report: true, probs: probs}, Threshold: 0.6, DupThreshold: 0.7, MaxTasks: 50}
}

func tagOwn(h *TagHandler) string {
	var reply string
	h.Handle(context.Background(), Message{
		ID: "tag2", SenderIDs: []string{"b"}, Text: "@botty me too", Mentioned: true, QuotedID: "q1", QuotedText: "water isnt working",
		Reply: func(_ context.Context, t string) error { reply = t; return nil },
	})
	return reply
}

func TestOwnReportRepeatIsCountedNotAddedAgain(t *testing.T) {
	st := &fakeStore{existing: true, rep: db.Report{TaskID: "t9"}, openReps: []services.ReportCandidate{{ID: "r7", Text: "me too"}}}
	if r := tagOwn(ownReportHandler(st, map[string]float64{"r7": 0.95, optDiff: 0.05})); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.dupOf != "r7" || st.created != nil {
		t.Fatalf("dupOf=%q created=%v, want a repeat of r7 and no new report", st.dupOf, st.created)
	}
}

func TestOwnReportDifferentFromEarlierOneIsAdded(t *testing.T) {
	st := &fakeStore{existing: true, rep: db.Report{TaskID: "t9"}, openReps: []services.ReportCandidate{{ID: "r7", Text: "also the wifi"}}}
	tagOwn(ownReportHandler(st, map[string]float64{"r7": 0.2, optDiff: 0.8}))
	if st.dupOf != "" || st.created == nil || st.created.TaskID != "t9" {
		t.Fatalf("dupOf=%q created=%v, want a new report on t9", st.dupOf, st.created)
	}
}

type fakeMedia struct {
	saved    int
	reportID string
	msgID    string
	err      error
}

func (f *fakeMedia) Save(_ context.Context, _ services.TagContext, reportID, messageID string, _ []byte) error {
	f.saved++
	f.reportID, f.msgID = reportID, messageID
	return f.err
}

func img(size uint64, mime string) *ImageRef {
	return &ImageRef{Mime: mime, Size: size, Download: func(context.Context) ([]byte, error) { return []byte("bytes"), nil }}
}

func imgHandler(st *fakeStore, media *fakeMedia) *TagHandler {
	return &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{optNone: 0.9}}, Threshold: 0.6, MaxTasks: 50, Media: media, MaxImage: 5 << 20}
}

func sendImage(h *TagHandler, text string, in *ImageRef, quoted bool) (reply string) {
	m := Message{ID: "tag3", SenderIDs: []string{"s"}, Text: text, Mentioned: true, Image: in,
		Reply: func(_ context.Context, t string) error { reply = t; return nil }}
	if quoted {
		m.Image, m.QuotedImage = nil, in
		m.QuotedID, m.QuotedText = "q9", "sink is broken"
	}
	h.Handle(context.Background(), m)
	return reply
}

func TestCaptionedImageCreatesReportAndSavesImage(t *testing.T) {
	st, media := &fakeStore{tasks: tasks}, &fakeMedia{}
	if r := sendImage(imgHandler(st, media), "the sink is broken @botty", img(1000, "image/jpeg"), false); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.created == nil || st.created.MessageText != "the sink is broken" || media.saved != 1 || media.reportID != "r-new" || media.msgID != "tag3" {
		t.Fatalf("created=%+v media=%+v", st.created, media)
	}
}

func TestImageWithoutCaptionAsksForOne(t *testing.T) {
	st, media := &fakeStore{tasks: tasks}, &fakeMedia{}
	if r := sendImage(imgHandler(st, media), "@botty", img(1000, "image/jpeg"), false); r != noCaption {
		t.Fatalf("reply %q", r)
	}
	if st.created != nil || media.saved != 0 {
		t.Fatal("nothing should be saved")
	}
}

func TestRepliedToImageIsSavedOnTheReport(t *testing.T) {
	st, media := &fakeStore{tasks: tasks}, &fakeMedia{}
	if r := sendImage(imgHandler(st, media), "@botty add", img(1000, "image/png"), true); r != "done" {
		t.Fatalf("reply %q", r)
	}
	if st.created == nil || media.saved != 1 || media.msgID != "q9" {
		t.Fatalf("created=%+v media=%+v", st.created, media)
	}
}

func TestBigOrWrongTypeImageIsRefusedBeforeSaving(t *testing.T) {
	for _, in := range []*ImageRef{img(6<<20, "image/jpeg"), img(1000, "image/gif")} {
		st, media := &fakeStore{tasks: tasks}, &fakeMedia{}
		r := sendImage(imgHandler(st, media), "leak @botty", in, false)
		if r == "done" || st.created != nil || media.saved != 0 {
			t.Fatalf("reply %q created=%v", r, st.created)
		}
	}
}

func TestImageSaveFailureKeepsTheReport(t *testing.T) {
	st, media := &fakeStore{tasks: tasks}, &fakeMedia{err: errors.New("disk full")}
	if r := sendImage(imgHandler(st, media), "leak @botty", img(1000, "image/jpeg"), false); r != "done, but I couldn't save the image." || st.created == nil {
		t.Fatalf("reply %q created=%v", r, st.created)
	}
}
