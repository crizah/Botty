package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"server/internal/util"
)

const CookieName = "access_token"

type ctxKey struct{}

// UserID returns the authenticated user's id (set by RequireAuth).
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// RequireAuth validates the JWT from the httpOnly cookie, falling back to the
// Authorization: Bearer header, and stores the user id on the request context.
func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if c, err := r.Cookie(CookieName); err == nil {
				token = c.Value
			} else if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				token = strings.TrimPrefix(h, "Bearer ")
			}
			if token == "" {
				unauthorized(w, "missing token")
				return
			}
			userID, _, err := util.ParseAccessToken(token, secret)
			if err != nil {
				unauthorized(w, "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, userID)))
		})
	}
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
