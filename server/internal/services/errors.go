// Package services holds all business logic. Handlers only decode, call a service and encode.
package services

import (
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
)

// Error is what services return for anything the caller should see; handlers map Kind to an HTTP status.
// Any other error is treated as an internal error.
type Error struct {
	Kind Kind
	Msg  string
}

type Kind int

const (
	KindBadRequest Kind = iota + 1
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
)

func (e *Error) Error() string { return e.Msg }

func BadRequest(msg string) error   { return &Error{KindBadRequest, msg} }
func Unauthorized(msg string) error { return &Error{KindUnauthorized, msg} }
func Forbidden(msg string) error    { return &Error{KindForbidden, msg} }
func NotFound(msg string) error     { return &Error{KindNotFound, msg} }
func Conflict(msg string) error     { return &Error{KindConflict, msg} }

// notFoundOr turns sql.ErrNoRows into a NotFound(msg), leaving other errors as they are.
func notFoundOr(err error, msg string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return NotFound(msg)
	}
	return err
}

// uniqueViolation reports a SQLite UNIQUE / PRIMARY KEY violation and its message
// (e.g. "UNIQUE constraint failed: index 'idx_users_username_lower'").
func uniqueViolation(err error) (string, bool) {
	var se *sqlite.Error
	if errors.As(err, &se) {
		if c := se.Code(); c == 2067 || c == 1555 {
			return se.Error(), true
		}
	}
	return "", false
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return fmt.Sprintf("%s…", string(r[:n]))
}
