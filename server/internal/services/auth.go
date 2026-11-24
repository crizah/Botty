package services

import (
	"context"
	"database/sql"
	"strings"
	"time"

	db "server/internal/db/sqlc"
	"server/internal/schemas"
	"server/internal/util"
)

const TokenTTL = 7 * 24 * time.Hour

type AuthService struct {
	db        *sql.DB
	Queries   *db.Queries
	JwtSecret []byte
}

func NewAuthService(dbConn *sql.DB, jwtSecret []byte) *AuthService {
	return &AuthService{db: dbConn, Queries: db.New(dbConn), JwtSecret: jwtSecret}
}

func toUser(u db.User) schemas.UserResponse {
	return schemas.UserResponse{ID: u.ID, Email: u.Email, Username: u.Username}
}

// Signup creates the account and returns it with a fresh session token.
func (s *AuthService) Signup(ctx context.Context, req schemas.SignupRequest) (schemas.UserResponse, string, error) {
	hash, err := util.HashPassword(req.Password)
	if err != nil {
		return schemas.UserResponse{}, "", err
	}
	user, err := s.Queries.CreateUser(ctx, db.CreateUserParams{
		ID:           newID(),
		Email:        req.Email,
		Username:     strPtr(req.Username),
		PasswordHash: hash,
	})
	if err != nil {
		if msg, ok := uniqueViolation(err); ok {
			if strings.Contains(msg, "username") {
				return schemas.UserResponse{}, "", Conflict("that username is taken")
			}
			return schemas.UserResponse{}, "", Conflict("an account with that email already exists")
		}
		return schemas.UserResponse{}, "", err
	}
	token, err := util.GenerateAccessToken(user.ID, user.Email, s.JwtSecret, TokenTTL)
	return toUser(user), token, err
}

// Login checks the credentials. Unknown email and wrong password give the same error.
func (s *AuthService) Login(ctx context.Context, req schemas.LoginRequest) (schemas.UserResponse, string, error) {
	user, err := s.Queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return schemas.UserResponse{}, "", Unauthorized("invalid email or password")
		}
		return schemas.UserResponse{}, "", err
	}
	if !util.CheckPassword(req.Password, user.PasswordHash) {
		return schemas.UserResponse{}, "", Unauthorized("invalid email or password")
	}
	token, err := util.GenerateAccessToken(user.ID, user.Email, s.JwtSecret, TokenTTL)
	return toUser(user), token, err
}

func (s *AuthService) Me(ctx context.Context, userID string) (schemas.UserResponse, error) {
	user, err := s.Queries.GetUserByID(ctx, userID)
	if err != nil {
		return schemas.UserResponse{}, notFoundOrUnauthorized(err)
	}
	return toUser(user), nil
}

func notFoundOrUnauthorized(err error) error {
	if err == sql.ErrNoRows {
		return Unauthorized("user no longer exists")
	}
	return err
}
