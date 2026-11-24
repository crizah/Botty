package handlers

import (
	"net/http"

	"server/internal/schemas"
	"server/internal/services"
)

type GroupHandler struct{ svc *services.GroupService }

func NewGroupHandler(svc *services.GroupService) *GroupHandler { return &GroupHandler{svc} }

func (h *GroupHandler) BotInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.BotInfo())
}

func (h *GroupHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListMine(r.Context(), uid(r))
	respond(w, http.StatusOK, out, err)
}

// DevCreate: TODO: change - dev only, removed together with the service method.
func (h *GroupHandler) DevCreate(w http.ResponseWriter, r *http.Request) {
	var req schemas.DevCreateGroupRequest
	if !decode(w, r, &req) {
		return
	}
	g, err := h.svc.DevCreate(r.Context(), uid(r), req)
	respond(w, http.StatusCreated, schemas.GroupResponse{Group: g}, err)
}

func (h *GroupHandler) ListForProject(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListForProject(r.Context(), uid(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *GroupHandler) Attach(w http.ResponseWriter, r *http.Request) {
	var req schemas.AttachGroupRequest
	if !decode(w, r, &req) {
		return
	}
	err := h.svc.Attach(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusCreated, schemas.MessageResponse{Status: "attached"}, err)
}

func (h *GroupHandler) Detach(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Detach(r.Context(), uid(r), r.PathValue("id"), r.PathValue("gid"))
	respond(w, http.StatusOK, schemas.MessageResponse{Status: "detached"}, err)
}

func (h *GroupHandler) Members(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Members(r.Context(), uid(r), r.PathValue("id"), r.PathValue("gid"))
	respond(w, http.StatusOK, out, err)
}

func (h *GroupHandler) Allowlist(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Allowlist(r.Context(), uid(r), r.PathValue("id"), r.PathValue("gid"))
	respond(w, http.StatusOK, out, err)
}

func (h *GroupHandler) AllowlistAdd(w http.ResponseWriter, r *http.Request) {
	var req schemas.AllowlistRequest
	if !decode(w, r, &req) {
		return
	}
	err := h.svc.AllowlistAdd(r.Context(), uid(r), r.PathValue("id"), r.PathValue("gid"), req)
	respond(w, http.StatusCreated, schemas.MessageResponse{Status: "added"}, err)
}

func (h *GroupHandler) AllowlistRemove(w http.ResponseWriter, r *http.Request) {
	err := h.svc.AllowlistRemove(r.Context(), uid(r), r.PathValue("id"), r.PathValue("gid"), r.PathValue("mid"))
	respond(w, http.StatusOK, schemas.MessageResponse{Status: "removed"}, err)
}
