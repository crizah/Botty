// Package handlers is the thin HTTP layer: decode -> call a service -> encode.
// No business logic and no database access lives here.
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"server/internal/middleware"
	"server/internal/schemas"
	"server/internal/services"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// respond writes body with status, or the mapped error response when err != nil.
func respond(w http.ResponseWriter, status int, body any, err error) {
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, status, body)
}

// handleError maps a service error to an HTTP status; anything unexpected is a logged 500.
func handleError(w http.ResponseWriter, err error) {
	var se *services.Error
	if !errors.As(err, &se) {
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	status := map[services.Kind]int{
		services.KindBadRequest:   http.StatusBadRequest,
		services.KindUnauthorized: http.StatusUnauthorized,
		services.KindForbidden:    http.StatusForbidden,
		services.KindNotFound:     http.StatusNotFound,
		services.KindConflict:     http.StatusConflict,
	}[se.Kind]
	writeError(w, status, se.Msg)
}

// decode reads a JSON body (max 1MB) and runs the schema's Validate method if it has one.
// It writes the 400 itself and returns false on failure.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if val, ok := v.(schemas.Validator); ok {
		if err := val.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return false
		}
	}
	return true
}

// uid is the authenticated user's id, set by middleware.RequireAuth.
func uid(r *http.Request) string { return middleware.UserID(r.Context()) }
