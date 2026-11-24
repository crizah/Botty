package handlers

import (
	"net/http"

	"server/internal/schemas"
	"server/internal/services"
)

type MemberHandler struct{ svc *services.MemberService }

func NewMemberHandler(svc *services.MemberService) *MemberHandler { return &MemberHandler{svc} }

func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context(), uid(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *MemberHandler) Invite(w http.ResponseWriter, r *http.Request) {
	var req schemas.InviteRequest
	if !decode(w, r, &req) {
		return
	}
	out, err := h.svc.Invite(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusCreated, out, err)
}

func (h *MemberHandler) Resend(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Resend(r.Context(), uid(r), r.PathValue("id"), r.PathValue("iid"))
	respond(w, http.StatusOK, out, err)
}

func (h *MemberHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Revoke(r.Context(), uid(r), r.PathValue("id"), r.PathValue("iid"))
	respond(w, http.StatusOK, schemas.MessageResponse{Status: "revoked"}, err)
}

func (h *MemberHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	var req schemas.MemberRoleRequest
	if !decode(w, r, &req) {
		return
	}
	err := h.svc.SetRole(r.Context(), uid(r), r.PathValue("id"), r.PathValue("uid"), req)
	respond(w, http.StatusOK, schemas.MessageResponse{Status: "updated"}, err)
}

func (h *MemberHandler) Remove(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Remove(r.Context(), uid(r), r.PathValue("id"), r.PathValue("uid"))
	respond(w, http.StatusOK, schemas.MessageResponse{Status: "removed"}, err)
}

// Preview is public: the emailed token is the credential.
func (h *MemberHandler) Preview(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Preview(r.Context(), r.URL.Query().Get("token"))
	respond(w, http.StatusOK, out, err)
}

func (h *MemberHandler) Accept(w http.ResponseWriter, r *http.Request) {
	var req schemas.AcceptInviteRequest
	if !decode(w, r, &req) {
		return
	}
	out, err := h.svc.Accept(r.Context(), uid(r), req)
	respond(w, http.StatusOK, out, err)
}
