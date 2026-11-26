package services

import (
	"context"
	"database/sql"
	"time"

	db "server/internal/db/sqlc"
	"server/internal/schemas"
)

type TaskService struct {
	db      *sql.DB
	Queries *db.Queries
}

func NewTaskService(dbConn *sql.DB) *TaskService {
	return &TaskService{db: dbConn, Queries: db.New(dbConn)}
}

// taskAccess loads a task and checks the caller's role on its project.
func (s *TaskService) taskAccess(ctx context.Context, taskID, userID string, min int) (db.Task, error) {
	t, err := s.Queries.GetTask(ctx, taskID)
	if err != nil {
		return t, notFoundOr(err, "task not found")
	}
	_, err = requireProject(ctx, s.Queries, t.ProjectID, userID, min)
	return t, err
}

func (s *TaskService) List(ctx context.Context, userID, projectID string) (schemas.TasksResponse, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelViewer); err != nil {
		return schemas.TasksResponse{}, err
	}
	rows, err := s.Queries.ListTasksByProject(ctx, projectID)
	if rows == nil {
		rows = []db.ListTasksByProjectRow{}
	}
	return schemas.TasksResponse{Tasks: rows}, err
}

func (s *TaskService) Create(ctx context.Context, userID, projectID string, req schemas.CreateTaskRequest) (db.Task, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelEditor); err != nil {
		return db.Task{}, err
	}
	var t db.Task
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		num, err := q.NextTaskNumber(ctx, projectID)
		if err != nil {
			return err
		}
		if t, err = q.CreateTask(ctx, db.CreateTaskParams{ID: newID(), ProjectID: projectID, Number: num, Title: req.Title, Description: req.Description, CreatedByUserID: &userID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, TaskID: t.ID, ActorUser: userID, Action: "task.created", Detail: t.Title})
	})
	return t, err
}

// Get returns a task with its reports, notes and activity.
func (s *TaskService) Get(ctx context.Context, userID, taskID string) (schemas.TaskDetailResponse, error) {
	t, err := s.taskAccess(ctx, taskID, userID, levelViewer)
	if err != nil {
		return schemas.TaskDetailResponse{}, err
	}
	out := schemas.TaskDetailResponse{Task: t}
	if out.Reports, err = s.Queries.ListReportsByTask(ctx, t.ID); err != nil {
		return out, err
	}
	if out.Duplicates, err = s.Queries.ListDuplicatesByTask(ctx, t.ID); err != nil {
		return out, err
	}
	if out.Duplicates == nil {
		out.Duplicates = []db.ReportDuplicate{}
	}
	if out.Media, err = s.Queries.ListMediaByTask(ctx, t.ID); err != nil {
		return out, err
	}
	if out.Media == nil {
		out.Media = []db.ListMediaByTaskRow{}
	}
	if out.Comments, err = s.Queries.ListCommentsByTask(ctx, t.ID); err != nil {
		return out, err
	}
	if out.Audit, err = s.Queries.ListAuditByTask(ctx, &t.ID); err != nil {
		return out, err
	}
	if out.Reports == nil {
		out.Reports = []db.ListReportsByTaskRow{}
	}
	if out.Comments == nil {
		out.Comments = []db.ListCommentsByTaskRow{}
	}
	if out.Audit == nil {
		out.Audit = []db.ListAuditByTaskRow{}
	}
	return out, nil
}

func (s *TaskService) Update(ctx context.Context, userID, taskID string, req schemas.UpdateTaskRequest) (db.Task, error) {
	t, err := s.taskAccess(ctx, taskID, userID, levelEditor)
	if err != nil {
		return t, err
	}
	var updated db.Task
	err = withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		if updated, err = q.UpdateTask(ctx, db.UpdateTaskParams{ID: t.ID, Title: req.Title, Description: req.Description}); err != nil {
			return err
		}
		if updated.Title != t.Title {
			if err := logAudit(ctx, q, auditEntry{ProjectID: t.ProjectID, TaskID: t.ID, ActorUser: userID, Action: "task.renamed", Old: t.Title, New: updated.Title}); err != nil {
				return err
			}
		}
		if updated.Description != t.Description {
			return logAudit(ctx, q, auditEntry{ProjectID: t.ProjectID, TaskID: t.ID, ActorUser: userID, Action: "task.description_changed"})
		}
		return nil
	})
	return updated, err
}

