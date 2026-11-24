package schemas

import (
	"errors"
	netmail "net/mail"
	"strings"

	db "server/internal/db/sqlc"
)

// --- Requests ---

type InviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (r *InviteRequest) Validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if a, err := netmail.ParseAddress(r.Email); err != nil || a.Address != r.Email || len(r.Email) > 254 {
		return errors.New("enter a valid email address")
	}
	if r.Role != "editor" && r.Role != "viewer" {
		return errors.New("role must be editor or viewer")
	}
	return nil
}

type MemberRoleRequest struct {
	Role string `json:"role"`
}

func (r *MemberRoleRequest) Validate() error {
	if r.Role != "editor" && r.Role != "viewer" {
		return errors.New("role must be editor or viewer")
	}
	return nil
}

type AcceptInviteRequest struct {
	Token string `json:"token"`
}

func (r *AcceptInviteRequest) Validate() error {
	if strings.TrimSpace(r.Token) == "" {
		return errors.New("token is required")
	}
	return nil
}

// --- Responses ---

// ProjectUsersResponse lists the project's users; pending invites are only filled in for the owner.
type ProjectUsersResponse struct {
	Members []db.ListProjectMembersRow `json:"members"`
	Invites []db.ListPendingInvitesRow `json:"invites"`
}

type InviteResponse struct {
	Invite db.ProjectInvite `json:"invite"`
}

type InvitePreviewResponse struct {
	ProjectName string `json:"project_name"`
	InvitedBy   string `json:"invited_by"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	HasAccount  bool   `json:"has_account"`
}

type AcceptInviteResponse struct {
	ProjectID string `json:"project_id"`
}
