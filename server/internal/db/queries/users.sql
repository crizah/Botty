-- name: CreateUser :one
INSERT INTO users (id, email, username, password_hash)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE lower(email) = lower(?);

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;
