#!/usr/bin/env bash
# TLS SAN шалгагч — nginx-ийн үйлчилдэг нэр бүрийн хувьд тухайн нэрээр SNI
# тавьж холбогдоод, буцаж ирсэн сертификатад уг нэр ҮНЭХЭЭР байгаа эсэхийг
# шалгана.
#
# Яагаад: wildcard сертификат НЭГ л шат таардаг. `*.example.com` нь
# `admin.nexus.example.com` шиг хоёр шаттай нэрийг ХАМРАХГҮЙ. Ийм тохиолдолд
# nginx конфиг зөв, DNS зөв, service асаалттай, лог цэвэр — гэхдээ браузер
# TLS handshake дээрээ унана. Чимээгүй эвдрэл тул deploy бүрт шалгана.
#
# Хэрэглээ:
#   bash deploy/check-tls.sh                # nginx-ийн бүх 443 vhost
#   bash deploy/check-tls.sh a.example.com  # зөвхөн өгсөн нэрс
set -uo pipefail

WARN_DAYS=${WARN_DAYS:-21}
ADDR=${ADDR:-127.0.0.1:443}

nginx_hosts() {
  local dump
  dump=$(sudo -n nginx -T 2>/dev/null || nginx -T 2>/dev/null) || return 1
  # 443 сонсдог server блок бүрийн server_name-ууд (wildcard, `_` хасна).
  printf '%s\n' "$dump" | awk '
    /^[[:space:]]*server[[:space:]]*\{/ { depth=1; names=""; ssl=0; next }
    depth > 0 {
      if ($1 == "server_name") { for (i = 2; i <= NF; i++) { g = $i; sub(/;$/, "", g); names = names (names == "" ? "" : " ") g } }
      if ($1 == "listen" && $0 ~ /443/) ssl = 1
      n = gsub(/\{/, "{"); depth += n
      n = gsub(/\}/, "}"); depth -= n
      if (depth <= 0 && ssl && names != "") print names
    }
  ' | tr ' ' '\n' | grep -vE '^\*|^_$|^$' | sort -u
}

# $1 = хост, $2… = сертийн SAN нэрс. Wildcard нь яг нэг шат орлоно.
covers() {
  local host=$1 san; shift
  for san in "$@"; do
    [ "$san" = "$host" ] && return 0
    case $san in
      \*.*)
        local suffix=${san#\*.} head=${host%%.*}
        [ "$host" = "$head.$suffix" ] && [ "$head" != "$host" ] && return 0
        ;;
    esac
  done
  return 1
}

hosts=("$@")
if [ ${#hosts[@]} -eq 0 ]; then
  mapfile -t hosts < <(nginx_hosts)
  if [ ${#hosts[@]} -eq 0 ]; then
    echo "check-tls: nginx -T уншиж чадсангүй (sudo эрх?) — хостуудыг аргументаар өгнө үү" >&2
    exit 2
  fi
fi

fail=0
for host in "${hosts[@]}"; do
  cert=$(echo | openssl s_client -connect "$ADDR" -servername "$host" 2>/dev/null | openssl x509 2>/dev/null)
  if [ -z "$cert" ]; then
    printf 'FAIL %-44s TLS handshake бүтсэнгүй\n' "$host"
    fail=1
    continue
  fi
  mapfile -t san < <(printf '%s\n' "$cert" | openssl x509 -noout -ext subjectAltName 2>/dev/null |
    tr -d ' ' | tr ',' '\n' | sed -n 's/^DNS://p')
  if ! covers "$host" "${san[@]}"; then
    printf 'FAIL %-44s серт дээр SAN алга (%s)\n' "$host" "$(printf '%s,' "${san[@]}" | sed 's/,$//')"
    fail=1
    continue
  fi
  end=$(printf '%s\n' "$cert" | openssl x509 -noout -enddate | cut -d= -f2)
  left=$(( ( $(date -d "$end" +%s 2>/dev/null || echo 0) - $(date +%s) ) / 86400 ))
  if [ "$left" -lt "$WARN_DAYS" ]; then
    printf 'WARN %-44s %s хоногийн дараа дуусна\n' "$host" "$left"
  else
    printf 'ok   %-44s %s хоног\n' "$host" "$left"
  fi
done

[ "$fail" -eq 0 ] || echo "check-tls: дутуу SAN олдлоо — certbot --expand хийж nginx-ээ reload хийнэ үү" >&2
exit "$fail"
