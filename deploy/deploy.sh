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
  local root=$PWD
  # Next build (NEXT_DIST_DIR=.next.new) нь эдгээр ҮҮСГЭСЭН файлуудыг .next.new
  # рүү заалгаж бичдэг — track хийгддэг тул upstream тэдгээрт хүрвэл pull унана.
  # Хүний засвар биш учир pull-ийн өмнө буцаана (бусад файлд хүрэхгүй).
  git checkout -- frontend/next-env.d.ts frontend/tsconfig.json \
                  admin/next-env.d.ts admin/tsconfig.json 2>/dev/null || true
  git pull --ff-only

  export PATH=/usr/local/go/bin:$PATH

  echo "== backend build + migrate (Makefile) =="
  # Атом солилт Makefile-ийн build-д; env-ийг export хийхгүй (#8: ADMIN_*
  # гэх мэт нууц child process бүрт задрах ёсгүй) — ENV_FILE флагаар уншина.
  make migrate ENV_FILE="${NEXUS_ENV_FILE:-/etc/nexus-mini/nexus-mini.env}"
  cd backend

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

  # nginx-ийн нэр бүр өөрийгөө хамарсан сертификаттай эсэх. Wildcard нь нэг л
  # шат таардаг тул хоёр шаттай дэд домэйн чимээгүй унасан байж болно.
  echo "== TLS =="
  bash "$root/deploy/check-tls.sh" || true
  exit 0
}

main "$@"
