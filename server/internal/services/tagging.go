package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	db "server/internal/db/sqlc"
)

// TagService is what the WhatsApp bot uses when someone tags it: it authorises the sender and
// turns a tagged message into a task / report, or closes one. The bot acts as actor_kind "bot".
type TagService struct {
	db      *sql.DB
	Queries *db.Queries
}

func NewTagService(dbConn *sql.DB) *TagService {
	return &TagService{db: dbConn, Queries: db.New(dbConn)}
}

// TagContext says where a tag came from: a linked group attached to a project, and an allowed sender.
type TagContext struct {
	ProjectID string
	GroupID   string
	MemberID  string // the member who tagged the bot
}

// TaskCandidate is an open task the classifier can pick.
type TaskCandidate struct {
	ID          string
	Number      int64
	Title       string
	Description string
}

// Authorize checks the group is linked + attached to a project and the sender is allowlisted.
// NotFound = not linked / not attached / unknown sender (stay silent); Forbidden = known but not allowed.
func (s *TagService) Authorize(ctx context.Context, groupJID string, senderIDs []string) (TagContext, error) {
	g, err := s.Queries.GetGroupByJID(ctx, groupJID)
	if err != nil {
		return TagContext{}, notFoundOr(err, "group not linked")
	}
	pid, err := s.Queries.GetProjectIDForGroup(ctx, g.ID)
	if err != nil {
		return TagContext{}, notFoundOr(err, "group not in a project")
	}
	m, err := s.Queries.GetActiveMemberByWaIDs(ctx, db.GetActiveMemberByWaIDsParams{GroupID: g.ID, Ids: senderIDs})
	if err != nil {
		return TagContext{}, notFoundOr(err, "unknown sender")
	}
	n, err := s.Queries.IsAllowlisted(ctx, db.IsAllowlistedParams{ProjectID: pid, GroupID: g.ID, MemberID: m.ID})
	if err != nil {
		return TagContext{}, err
	}
	if n == 0 {
		return TagContext{}, Forbidden("not on the tag allowlist")
	}
	return TagContext{ProjectID: pid, GroupID: g.ID, MemberID: m.ID}, nil
}

// OpenTasks returns the most recently active open tasks, at most limit.
func (s *TagService) OpenTasks(ctx context.Context, projectID string, limit int) ([]TaskCandidate, error) {
	rows, err := s.Queries.ListOpenTasks(ctx, db.ListOpenTasksParams{ProjectID: projectID, Limit: int64(limit)})
	out := make([]TaskCandidate, 0, len(rows))
	for _, r := range rows {
		out = append(out, TaskCandidate{ID: r.ID, Number: r.Number, Title: r.Title, Description: r.Description})
	}
	return out, err
}

// TagInput is the message being turned into (or added to) a task.
type TagInput struct {
	TaskID       string // existing task to add to; empty = create a new one
	NewTitle     string
	NewDesc      string
	MessageID    string
	MessageText  string
	TagMessageID string   // the message that tagged the bot; replying to it later also finds this report
	ReporterWaID []string // who wrote the quoted message (any of their IDs)
}

// TagResult describes what happened, for the bot's reply.
type TagResult struct {
	Number  int64
	Title   string
	Created bool // a new task was made
}

func (s *TagService) reporterID(ctx context.Context, q *db.Queries, tc TagContext, ids []string) *string {
	if len(ids) == 0 {
		return nil
	}
	m, err := q.GetActiveMemberByWaIDs(ctx, db.GetActiveMemberByWaIDsParams{GroupID: tc.GroupID, Ids: ids})
	if err != nil {
		return nil // reporter not a known member: keep the report, just without a reporter
	}
	return &m.ID
}

