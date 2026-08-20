#!/usr/bin/env bash
# Сервер дээр ажиллана: git pull → build → migrate → restart.
# Урсгал: локал засвар → GitHub → энэ скрипт (сервер дээр шууд засвар хийхгүй).
#
# Бүх ажил main() дотор: bash скриптийг мөр мөрөөр уншдаг тул git pull
# скриптийг өөрийг нь шинэчлэхэд дундаас нь хуучин/шинэ хольж уншихаас
# хамгаална — main-ийг дуудах мөр файлын төгсгөлд байгаа үед функц бүхэлдээ
# аль хэдийн уншигдсан байдаг.
set -euo pipefail

main() {
  cd /srv/nexus-mini
  git pull --ff-only

  export PATH=/usr/local/go/bin:$PATH

  echo "== backend build =="
  cd backend
  # Атом солилт: шинийг тусад нь build хийж, ажиллаж буй binary-г mv-ээр
  # (rename нь атом) дарна; өмнөхийг rollback-д үлдээнэ.
  go build -o bin/nexus-mini.new ./cmd/nexus-mini
  [ -f bin/nexus-mini ] && cp bin/nexus-mini bin/nexus-mini.prev
  mv -f bin/nexus-mini.new bin/nexus-mini

  echo "== migrate =="
  # Env-ийг export хийхгүй (#8: ADMIN_* гэх мэт нууц child process бүрт
  # задрах ёсгүй) — migrate --env флагаараа өөрөө уншина.
  ./bin/nexus-mini migrate --env /home/bay/secrets/nexus-mini.env

  # Next build-ийг амьд .next дээр биш тусдаа хавтаст хийж, дуусмагц
  # атомоор солино (mid-build 500/404-өөс сэргийлнэ).
  build_next() {
    pnpm install --frozen-lockfile
    rm -rf .next.new
    API_URL=http://127.0.0.1:8084 NEXT_DIST_DIR=.next.new pnpm build
    rm -rf .next.prev
    [ -d .next ] && mv .next .next.prev
    mv .next.new .next
  }

  echo "== frontend build =="
  cd ../frontend
  build_next

  echo "== admin build =="
  cd ../admin
  build_next

  echo "== restart =="
  sudo systemctl restart nexus-mini-api nexus-mini-web nexus-mini-adminweb
  sleep 2
  systemctl is-active nexus-mini-api nexus-mini-web nexus-mini-adminweb
  curl -sf http://127.0.0.1:8084/health > /dev/null && echo "api: ok"
  curl -sf -o /dev/null http://127.0.0.1:3020/ && echo "web: ok"
  curl -sf -o /dev/null http://127.0.0.1:3021/login && echo "admin: ok"
  exit 0
}

main "$@"
