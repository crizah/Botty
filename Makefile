DB_FILE       ?= server/internal/db/.data/botty.db
MIGRATIONS_DIR = server/internal/db/migrations
SCHEMA_DIR     = server/internal/db/schema

.PHONY: server web sqlc migrate-create migrate-up migrate-status migrate-hash

# run the API (loads server/.env)
server:
	cd server && go run ./cmd/api

web:
	cd web && npm start

# regenerate Go code from server/internal/db/queries + server/internal/db/schema
sqlc:
	cd server && sqlc generate

# diff schema/ against migrations/ and write a new migration
# usage: make migrate-create name=add_foo
migrate-create:
	atlas migrate diff $(name) \
		--dir "file://$(MIGRATIONS_DIR)" \
		--to "file://$(SCHEMA_DIR)" \
		--dev-url "sqlite://dev?mode=memory"

# apply pending migrations to the sqlite file
migrate-up:
	mkdir -p $(dir $(DB_FILE))
	atlas migrate apply \
		--dir "file://$(MIGRATIONS_DIR)" \
		--url "sqlite://$(DB_FILE)"

migrate-status:
	atlas migrate status \
		--dir "file://$(MIGRATIONS_DIR)" \
		--url "sqlite://$(DB_FILE)"

# fix atlas.sum after hand-editing a migration
migrate-hash:
	atlas migrate hash --dir "file://$(MIGRATIONS_DIR)"