// SetStatus is only for tasks with no reports (e.g. created by hand). Once a task has
// reports its status is derived from them.
func (s *TaskService) SetStatus(ctx context.Context, userID, taskID string, req schemas.StatusRequest) (schemas.MessageResponse, error) {
	t, err := s.taskAccess(ctx, taskID, userID, levelEditor)
	if err != nil {
		return schemas.MessageResponse{}, err
	}
	err = withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		n, err := q.CountReports(ctx, t.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			return Conflict("this task has reports - change the status of its reports instead")
		}
		if t.Status == req.Status {
			return nil
		}
		if err := q.SetTaskStatus(ctx, db.SetTaskStatusParams{Status: req.Status, ID: t.ID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: t.ProjectID, TaskID: t.ID, ActorUser: userID, Action: "task.status_changed", Old: t.Status, New: req.Status})
	})
	return schemas.MessageResponse{Status: req.Status}, err
}

// CreateReport adds a report to a task. The ingest pipeline will call this when a tagged
// message arrives; for now it is also usable by hand from the dashboard.
// TODO: change - the bot path should set assigned_by from the allowlisted tagger.
func (s *TaskService) CreateReport(ctx context.Context, userID, taskID string, req schemas.CreateReportRequest) (db.Report, error) {
	t, err := s.taskAccess(ctx, taskID, userID, levelEditor)
	if err != nil {
		return db.Report{}, err
	}
	// group must be in this project; reporter / assigned_by must be members of that group
	if req.GroupID != "" {
		if err := requireGroupInProject(ctx, s.Queries, t.ProjectID, req.GroupID); err != nil {
			return db.Report{}, err
		}
	}
	for _, mid := range []string{req.ReporterMemberID, req.AssignedByMemberID} {
		if mid == "" {
			continue
		}
		m, err := s.Queries.GetMember(ctx, mid)
		if err != nil && err != sql.ErrNoRows {
			return db.Report{}, err
		}
		if err != nil || m.GroupID != req.GroupID {
			return db.Report{}, BadRequest("reporter / assigned_by must be members of the chosen group")
		}
	}

	var rep db.Report
	err = withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		var err error
		if rep, err = q.CreateReport(ctx, db.CreateReportParams{
			ID: newID(), TaskID: t.ID, GroupID: strPtr(req.GroupID), ReporterMemberID: strPtr(req.ReporterMemberID),
			MessageID: req.MessageID, MessageText: req.MessageText, AssignedByMemberID: strPtr(req.AssignedByMemberID),
		}); err != nil {
			return err
		}
		if err := logAudit(ctx, q, auditEntry{ProjectID: t.ProjectID, TaskID: t.ID, ReportID: rep.ID, ActorUser: userID, Action: "report.created", Detail: truncate(req.MessageText, 80)}); err != nil {
			return err
		}
		return syncTaskStatus(ctx, q, t, "a new report was added")
	})
	return rep, err
}

