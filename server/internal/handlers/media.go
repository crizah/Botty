package handlers

import (
	"net/http"
	"time"

	"server/internal/services"
)

type MediaHandler struct{ svc *services.MediaService }

func NewMediaHandler(svc *services.MediaService) *MediaHandler { return &MediaHandler{svc} }

// Get streams a report image to a project member.
func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	f, mime, err := h.svc.Open(r.Context(), uid(r), r.PathValue("id"))
	if err != nil {
		handleError(w, err)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeContent(w, r, "", time.Time{}, f)
}
