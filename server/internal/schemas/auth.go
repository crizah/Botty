// Package schemas holds every request and response body. Requests validate (and
// normalise) themselves in Validate(); the handlers run it right after decoding.
package schemas

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
)

// Validator is implemented by request schemas that need checking.
type Validator interface{ Validate() error }

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,32}$`)

func normEmail(e *string) error {
	*e = strings.ToLower(strings.TrimSpace(*e))
	addr, err := mail.ParseAddress(*e)
	if err != nil || addr.Address != *e || len(*e) > 254 {
		return errors.New("enter a valid email")
	}
	return nil
}

// --- Requests ---

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"` // optional
}

func (r *SignupRequest) Validate() error {
	if err := normEmail(&r.Email); err != nil {
		return err
	}
	if len(r.Password) < 8 || len(r.Password) > 72 { // 72 = bcrypt's limit
		return errors.New("password must be 8-72 characters")
	}
	r.Username = strings.TrimSpace(r.Username)
	if r.Username != "" && !usernameRe.MatchString(r.Username) {
		return errors.New("username must be 3-32 characters: letters, numbers, . _ -")
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if err := normEmail(&r.Email); err != nil {
		return err
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// --- Responses ---

type UserResponse struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Username *string `json:"username"`
}

type AuthResponse struct {
	User UserResponse `json:"user"`
}

// MessageResponse is the generic {"status": "..."} body.
type MessageResponse struct {
	Status string `json:"status"`
}