// SetReportStatus toggles one report and applies the rollup to its task.
func (s *TaskService) SetReportStatus(ctx context.Context, userID, reportID string, req schemas.StatusRequest) (schemas.ReportStatusResponse, error) {
	rep, err := s.Queries.GetReport(ctx, reportID)
	if err != nil {
		return schemas.ReportStatusResponse{}, notFoundOr(err, "report not found")
	}
	t, err := s.taskAccess(ctx, rep.TaskID, userID, levelEditor)
	if err != nil {
		return schemas.ReportStatusResponse{}, err
	}
	if req.Status == rep.Status {
		return schemas.ReportStatusResponse{Report: rep, TaskStatus: t.Status}, nil
	}

	out := schemas.ReportStatusResponse{}
	err = withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		params := db.SetReportStatusParams{ID: rep.ID, Status: req.Status}
		if req.Status == "done" {
			via, now := "dashboard", time.Now().UTC()
			params.ClosedByUserID, params.ClosedVia, params.ClosedAt = &userID, &via, &now
		} // reopening clears closed_* by leaving them NULL

		var err error
		if out.Report, err = q.SetReportStatus(ctx, params); err != nil {
			return err
		}
		if err := logAudit(ctx, q, auditEntry{ProjectID: t.ProjectID, TaskID: t.ID, ReportID: rep.ID, ActorUser: userID, Action: "report.status_changed", Old: rep.Status, New: req.Status, Detail: reportLabel(ctx, q, rep)}); err != nil {
			return err
		}
		reason := "all reports are done"
		if req.Status == "todo" {
			reason = "a report was reopened"
		}
		if err := syncTaskStatus(ctx, q, t, reason); err != nil {
			return err
		}
		cur, err := q.GetTask(ctx, t.ID)
		out.TaskStatus = cur.Status
		return err
	})
	return out, err
}

// syncTaskStatus applies the rollup rule: all reports done -> task done, any report
// todo -> task todo. Tasks without reports keep their manually set status.
func syncTaskStatus(ctx context.Context, q *db.Queries, t db.Task, reason string) error {
	total, err := q.CountReports(ctx, t.ID)
	if err != nil || total == 0 {
		return err
	}
	open, err := q.CountOpenReports(ctx, t.ID)
	if err != nil {
		return err
	}
	want := "done"
	if open > 0 {
		want = "todo"
	}
	cur, err := q.GetTask(ctx, t.ID)
	if err != nil || cur.Status == want {
		return err
	}
	if err := q.SetTaskStatus(ctx, db.SetTaskStatusParams{Status: want, ID: t.ID}); err != nil {
		return err
	}
	return logAudit(ctx, q, auditEntry{ProjectID: t.ProjectID, TaskID: t.ID, ActorKind: "system", Action: "task.status_changed", Old: cur.Status, New: want, Detail: reason})
}

func reportLabel(ctx context.Context, q *db.Queries, rep db.Report) string {
	if rep.ReporterMemberID != nil {
		if m, err := q.GetMember(ctx, *rep.ReporterMemberID); err == nil {
			return "report by " + memberLabel(m)
		}
	}
	return "report"
}

// ---- comments ----

func (s *TaskService) CommentOnTask(ctx context.Context, userID, taskID string, req schemas.CommentRequest) (db.Comment, error) {
	t, err := s.taskAccess(ctx, taskID, userID, levelEditor)
	if err != nil {
		return db.Comment{}, err
	}
	return s.addComment(ctx, t, nil, userID, req.Body)
}

func (s *TaskService) CommentOnReport(ctx context.Context, userID, reportID string, req schemas.CommentRequest) (db.Comment, error) {
	rep, err := s.Queries.GetReport(ctx, reportID)
	if err != nil {
		return db.Comment{}, notFoundOr(err, "report not found")
	}
	t, err := s.taskAccess(ctx, rep.TaskID, userID, levelEditor)
	if err != nil {
		return db.Comment{}, err
	}
	return s.addComment(ctx, t, &rep, userID, req.Body)
}

func (s *TaskService) addComment(ctx context.Context, t db.Task, rep *db.Report, userID, body string) (db.Comment, error) {
	var c db.Comment
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		entry := auditEntry{ProjectID: t.ProjectID, TaskID: t.ID, ActorUser: userID, Action: "comment.added", Detail: truncate(body, 80)}
		var reportID *string
		if rep != nil {
			reportID, entry.ReportID, entry.Action = &rep.ID, rep.ID, "report.comment_added"
		}
		var err error
		if c, err = q.CreateComment(ctx, db.CreateCommentParams{ID: newID(), TaskID: t.ID, ReportID: reportID, AuthorUserID: &userID, Body: body}); err != nil {
			return err
		}
		return logAudit(ctx, q, entry)
	})
	return c, err
}
