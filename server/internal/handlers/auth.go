package handlers

import (
	"net/http"

	"server/internal/middleware"
	"server/internal/schemas"
	"server/internal/services"
)

type AuthHandler struct {
	svc          *services.AuthService
	cookieSecure bool
}

func NewAuthHandler(svc *services.AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{svc: svc, cookieSecure: cookieSecure}
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: middleware.CookieName, Value: token, Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req schemas.SignupRequest
	if !decode(w, r, &req) {
		return
	}
	user, token, err := h.svc.Signup(r.Context(), req)
	if err == nil {
		h.setSessionCookie(w, token, int(services.TokenTTL.Seconds()))
	}
	respond(w, http.StatusCreated, schemas.AuthResponse{User: user}, err)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req schemas.LoginRequest
	if !decode(w, r, &req) {
		return
	}
	user, token, err := h.svc.Login(r.Context(), req)
	if err == nil {
		h.setSessionCookie(w, token, int(services.TokenTTL.Seconds()))
	}
	respond(w, http.StatusOK, schemas.AuthResponse{User: user}, err)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.setSessionCookie(w, "", -1)
	writeJSON(w, http.StatusOK, schemas.MessageResponse{Status: "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.Me(r.Context(), uid(r))
	respond(w, http.StatusOK, schemas.AuthResponse{User: user}, err)
}
