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
	probs map[string]float64
	err   error
}

func (f fakeAI) Choose(context.Context, string, string, []sarvam.Option) (map[string]float64, error) {
	return f.probs, f.err
}
func (f fakeAI) Name(context.Context, string) (sarvam.Naming, error) {
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
	var replied bool
	h.Handle(context.Background(), Message{Text: "@botty add", Mentioned: true, Reply: func(context.Context, string) error { replied = true; return nil }})
	if replied {
		t.Fatal("standalone tags are ignored for now")
	}
}
