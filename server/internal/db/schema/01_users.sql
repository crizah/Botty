CREATE TABLE users (
    id            TEXT NOT NULL PRIMARY KEY,
    email         TEXT NOT NULL,
    -- optional; NULL when not provided
    username      TEXT,
    password_hash TEXT NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- uniqueness is enforced here in the db layer, case-insensitively ("Asha" == "asha").
-- lower() expression indexes because atlas drops COLLATE NOCASE on sqlite.
CREATE UNIQUE INDEX idx_users_email_lower ON users (lower(email));
CREATE UNIQUE INDEX idx_users_username_lower ON users (lower(username));
