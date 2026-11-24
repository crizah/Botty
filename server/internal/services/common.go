package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	db "server/internal/db/sqlc"
)

func newID() string { return uuid.NewString() }

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// withTx runs fn inside a transaction; any error (including a service Error) rolls it back.
func withTx(ctx context.Context, conn *sql.DB, q *db.Queries, fn func(q *db.Queries) error) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// ---- RBAC ----

const (
	levelViewer = 1
	levelEditor = 2
	levelOwner  = 3
)

var roleLevel = map[string]int{"viewer": levelViewer, "editor": levelEditor, "owner": levelOwner}

// requireProject checks the caller's role on a project and returns it. Non-members get a
// NotFound so project ids can't be probed; members below `min` get Forbidden.
func requireProject(ctx context.Context, q *db.Queries, projectID, userID string, min int) (role string, err error) {
	role, err = q.GetProjectRole(ctx, db.GetProjectRoleParams{ProjectID: projectID, UserID: userID})
	if err != nil {
		return "", notFoundOr(err, "project not found")
	}
	if roleLevel[role] < min {
		return "", Forbidden("you don't have permission to do this")
	}
	return role, nil
}

// requireGroupInProject returns NotFound unless the group is attached to the project.
func requireGroupInProject(ctx context.Context, q *db.Queries, projectID, groupID string) error {
	n, err := q.IsGroupInProject(ctx, db.IsGroupInProjectParams{ProjectID: projectID, GroupID: groupID})
	if err != nil {
		return err
	}
	if n == 0 {
		return NotFound("group is not part of this project")
	}
	return nil
}

// ---- audit log ----

type auditEntry struct {
	ProjectID string
	TaskID    string // optional
	ReportID  string // optional
	ActorUser string // optional; empty for bot/system
	ActorKind string // user | bot | system (default user)
	Action    string
	Old, New  string
	Detail    string
}

func logAudit(ctx context.Context, q *db.Queries, e auditEntry) error {
	if e.ActorKind == "" {
		e.ActorKind = "user"
	}
	return q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		ID:          newID(),
		ProjectID:   e.ProjectID,
		TaskID:      strPtr(e.TaskID),
		ReportID:    strPtr(e.ReportID),
		ActorUserID: strPtr(e.ActorUser),
		ActorKind:   e.ActorKind,
		Action:      e.Action,
		OldValue:    e.Old,
		NewValue:    e.New,
		Detail:      e.Detail,
	})
}

func memberLabel(m db.WaGroupMember) string {
	switch {
	case m.PushName != nil && *m.PushName != "":
		return *m.PushName
	case m.Phone != nil && *m.Phone != "":
		return *m.Phone
	}
	return m.WaID
}
