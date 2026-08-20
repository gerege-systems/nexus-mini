# nexus-mini
# Локал хөгжүүлэлт: deploy/01-roles.sql (нэг удаа) → .env бөглөх → make migrate → make api

# Тохиргоо backend/nexus-mini.env-д амьдарна — .env.example-г хуулж бөглөнө.

.PHONY: migrate api web check push

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
