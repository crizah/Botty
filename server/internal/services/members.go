package services

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	db "server/internal/db/sqlc"
	"server/internal/mail"
	"server/internal/schemas"
	"server/internal/util"
)

const inviteTTL = 48 * time.Hour

// MemberService manages who is in a project: listing, email invites, role changes, removal.
// Only the project owner can invite, change roles or remove people.
type MemberService struct {
	db          *sql.DB
	Queries     *db.Queries
	secret      []byte
	mailer      mail.Sender
	frontendURL string
}

func NewMemberService(dbConn *sql.DB, secret []byte, mailer mail.Sender, frontendURL string) *MemberService {
	return &MemberService{db: dbConn, Queries: db.New(dbConn), secret: secret, mailer: mailer, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func userName(u db.User) string {
	if u.Username != nil && *u.Username != "" {
		return *u.Username
	}
	return u.Email
}

// List shows members to every project member; pending invites (emails) only to the owner.
func (s *MemberService) List(ctx context.Context, userID, projectID string) (schemas.ProjectUsersResponse, error) {
	role, err := requireProject(ctx, s.Queries, projectID, userID, levelViewer)
	if err != nil {
		return schemas.ProjectUsersResponse{}, err
	}
	out := schemas.ProjectUsersResponse{}
	if out.Members, err = s.Queries.ListProjectMembers(ctx, projectID); err != nil {
		return out, err
	}
	if role == "owner" {
		out.Invites, err = s.Queries.ListPendingInvites(ctx, projectID)
	}
	if out.Members == nil {
		out.Members = []db.ListProjectMembersRow{}
	}
	if out.Invites == nil {
		out.Invites = []db.ListPendingInvitesRow{}
	}
	return out, err
}

// Invite creates a pending invite (replacing any earlier pending one for that email) and emails the link.
func (s *MemberService) Invite(ctx context.Context, userID, projectID string, req schemas.InviteRequest) (schemas.InviteResponse, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelOwner); err != nil {
		return schemas.InviteResponse{}, err
	}
	if u, err := s.Queries.GetUserByEmail(ctx, req.Email); err == nil {
		if _, err := s.Queries.GetProjectMember(ctx, db.GetProjectMemberParams{ProjectID: projectID, UserID: u.ID}); err == nil {
			return schemas.InviteResponse{}, Conflict("that person is already in this project")
		}
	}
	var inv db.ProjectInvite
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) (err error) {
		if err = q.RevokePendingInvitesForEmail(ctx, db.RevokePendingInvitesForEmailParams{ProjectID: projectID, Email: req.Email}); err != nil {
			return err
		}
		if inv, err = q.CreateInvite(ctx, db.CreateInviteParams{ID: newID(), ProjectID: projectID, Email: req.Email, Role: req.Role, InvitedBy: userID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "member.invited", New: req.Role, Detail: req.Email})
	})
	if err != nil {
		return schemas.InviteResponse{}, err
	}
	return s.deliver(ctx, userID, inv)
}

// Resend gives a pending invite a fresh 48 hours and a new link, and emails it again.
func (s *MemberService) Resend(ctx context.Context, userID, projectID, inviteID string) (schemas.InviteResponse, error) {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelOwner); err != nil {
		return schemas.InviteResponse{}, err
	}
	inv, err := s.Queries.RefreshInvite(ctx, db.RefreshInviteParams{ID: inviteID, ProjectID: projectID})
	if err != nil {
		return schemas.InviteResponse{}, notFoundOr(err, "invite not found or already used")
	}
	return s.deliver(ctx, userID, inv)
}

// deliver signs the link and sends the email from a goroutine, so the request doesn't wait on Resend.
func (s *MemberService) deliver(ctx context.Context, inviterID string, inv db.ProjectInvite) (schemas.InviteResponse, error) {
	token, err := util.GenerateInviteToken(inv.ID, s.secret, inviteTTL)
	if err != nil {
		return schemas.InviteResponse{}, err
	}
	link := s.frontendURL + "/invite?token=" + token
	p, err := s.Queries.GetProject(ctx, inv.ProjectID)
	if err != nil {
		return schemas.InviteResponse{}, err
	}
	inviter, err := s.Queries.GetUserByID(ctx, inviterID)
	if err != nil {
		return schemas.InviteResponse{}, err
	}
	label := "a viewer (can view tasks)"
	if inv.Role == "editor" {
		label = "an editor (can view and update tasks)"
	}
	msg := mail.InviteEmail(inv.Email, userName(inviter), p.Name, label, link)
	go func() {
		sctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.mailer.Send(sctx, msg); err != nil {
			log.Printf("[mail] invite to %s failed: %v", inv.Email, err) // owner can hit Resend
		}
	}()
	return schemas.InviteResponse{Invite: inv}, nil
}

func (s *MemberService) Revoke(ctx context.Context, userID, projectID, inviteID string) error {
	if _, err := requireProject(ctx, s.Queries, projectID, userID, levelOwner); err != nil {
		return err
	}
	return withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		inv, err := q.GetInvite(ctx, inviteID)
		if err != nil || inv.ProjectID != projectID {
			return NotFound("invite not found")
		}
		n, err := q.RevokeInvite(ctx, db.RevokeInviteParams{ID: inviteID, ProjectID: projectID})
		if err != nil {
			return err
		}
		if n == 0 {
			return NotFound("invite not found or already used")
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "member.invite_revoked", Detail: inv.Email})
	})
}

