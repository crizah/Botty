package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"

	db "server/internal/db/sqlc"
	"server/internal/schemas"
)

// Crockford-style base32 without I, L, O, U so codes are easy to read and type.
const codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
const codeLen = 8
const codePrefix = "BOTTY-"

// ClaimService links a WhatsApp group to a dashboard user: the user generates a code,
// then sends "@botty link <code>" in the group.
type ClaimService struct {
	db      *sql.DB
	Queries *db.Queries
}

func NewClaimService(dbConn *sql.DB) *ClaimService {
	return &ClaimService{db: dbConn, Queries: db.New(dbConn)}
}

func generateCode() (string, error) {
	b := make([]byte, codeLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)] // 256 % 32 == 0, so no modulo bias
	}
	return string(b), nil
}

// NormalizeCode accepts what a human might type: any case, with or without the prefix/dashes/spaces,
// and maps look-alikes (O->0, I/L->1).
func NormalizeCode(in string) string {
	s := strings.ToUpper(in)
	s = strings.NewReplacer("-", "", " ", "").Replace(s)
	s = strings.TrimPrefix(s, "BOTTY")
	s = strings.NewReplacer("O", "0", "I", "1", "L", "1").Replace(s)
	return s
}

// Create makes a fresh single-use code for the user, valid for 30 minutes.
func (s *ClaimService) Create(ctx context.Context, userID string) (schemas.ClaimResponse, error) {
	for range 5 { // retry on the (very unlikely) unique collision
		code, err := generateCode()
		if err != nil {
			return schemas.ClaimResponse{}, err
		}
		c, err := s.Queries.CreateClaim(ctx, db.CreateClaimParams{ID: newID(), Code: code, UserID: userID})
		if err != nil {
			if _, dup := uniqueViolation(err); dup {
				continue
			}
			return schemas.ClaimResponse{}, err
		}
		return schemas.ClaimResponse{Code: codePrefix + c.Code, Command: "@botty link " + codePrefix + c.Code, ExpiresAt: c.ExpiresAt}, nil
	}
	return schemas.ClaimResponse{}, errors.New("could not generate a unique code")
}

// Redeem is called by the bot when it sees "@botty link <code>" in a group.
// It creates the group owned by the code's user and burns the code.
// TODO: change - the ingest worker calls this with the real group JID/name; today only the dev endpoint does.
func (s *ClaimService) Redeem(ctx context.Context, rawCode, jid, name string) (db.WaGroup, error) {
	code := NormalizeCode(rawCode)
	var g db.WaGroup
	err := withTx(ctx, s.db, s.Queries, func(q *db.Queries) error {
		c, err := q.GetOpenClaimByCode(ctx, code)
		if err != nil {
			return notFoundOr(err, "that code is invalid or expired")
		}
		if _, err := q.GetGroupByJID(ctx, jid); err == nil {
			return Conflict("this group is already linked")
		}
		if g, err = q.CreateGroup(ctx, db.CreateGroupParams{ID: newID(), Jid: jid, Name: name, OwnerUserID: c.UserID}); err != nil {
			return err
		}
		return q.MarkClaimUsed(ctx, db.MarkClaimUsedParams{GroupID: &g.ID, ID: c.ID})
	})
	return g, err
}
