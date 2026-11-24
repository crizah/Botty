package handlers

import (
	"net/http"

	"server/internal/schemas"
	"server/internal/services"
)

type TaskHandler struct{ svc *services.TaskService }

func NewTaskHandler(svc *services.TaskService) *TaskHandler { return &TaskHandler{svc} }

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context(), uid(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req schemas.CreateTaskRequest
	if !decode(w, r, &req) {
		return
	}
	t, err := h.svc.Create(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusCreated, schemas.TaskResponse{Task: t}, err)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Get(r.Context(), uid(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req schemas.UpdateTaskRequest
	if !decode(w, r, &req) {
		return
	}
	t, err := h.svc.Update(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusOK, schemas.TaskResponse{Task: t}, err)
}

func (h *TaskHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	var req schemas.StatusRequest
	if !decode(w, r, &req) {
		return
	}
	out, err := h.svc.SetStatus(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusOK, out, err)
}

func (h *TaskHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	var req schemas.CreateReportRequest
	if !decode(w, r, &req) {
		return
	}
	rep, err := h.svc.CreateReport(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusCreated, schemas.ReportResponse{Report: rep}, err)
}

func (h *TaskHandler) SetReportStatus(w http.ResponseWriter, r *http.Request) {
	var req schemas.StatusRequest
	if !decode(w, r, &req) {
		return
	}
	out, err := h.svc.SetReportStatus(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusOK, out, err)
}

func (h *TaskHandler) CommentOnTask(w http.ResponseWriter, r *http.Request) {
	var req schemas.CommentRequest
	if !decode(w, r, &req) {
		return
	}
	c, err := h.svc.CommentOnTask(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusCreated, schemas.CommentResponse{Comment: c}, err)
}

func (h *TaskHandler) CommentOnReport(w http.ResponseWriter, r *http.Request) {
	var req schemas.CommentRequest
	if !decode(w, r, &req) {
		return
	}
	c, err := h.svc.CommentOnReport(r.Context(), uid(r), r.PathValue("id"), req)
	respond(w, http.StatusCreated, schemas.CommentResponse{Comment: c}, err)
}
