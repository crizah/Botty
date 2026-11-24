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
	created  *services.TagInput
	closed   *services.TagInput
	openReps []services.ReportCandidate
	dupOf    string
}

func (f *fakeStore) Authorize(context.Context, string, []string) (services.TagContext, error) {
	return services.TagContext{ProjectID: "p", GroupID: "g", MemberID: "m"}, f.authErr
}
func (f *fakeStore) OpenTasks(context.Context, string, int) ([]services.TaskCandidate, error) {
	return f.tasks, nil
}
func (f *fakeStore) FindReport(context.Context, services.TagContext, string) (db.Report, bool) {
	return db.Report{}, f.existing
}
func (f *fakeStore) Create(_ context.Context, _ services.TagContext, in services.TagInput) (services.TagResult, error) {
	f.created = &in
	return services.TagResult{Number: 7, Title: "X", Created: in.TaskID == ""}, nil
}
func (f *fakeStore) Close(_ context.Context, _ services.TagContext, in services.TagInput) (services.TagResult, error) {
	f.closed = &in
	return services.TagResult{Number: 7, Title: "X"}, nil
}

type fakeAI struct {
	probs     map[string]float64
	seq       []map[string]float64 // if set, successive Choose calls return these in order
	calls     *int
	err       error
	nameCalls *int
}

func (f fakeAI) Choose(context.Context, string, string, []sarvam.Option) (map[string]float64, error) {
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

func TestIsDone(t *testing.T) {
	for _, s := range []string{"@botty done", "@botty fixed ✅", "@botty ho gaya", "@botty kar diya", "resolved"} {
		if !isDone(s) {
			t.Errorf("%q should be done", s)
		}
	}
	for _, s := range []string{"@botty add this", "@botty on it", "@botty please track", "@donetsk hi"} {
		if isDone(s) {
			t.Errorf("%q should be create", s)
		}
	}
}

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
	h := &TagHandler{Store: st, AI: fakeAI{probs: map[string]float64{"t1": 0.3, "t2": 0.2, optNone: 0.5}}, Threshold: 0.6, MaxTasks: 50}
	if r := run(h, "@botty done"); !strings.Contains(r, "couldn't tell") {
		t.Fatalf("reply %q", r)
	}
	if st.closed != nil || st.created != nil {
		t.Fatal("must not write anything")
	}
}

func TestDoneOnExistingReportSkipsModel(t *testing.T) {
	st := &fakeStore{existing: true}
	h := &TagHandler{Store: st, AI: fakeAI{err: errors.New("must not be called")}, Threshold: 0.6, MaxTasks: 50}
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
