package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	Email string `json:"email"`
	Typ   string `json:"typ,omitempty"` // empty for access tokens; invite tokens set "invite"
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID, email string, secret []byte, ttl time.Duration) (string, error) {
	claims := AccessClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ParseAccessToken returns the user id and email from a valid token.
func ParseAccessToken(tokenStr string, secret []byte) (userID, email string, err error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return "", "", err
	}
	if !token.Valid || claims.Subject == "" || claims.Typ != "" {
		return "", "", errors.New("invalid token")
	}
	return claims.Subject, claims.Email, nil
}

type inviteClaims struct {
	Typ string `json:"typ"`
	jwt.RegisteredClaims
}

// GenerateInviteToken signs a link token that carries only the invite id (the Subject).
func GenerateInviteToken(inviteID string, secret []byte, ttl time.Duration) (string, error) {
	c := inviteClaims{Typ: "invite", RegisteredClaims: jwt.RegisteredClaims{
		Subject: inviteID, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret)
}

// ParseInviteToken returns the invite id from a valid, unexpired invite token.
func ParseInviteToken(tokenStr string, secret []byte) (string, error) {
	c := &inviteClaims{}
	t, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !t.Valid || c.Typ != "invite" || c.Subject == "" {
		return "", errors.New("invalid invite token")
	}
	return c.Subject, nil
}
