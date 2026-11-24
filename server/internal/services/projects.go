package services

import (
	"context"
	"database/sql"

	db "server/internal/db/sqlc"
	"server/internal/schemas"
)

type ProjectService struct {
	db      *sql.DB
	Queries *db.Queries
}

func NewProjectService(dbConn *sql.DB) *ProjectService {
	return &ProjectService{db: dbConn, Queries: db.New(dbConn)}
}

func (s *ProjectService) List(ctx context.Context, userID string) (schemas.ProjectsResponse, error) {
	rows, err := s.Queries.ListProjectsForUser(ctx, userID)
	if rows == nil {
		rows = []db.ListProjectsForUserRow{}
	}
	return schemas.ProjectsResponse{Projects: rows}, err
}

// Create makes the project and the caller its owner.
func (s *ProjectService) Create(ctx context.Context, userID string, req schemas.CreateProjectRequest) (db.Project, error) {
	var p db.Project
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) (err error) {
		if p, err = q.CreateProject(ctx, db.CreateProjectParams{ID: newID(), Name: req.Name, Description: req.Description, CreatedBy: userID}); err != nil {
			return err
		}
		if err = q.AddProjectMember(ctx, db.AddProjectMemberParams{ProjectID: p.ID, UserID: userID, Role: "owner"}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: p.ID, ActorUser: userID, Action: "project.created", Detail: p.Name})
	})
	return p, err
}

func (s *ProjectService) Get(ctx context.Context, userID, projectID string) (schemas.ProjectDetailResponse, error) {
	role, err := requireProject(ctx, s.Queries, projectID, userID, levelViewer)
	if err != nil {
		return schemas.ProjectDetailResponse{}, err
	}
	p, err := s.Queries.GetProject(ctx, projectID)
	return schemas.ProjectDetailResponse{Project: p, Role: role}, err
}

// Update changes name / description / listen mode (owner only) and logs what changed.
func (s *ProjectService) Update(ctx context.Context, userID, projectID string, req schemas.UpdateProjectRequest) (db.Project, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelOwner); err != nil {
		return db.Project{}, err
	}
	var p db.Project
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		before, err := q.GetProject(ctx, projectID)
		if err != nil {
			return err
		}
		if p, err = q.UpdateProject(ctx, db.UpdateProjectParams{ID: projectID, Name: req.Name, Description: req.Description, ListenMode: req.ListenMode}); err != nil {
			return err
		}
		if p.ListenMode != before.ListenMode {
			if err := logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "project.listen_mode_changed", Old: before.ListenMode, New: p.ListenMode}); err != nil {
				return err
			}
		}
		if p.Name != before.Name {
			return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "project.renamed", Old: before.Name, New: p.Name})
		}
		return nil
	})
	return p, err
}

// Audit returns the project-wide activity feed (newest first). limit <= 0 means the default.
func (s *ProjectService) Audit(ctx context.Context, userID, projectID string, limit int) (schemas.ProjectAuditResponse, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelViewer); err != nil {
		return schemas.ProjectAuditResponse{}, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.Queries.ListAuditByProject(ctx, db.ListAuditByProjectParams{ProjectID: projectID, Limit: int64(limit)})
	if rows == nil {
		rows = []db.ListAuditByProjectRow{}
	}
	return schemas.ProjectAuditResponse{Entries: rows}, err
}