// target checks the owner may change/remove this member: it must exist, not be the owner, not be the caller.
func (s *MemberService) target(ctx context.Context, q *db.Queries, userID, projectID, targetID string) (db.User, db.ProjectMember, error) {
	if _, err := requireProject(ctx, q, projectID, userID, levelOwner); err != nil {
		return db.User{}, db.ProjectMember{}, err
	}
	m, err := q.GetProjectMember(ctx, db.GetProjectMemberParams{ProjectID: projectID, UserID: targetID})
	if err != nil {
		return db.User{}, db.ProjectMember{}, notFoundOr(err, "member not found")
	}
	if m.Role == "owner" || targetID == userID {
		return db.User{}, db.ProjectMember{}, Forbidden("the owner's access can't be changed or removed")
	}
	u, err := q.GetUserByID(ctx, targetID)
	return u, m, err
}

// SetRole switches a member between viewer (read) and editor (write). Owner only.
func (s *MemberService) SetRole(ctx context.Context, userID, projectID, targetID string, req schemas.MemberRoleRequest) error {
	return withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		u, m, err := s.target(ctx, q, userID, projectID, targetID)
		if err != nil {
			return err
		}
		if m.Role == req.Role {
			return nil
		}
		if _, err := q.SetProjectMemberRole(ctx, db.SetProjectMemberRoleParams{Role: req.Role, ProjectID: projectID, UserID: targetID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "member.role_changed", Old: m.Role, New: req.Role, Detail: userName(u)})
	})
}

// Remove takes someone out of the project. Owner only.
func (s *MemberService) Remove(ctx context.Context, userID, projectID, targetID string) error {
	return withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		u, _, err := s.target(ctx, q, userID, projectID, targetID)
		if err != nil {
			return err
		}
		if _, err := q.RemoveProjectMember(ctx, db.RemoveProjectMemberParams{ProjectID: projectID, UserID: targetID}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: projectID, ActorUser: userID, Action: "member.removed", Detail: userName(u)})
	})
}

// openInvite resolves a link token to a still-usable invite.
func (s *MemberService) openInvite(ctx context.Context, q *db.Queries, token string) (db.ProjectInvite, error) {
	id, err := util.ParseInviteToken(token, s.secret)
	if err != nil {
		return db.ProjectInvite{}, NotFound("this invite link is invalid or has expired")
	}
	inv, err := q.GetOpenInvite(ctx, id)
	if err != nil {
		return db.ProjectInvite{}, notFoundOr(err, "this invite link is invalid or has expired")
	}
	return inv, nil
}

// Preview tells the invite page who invited whom to what, and whether the email already has an account.
// Public: holding the emailed token is the proof.
func (s *MemberService) Preview(ctx context.Context, token string) (schemas.InvitePreviewResponse, error) {
	inv, err := s.openInvite(ctx, s.Queries, token)
	if err != nil {
		return schemas.InvitePreviewResponse{}, err
	}
	p, err := s.Queries.GetProject(ctx, inv.ProjectID)
	if err != nil {
		return schemas.InvitePreviewResponse{}, err
	}
	inviter, err := s.Queries.GetUserByID(ctx, inv.InvitedBy)
	if err != nil {
		return schemas.InvitePreviewResponse{}, err
	}
	_, accErr := s.Queries.GetUserByEmail(ctx, inv.Email)
	return schemas.InvitePreviewResponse{ProjectName: p.Name, InvitedBy: userName(inviter), Email: inv.Email, Role: inv.Role, HasAccount: accErr == nil}, nil
}

// Accept adds the signed-in user to the project. Their account email must match the invited email,
// and the invite is burned in the same transaction, so a link works once.
func (s *MemberService) Accept(ctx context.Context, userID string, req schemas.AcceptInviteRequest) (schemas.AcceptInviteResponse, error) {
	var out schemas.AcceptInviteResponse
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		inv, err := s.openInvite(ctx, q, req.Token)
		if err != nil {
			return err
		}
		u, err := q.GetUserByID(ctx, userID)
		if err != nil {
			return notFoundOr(err, "user not found")
		}
		if !strings.EqualFold(u.Email, inv.Email) {
			return Forbidden("this invite was sent to " + inv.Email + ". Log in with that email to accept it.")
		}
		if n, err := q.AcceptInvite(ctx, inv.ID); err != nil {
			return err
		} else if n == 0 {
			return NotFound("this invite link is invalid or has expired")
		}
		out.ProjectID = inv.ProjectID
		if _, err := q.GetProjectMember(ctx, db.GetProjectMemberParams{ProjectID: inv.ProjectID, UserID: userID}); err == nil {
			return nil // already in the project: keep their current role
		}
		if err := q.AddProjectMember(ctx, db.AddProjectMemberParams{ProjectID: inv.ProjectID, UserID: userID, Role: inv.Role}); err != nil {
			return err
		}
		return logAudit(ctx, q, auditEntry{ProjectID: inv.ProjectID, ActorUser: userID, Action: "member.joined", New: inv.Role, Detail: userName(u)})
	})
	return out, err
}
