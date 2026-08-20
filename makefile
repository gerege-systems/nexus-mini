# nexus-mini
# Локал хөгжүүлэлт: `make dev-db` (нэг удаа) → `make migrate` → `make api`

PG_SUPER ?= postgres
DEV_PW   ?= dev

export DATABASE_URL       ?= postgres://nexus_app:$(DEV_PW)@127.0.0.1:5432/nexus_mini
export DATABASE_URL_ADMIN ?= postgres://nexus_admin:$(DEV_PW)@127.0.0.1:5432/nexus_mini
export DATABASE_URL_OWNER ?= postgres://nexus_owner:$(DEV_PW)@127.0.0.1:5432/nexus_mini

.PHONY: dev-db migrate api web check push

dev-db: ## локал Postgres дээр role + DB үүсгэнэ (нэг удаа)
	psql -h 127.0.0.1 -d $(PG_SUPER) \
	  -v owner_pw='$(DEV_PW)' -v app_pw='$(DEV_PW)' -v admin_pw='$(DEV_PW)' \
	  -f deploy/01-roles.sql

migrate:
	cd backend && go run ./cmd/migrate

api:
	cd backend && go run ./cmd/api

web:
	cd frontend && pnpm dev

# push бүрийн өмнө заавал (docs/01-lessons.md #4): linux build + vet + test
check:
	cd backend && GOOS=linux GOARCH=amd64 go build ./... && go vet ./... && go test ./...

push: check
	git push
