package schemas

import (
	"errors"
	"strings"

	db "server/internal/db/sqlc"
)

// --- Requests ---

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *CreateProjectRequest) Validate() error {
	r.Name, r.Description = strings.TrimSpace(r.Name), strings.TrimSpace(r.Description)
	if r.Name == "" || len(r.Name) > 100 {
		return errors.New("project name is required (max 100 characters)")
	}
	if len(r.Description) > 2000 {
		return errors.New("description is too long")
	}
	return nil
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	ListenMode  *string `json:"listen_mode"`
}

func (r *UpdateProjectRequest) Validate() error {
	if r.ListenMode != nil && *r.ListenMode != "bot_only" && *r.ListenMode != "all_texts" {
		return errors.New("listen_mode must be bot_only or all_texts")
	}
	if r.Name != nil {
		n := strings.TrimSpace(*r.Name)
		if n == "" || len(n) > 100 {
			return errors.New("project name is required (max 100 characters)")
		}
		r.Name = &n
	}
	return nil
}

type AttachGroupRequest struct {
	GroupID string `json:"group_id"`
}

func (r *AttachGroupRequest) Validate() error {
	if r.GroupID == "" {
		return errors.New("group_id is required")
	}
	return nil
}

type AllowlistRequest struct {
	MemberID string `json:"member_id"`
}

func (r *AllowlistRequest) Validate() error {
	if r.MemberID == "" {
		return errors.New("member_id is required")
	}
	return nil
}

// --- Responses ---

type ProjectsResponse struct {
	Projects []db.ListProjectsForUserRow `json:"projects"`
}

type ProjectResponse struct {
	Project db.Project `json:"project"`
}

type ProjectDetailResponse struct {
	Project db.Project `json:"project"`
	Role    string     `json:"role"`
}

type ProjectAuditResponse struct {
	Entries []db.ListAuditByProjectRow `json:"entries"`
}
