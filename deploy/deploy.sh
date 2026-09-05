#!/usr/bin/env bash
# Сервер дээр ажиллана: git pull → build → migrate → restart.
# Урсгал: локал засвар → GitHub → энэ скрипт (сервер дээр шууд засвар хийхгүй).
#
# Бүх ажил main() дотор: bash скриптийг мөр мөрөөр уншдаг тул git pull
# скриптийг өөрийг нь шинэчлэхэд дундаас нь хуучин/шинэ хольж уншихаас
# хамгаална — main-ийг дуудах мөр файлын төгсгөлд байгаа үед функц бүхэлдээ
# аль хэдийн уншигдсан байдаг. Pull-аар deploy.sh өөрөө өөрчлөгдвөл шинэ
# хувилбараар exec-ээр дахин эхэлнэ (NEXUS_DEPLOY_REEXEC) — засвар тэр даруй хэрэгжинэ.
set -euo pipefail

main() {
  cd /srv/nexus-mini
  local root=$PWD
  # Next build (NEXT_DIST_DIR=.next.new) нь эдгээр ҮҮСГЭСЭН файлуудыг .next.new
  # рүү заалгаж бичдэг — track хийгддэг тул upstream тэдгээрт хүрвэл pull унана.
  # Хүний засвар биш учир pull-ийн өмнө буцаана (бусад файлд хүрэхгүй).
  git checkout -- frontend/next-env.d.ts frontend/tsconfig.json \
                  admin/next-env.d.ts admin/tsconfig.json 2>/dev/null || true
  # Дахин ачаалагдсан (доор) бол pull аль хэдийн хийгдсэн.
  if [ -z "${NEXUS_DEPLOY_REEXEC:-}" ]; then
    local before; before=$(git rev-parse HEAD)
    git pull --ff-only
    # main() бүхэлдээ уншигдсан тул энэ ажиллалт ХУУЧИН deploy.sh — скрипт
    # өөрөө өөрчлөгдсөн бол шинэ хувилбараар нэг удаа дахин эхэлнэ.
    if ! git diff --quiet "$before" HEAD -- deploy/deploy.sh; then
      echo "== deploy.sh өөрчлөгдсөн → шинэ хувилбараар дахин эхэлнэ =="
      NEXUS_DEPLOY_REEXEC=1 exec bash "$root/deploy/deploy.sh" "$@"
    fi
  fi

  export PATH=/usr/local/go/bin:$PATH

  local envf="${NEXUS_ENV_FILE:-/etc/nexus-mini/nexus-mini.env}"

  echo "== DB нөөц (миграцын өмнө) =="
  # pg_dump -Fc; хамгийн сүүлийн 14-ийг үлдээнэ. DB нэрийг env-ийн
  # DATABASE_URL_OWNER-оос авна (default nexus_mini). Сэргээх: pg_restore -d <db> <файл>.
  local dbname bdir
  dbname=$(grep -E '^DATABASE_URL_OWNER=' "$envf" | sed -E 's#.*/([^/?"[:space:]]+).*#\1#' || true)
  dbname=${dbname:-nexus_mini}
  bdir="${NEXUS_BACKUP_DIR:-/var/backups/nexus-mini}"
  # sudo -n: password асуувал зогсох биш шууд унана (non-interactive deploy).
  local dump="$bdir/$dbname-$(date +%Y%m%d-%H%M%S).dump"
  sudo -n install -d -m 700 "$bdir"
  sudo -n -u postgres pg_dump -Fc "$dbname" | sudo -n tee "$dump" > /dev/null
  sudo -n chmod 600 "$dump"   # нууц үгийн hash, session агуулна — зөвхөн root
  # Хавтас 700 root тул жагсаалт/эргэлтийг ч root-оор.
  sudo -n sh -c "ls -1t '$bdir'/*.dump | tail -n +15 | xargs -r rm --"
  echo "нөөц: $dump"

  echo "== backend build + migrate (Makefile) =="
  # Атом солилт Makefile-ийн build-д; env-ийг export хийхгүй (#8: ADMIN_*
  # гэх мэт нууц child process бүрт задрах ёсгүй) — ENV_FILE флагаар уншина.
  make migrate ENV_FILE="$envf"
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
