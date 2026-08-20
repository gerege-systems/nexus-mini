# nexus-mini
# Локал хөгжүүлэлт: `make dev-db` (нэг удаа) → `make migrate` → `make api`

# Тохиргоо backend/nexus-mini.env-д амьдарна — `make setup` үүсгэнэ.

.PHONY: setup migrate api web check push

setup: ## анхны тохируулга — интерактив CLI (role, DB, миграц, админ)
	cd backend && go run ./cmd/nexus-mini setup

migrate:
	cd backend && go run ./cmd/nexus-mini migrate

api:
	cd backend && go run ./cmd/nexus-mini serve

web:
	cd frontend && pnpm dev

# push бүрийн өмнө заавал (docs/01-lessons.md #4): linux build + vet + test
check:
	cd backend && GOOS=linux GOARCH=amd64 go build ./... && go vet ./... && go test ./...

push: check
	git push
