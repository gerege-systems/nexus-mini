#!/usr/bin/env bash
# Сервер дээр ажиллана: git pull → build → migrate → restart.
# Урсгал: локал засвар → GitHub → энэ скрипт (сервер дээр шууд засвар хийхгүй).
set -euo pipefail

cd /srv/nexus-mini
git pull --ff-only

export PATH=/usr/local/go/bin:$PATH
set -a
source /home/bay/secrets/nexus-mini.env
set +a

echo "== backend build =="
cd backend
go build -o bin/api ./cmd/api
go build -o bin/migrate ./cmd/migrate

echo "== migrate =="
./bin/migrate

echo "== frontend build =="
cd ../frontend
pnpm install --frozen-lockfile
API_URL=http://127.0.0.1:8084 pnpm build

echo "== restart =="
sudo systemctl restart nexus-mini-api nexus-mini-web
sleep 2
systemctl is-active nexus-mini-api nexus-mini-web
curl -sf http://127.0.0.1:8084/health > /dev/null && echo "api: ok"
curl -sf -o /dev/null http://127.0.0.1:3020/ && echo "web: ok"
