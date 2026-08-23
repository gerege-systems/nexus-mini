# nexus-mini — бүх командыг зөвхөн Makefile-аар ажиллуулна; бинарийг шууд дуудахгүй.
# Эхний ажиллуулалт: deploy/01-roles.sql (нэг удаа) → cp .env.example backend/nexus-mini.env
#                    → make migrate → make serve → make web
#
# ENV_FILE=/зам/nexus-mini.env  — өөр газрын env (сервер дээр secrets/); өгөхгүй бол
#                                 backend/nexus-mini.env-г уншина.

ENV_FLAG := $(if $(ENV_FILE),--env $(ENV_FILE),)

.PHONY: help build migrate serve web admin check check-db push manifest

help:
	@echo "make migrate   миграц + (env-д ADMIN_* байвал) анхны платформ админ"
	@echo "make serve     API сервер :8084"
	@echo "make web       portal dev :3020      make admin   админ панель dev :3021"
	@echo "make build     бинари (backend/bin/nexus-mini, атом солилт)"
	@echo "make manifest  registry манифест JSON (MOD=<short_id> нэг модуль)"
	@echo "make check     linux build + vet + test + SDK-ийн хил    make push   check → git push"
	@echo "make check-db  check + RLS/RBAC integration (NEXUS_TEST_DATABASE_URL{,_OWNER} заавал)"
	@echo "ENV_FILE=...   env файлын зам (default backend/nexus-mini.env)"

# Атом солилт: шинийг тусад нь build хийж mv-ээр (rename атом) дарна; өмнөхийг
# rollback-д үлдээнэ — ажиллаж буй процессын бинарийг хагас бичихгүй.
build:
	cd backend && go build -o bin/nexus-mini.new ./cmd/nexus-mini \
	  && { [ -f bin/nexus-mini ] && cp bin/nexus-mini bin/nexus-mini.prev || true; } \
	  && mv -f bin/nexus-mini.new bin/nexus-mini

migrate: build
	cd backend && ./bin/nexus-mini migrate $(ENV_FLAG)

serve: build
	cd backend && ./bin/nexus-mini serve $(ENV_FLAG)

# Registry манифест (кодоос). Нийтлэх: nexus-registry репогийн manifests/ руу.
manifest: build
	cd backend && ./bin/nexus-mini manifest $(MOD)

web:
	cd frontend && pnpm dev

admin:
	cd admin && pnpm dev

# push бүрийн өмнө заавал (docs/01-lessons.md #4): linux build + vet + test
# + SDK-ийн хил: модуль (apps/) internal/*-ээс юу ч импортлохгүй байх ёстой
check:
	cd backend && GOOS=linux GOARCH=amd64 go build ./... && go vet ./... && go test ./...
	@cd backend && bad=$$(go list -deps ./apps/... | grep 'backend/internal' || true); \
	 if [ -n "$$bad" ]; then echo "SDK хил зөрчигдөв — модуль internal/* импортолж байна:"; echo "$$bad"; exit 1; fi
	@if [ -z "$$NEXUS_TEST_DATABASE_URL" ]; then \
	 echo ""; \
	 echo "⚠  RLS + RBAC-ийн integration тест АЛГАСАГДЛАА (NEXUS_TEST_DATABASE_URL алга)."; \
	 echo "   go test 'ok' гэж бичсэн ч мөрийн түвшний тусгаарлалт болон эрхийн"; \
	 echo "   олголт — цөмийн хоёр гол инвариант — шалгагдаагүй байна."; \
	 echo "   Мөн: signup (хоёр DB role), түдгэлзүүлэлт, каталогийн апп."; \
	 echo "   Бүрэн шалгах:  make check-db NEXUS_TEST_DATABASE_URL=... _OWNER=... _AUTH=... _ADMIN=..."; \
	 echo ""; \
	 fi

# check-db — check-ийн бүрэн хувилбар: DB заавал, алгасалт = алдаа.
# CI болон прод deploy-ийн өмнө үүнийг ажиллуулна.
check-db: check
	@[ -n "$(NEXUS_TEST_DATABASE_URL)" ] && [ -n "$(NEXUS_TEST_DATABASE_URL_OWNER)" ] && \
	 [ -n "$(NEXUS_TEST_DATABASE_URL_AUTH)" ] && [ -n "$(NEXUS_TEST_DATABASE_URL_ADMIN)" ] || \
	 { echo "check-db: NEXUS_TEST_DATABASE_URL, _OWNER, _AUTH, _ADMIN бүгд шаардлагатай"; exit 1; }
	cd backend && NEXUS_TEST_REQUIRE_DB=1 go test -count=1 \
	  ./internal/core/db/... ./internal/core/rbac/... ./internal/core/handlers/... ./internal/core/appstore/...

push: check
	git push
