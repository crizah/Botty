package handlers

import (
	"net/http"
	"strconv"

	"server/internal/schemas"
	"server/internal/services"
)

type ProjectHandler struct{ svc *services.ProjectService }

func NewProjectHandler(svc *services.ProjectService) *ProjectHandler { return &ProjectHandler{svc} }

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context(), uid(r))
	respond(w, http.StatusOK, out, err)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req schemas.CreateProjectRequest
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Create(r.Context(), uid(r), req)
	respond(w, http.StatusCreated, schemas.ProjectResponse{Project: p}, err)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Get(r.Context(), uid(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req schemas.UpdateProjectRequest
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Update(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusOK, schemas.ProjectResponse{Project: p}, err)
}

func (h *ProjectHandler) Audit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := h.svc.Audit(r.Context(), uid(r), r.PathValue("id"), limit)
	respond(w, http.StatusOK, out, err)
}