// Create adds the quoted message as a report on an existing task, or makes a new task for it.
func (s *TagService) Create(ctx context.Context, tc TagContext, in TagInput) (TagResult, error) {
	var res TagResult
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		var t db.Task
		if in.TaskID != "" {
			var err error
			if t, err = q.GetTask(ctx, in.TaskID); err != nil || t.ProjectID != tc.ProjectID {
				return NotFound("task not found")
			}
		} else {
			num, err := q.NextTaskNumber(ctx, tc.ProjectID)
			if err != nil {
				return err
			}
			if t, err = q.CreateTask(ctx, db.CreateTaskParams{ID: newID(), ProjectID: tc.ProjectID, Number: num, Title: truncate(in.NewTitle, 120), Description: truncate(in.NewDesc, 500)}); err != nil {
				return err
			}
			res.Created = true
			if err := logAudit(ctx, q, auditEntry{ProjectID: tc.ProjectID, TaskID: t.ID, ActorKind: "bot", Action: "task.created", Detail: t.Title}); err != nil {
				return err
			}
		}
		rep, err := q.CreateReport(ctx, db.CreateReportParams{
			ID: newID(), TaskID: t.ID, GroupID: &tc.GroupID, ReporterMemberID: s.reporterID(ctx, q, tc, in.ReporterWaID),
			MessageID: in.MessageID, MessageText: in.MessageText, AssignedByMemberID: &tc.MemberID, TagMessageID: in.TagMessageID,
		})
		if err != nil {
			return err
		}
		if err := logAudit(ctx, q, auditEntry{ProjectID: tc.ProjectID, TaskID: t.ID, ReportID: rep.ID, ActorKind: "bot", Action: "report.created", Detail: truncate(in.MessageText, 80)}); err != nil {
			return err
		}
		res.Number, res.Title = t.Number, t.Title
		return syncTaskStatus(ctx, q, t, "a new report was added")
	})
	return res, err
}

// FindReport looks for a report that was made from this exact WhatsApp message.
func (s *TagService) FindReport(ctx context.Context, tc TagContext, messageID string) (db.Report, bool) {
	if messageID == "" {
		return db.Report{}, false
	}
	r, err := s.Queries.GetReportByMessage(ctx, db.GetReportByMessageParams{ProjectID: tc.ProjectID, GroupID: &tc.GroupID, MessageID: messageID})
	return r, err == nil
}

// Close marks work as done, closed via a WhatsApp tag by the tagging member.
// If the quoted message already is a report it is closed directly; otherwise the quoted message
// becomes a report on in.TaskID and is closed right away.
func (s *TagService) Close(ctx context.Context, tc TagContext, in TagInput) (TagResult, error) {
	var res TagResult
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		var rep db.Report
		if r, err := q.GetReportByMessage(ctx, db.GetReportByMessageParams{ProjectID: tc.ProjectID, GroupID: &tc.GroupID, MessageID: in.MessageID}); in.MessageID != "" && err == nil {
			rep = r
		} else {
			t, err := q.GetTask(ctx, in.TaskID)
			if err != nil || t.ProjectID != tc.ProjectID {
				return NotFound("task not found")
			}
			if rep, err = q.CreateReport(ctx, db.CreateReportParams{
				ID: newID(), TaskID: t.ID, GroupID: &tc.GroupID, ReporterMemberID: s.reporterID(ctx, q, tc, in.ReporterWaID),
				MessageID: in.MessageID, MessageText: in.MessageText, AssignedByMemberID: &tc.MemberID, TagMessageID: in.TagMessageID,
			}); err != nil {
				return err
			}
			if err := logAudit(ctx, q, auditEntry{ProjectID: tc.ProjectID, TaskID: t.ID, ReportID: rep.ID, ActorKind: "bot", Action: "report.created", Detail: truncate(in.MessageText, 80)}); err != nil {
				return err
			}
		}
		t, err := q.GetTask(ctx, rep.TaskID)
		if err != nil {
			return err
		}
		if rep.Status != "done" {
			via, now := "tag", time.Now().UTC()
			if rep, err = q.SetReportStatus(ctx, db.SetReportStatusParams{ID: rep.ID, Status: "done", ClosedByMemberID: &tc.MemberID, ClosedVia: &via, ClosedAt: &now}); err != nil {
				return err
			}
			if err := logAudit(ctx, q, auditEntry{ProjectID: tc.ProjectID, TaskID: t.ID, ReportID: rep.ID, ActorKind: "bot", Action: "report.status_changed", Old: "todo", New: "done", Detail: "closed via WhatsApp tag"}); err != nil {
				return err
			}
		}
		if err := syncTaskStatus(ctx, q, t, "all reports are done"); err != nil {
			return err
		}
		res.Number, res.Title = t.Number, t.Title
		return nil
	})
	return res, err
}

func (c TaskCandidate) String() string { return fmt.Sprintf("T-%d %s", c.Number, c.Title) }
