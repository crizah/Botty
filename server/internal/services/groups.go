package services

import (
	"context"
	"database/sql"
	"fmt"

	db "server/internal/db/sqlc"
	"server/internal/schemas"
)

// GroupService covers the WhatsApp groups the bot is in, attaching them to projects,
// and the per-group tag allowlist.
type GroupService struct {
	db        *sql.DB
	Queries   *db.Queries
	botNumber string
	devMode   bool
}

func NewGroupService(dbConn *sql.DB, botNumber string, devMode bool) *GroupService {
	return &GroupService{db: dbConn, Queries: db.New(dbConn), botNumber: botNumber, devMode: devMode}
}

func (s *GroupService) BotInfo() schemas.BotInfoResponse {
	return schemas.BotInfoResponse{Number: s.botNumber, DevMode: s.devMode}
}

func (s *GroupService) DevMode() bool { return s.devMode }

// ListMine returns the groups the bot was added to on this user's behalf, with their project (if any).
func (s *GroupService) ListMine(ctx context.Context, userID string) (schemas.GroupsResponse, error) {
	rows, err := s.Queries.ListGroupsForOwner(ctx, userID)
	if rows == nil {
		rows = []db.ListGroupsForOwnerRow{}
	}
	return schemas.GroupsResponse{Groups: rows}, err
}

// DevCreate fakes "the bot got added to a group" so the dashboard works without WhatsApp.
// TODO: change - delete once the ingest worker discovers real groups.
func (s *GroupService) DevCreate(ctx context.Context, userID string, req schemas.DevCreateGroupRequest) (db.WaGroup, error) {
	if !s.devMode {
		return db.WaGroup{}, NotFound("not found")
	}
	if req.Name == "" {
		req.Name = "Test group"
	}
	if len(req.Members) == 0 {
		req.Members = []string{"Asha", "Ben", "Chitra", "Dev"}
	}
	var g db.WaGroup
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) (err error) {
		if g, err = q.CreateGroup(ctx, db.CreateGroupParams{ID: newID(), Jid: fmt.Sprintf("dev-%s@g.us", newID()[:8]), Name: req.Name, OwnerUserID: userID}); err != nil {
			return err
		}
		for i, name := range req.Members {
			phone, n := fmt.Sprintf("+91 90000 0%04d", i), name // fake numbers
			if _, err := q.UpsertMember(ctx, db.UpsertMemberParams{ID: newID(), GroupID: g.ID, WaID: fmt.Sprintf("dev-%d@s.whatsapp.net", i), Phone: &phone, PushName: &n}); err != nil {
				return err
			}
		}
		return nil
	})
	return g, err
}

// ---- groups inside a project ----

func (s *GroupService) ListForProject(ctx context.Context, userID, projectID string) (schemas.ProjectGroupsResponse, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelViewer); err != nil {
		return schemas.ProjectGroupsResponse{}, err
	}
	rows, err := s.Queries.ListProjectGroups(ctx, projectID)
	if rows == nil {
		rows = []db.ListProjectGroupsRow{}
	}
	return schemas.ProjectGroupsResponse{Groups: rows}, err
}

// Attach puts one of the caller's groups into the project (a group can be in one project only).
func (s *GroupService) Attach(ctx context.Context, userID, projectID string, req schemas.AttachGroupRequest) error {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelOwner); err != nil {
		return err
	}
	g, err := s.Queries.GetGroup(ctx, req.GroupID)
	if err != nil {
		return notFoundOr(err, "group not found")
	}
	if g.OwnerUserID != userID { // only groups the caller added the bot to
		return NotFound("group not found")
	}
	return withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		if err := q.AttachGroup(ctx, db.AttachGroupParams{ProjectID: projectID, GroupID: g.ID}); err != nil {
			if _, dup := uniqueViolation(err); dup {
				return Conflict("that group is already attached to a project")
			}
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "project.group_added", Detail: g.Name})
	})
}

func (s *GroupService) Detach(ctx context.Context, userID, projectID, groupID string) error {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelOwner); err != nil {
		return err
	}
	g, err := s.Queries.GetGroup(ctx, groupID)
	if err != nil {
		return notFoundOr(err, "group not found")
	}
	return withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		if err := q.DetachGroup(ctx, db.DetachGroupParams{ProjectID: projectID, GroupID: groupID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "project.group_removed", Detail: g.Name})
	})
}

func (s *GroupService) Members(ctx context.Context, userID, projectID, groupID string) (schemas.MembersResponse, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelViewer); err != nil {
		return schemas.MembersResponse{}, err
	}
	if err := requireGroupInProject(ctx, s.Queries, projectID, groupID); err != nil {
		return schemas.MembersResponse{}, err
	}
	rows, err := s.Queries.ListMembersByGroup(ctx, groupID)
	if rows == nil {
		rows = []db.WaGroupMember{}
	}
	return schemas.MembersResponse{Members: rows}, err
}

// ---- who may tag the bot (per project AND group) ----

func (s *GroupService) Allowlist(ctx context.Context, userID, projectID, groupID string) (schemas.AllowlistResponse, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelViewer); err != nil {
		return schemas.AllowlistResponse{}, err
	}
	if err := requireGroupInProject(ctx, s.Queries, projectID, groupID); err != nil {
		return schemas.AllowlistResponse{}, err
	}
	rows, err := s.Queries.ListAllowlist(ctx, db.ListAllowlistParams{ProjectID: projectID, GroupID: groupID})
	if rows == nil {
		rows = []db.ListAllowlistRow{}
	}
	return schemas.AllowlistResponse{Members: rows}, err
}

func (s *GroupService) AllowlistAdd(ctx context.Context, userID, projectID, groupID string, req schemas.AllowlistRequest) error {
	m, err := s.allowlistTarget(ctx, userID, projectID, groupID, req.MemberID)
	if err != nil {
		return err
	}
	return withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		if err := q.AddAllowlist(ctx, db.AddAllowlistParams{ProjectID: projectID, GroupID: groupID, MemberID: m.ID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "allowlist.added", Detail: memberLabel(m)})
	})
}

func (s *GroupService) AllowlistRemove(ctx context.Context, userID, projectID, groupID, memberID string) error {
	m, err := s.allowlistTarget(ctx, userID, projectID, groupID, memberID)
	if err != nil {
		return err
	}
	return withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		if err := q.RemoveAllowlist(ctx, db.RemoveAllowlistParams{ProjectID: projectID, GroupID: groupID, MemberID: memberID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "allowlist.removed", Detail: memberLabel(m)})
	})
}

// allowlistTarget checks owner access, that the group is in the project and that the member is in the group.
func (s *GroupService) allowlistTarget(ctx context.Context, userID, projectID, groupID, memberID string) (db.WaGroupMember, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelOwner); err != nil {
		return db.WaGroupMember{}, err
	}
	if err := requireGroupInProject(ctx, s.Queries, projectID, groupID); err != nil {
		return db.WaGroupMember{}, err
	}
	m, err := s.Queries.GetMember(ctx, memberID)
	if err != nil {
		return m, notFoundOr(err, "member not found in that group")
	}
	if m.GroupID != groupID {
		return m, NotFound("member not found in that group")
	}
	return m, nil
}
