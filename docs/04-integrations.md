# Гадны системтэй холбох — OIDC provider, SSO, federation

nexus-mini нь **OpenID Connect provider** (бусад систем энэ платформын
бүртгэлээр нэвтэрнэ) ба **relying party** (энэ платформ Google/өөр OIDC
issuer-ээр нэвтэрнэ) хоёулаа. Хоёр nexus-mini хоорондоо = federation.

## 1. Таны систем nexus-mini-ээр нэвтрэх (provider)

1. Portal → **SSO клиентүүд** (`core.sso.manage`) → «Клиент нэмэх»: нэр,
   redirect URI (https, эсвэл localhost), scope. Confidential клиент бол
   `client_secret` **нэг л удаа** харагдана; SPA/mobile бол «Public» (PKCE).
2. Discovery: `<PORTAL_URL>/api/oauth2/.well-known/openid-configuration`
   (issuer = `<PORTAL_URL>/api/oauth2`). Дурын OIDC номын сан (openid-client,
   oidc-client-ts, Spring Security, Keycloak broker, NextAuth…) үүгээр бүх
   endpoint-ийг олно.
3. Урсгал: `authorization_code` + **PKCE S256 заавал** (public/confidential
   аль алинд). Хэрэглэгч portal-д нэвтэрч (эсвэл `next` параметртэй login
   руу), клиентийн **байгууллагын гишүүн** бол consent хуудас → code →
   `/token`. Зөвшөөрөл санагдана (`prompt=consent` дахин асууна).
4. Токен: access token opaque (`/introspect`-ээр шалгана, `/revoke`-оор
   хүчингүй), `id_token` RS256 (`/jwks`), `offline_access` scope →
   refresh token (rotation; хуучин refresh дахин ирвэл гэр бүлээр хүчингүй).
5. Claims: `sub` (user id), `profile`→`name`, `email`, `tenant`→`tenant`
   (slug) + `tenant_id`, `roles`→role кодууд (тухайн байгууллагад).
   `/userinfo` ч ижил.
6. Гарах: `end_session?id_token_hint=…&post_logout_redirect_uri=…` (бүртгэлтэй
   URI) — portal session дуусна.
7. Сервер-сервер: `client_credentials` (confidential) → `tenant` scope-той
   access token; `sub` байхгүй, `tenant_id` бий.

Жишээ (curl, confidential клиент):

```bash
# 1. browser: <issuer>/authorize?response_type=code&client_id=…&redirect_uri=…&scope=openid%20profile%20email&state=…&nonce=…&code_challenge=…&code_challenge_method=S256
# 2. callback-д ирсэн code-оор:
curl -u "$CLIENT_ID:$CLIENT_SECRET" -X POST <issuer>/token \
  -d grant_type=authorization_code -d code=… -d redirect_uri=… -d code_verifier=…
# 3. access token шалгах:
curl -u "$CLIENT_ID:$CLIENT_SECRET" -X POST <issuer>/introspect -d token=…
```

Аюулгүй байдал: redirect_uri яг таарна (open redirect байхгүй), code 5 мин +
нэг удаа, PKCE заавал, `/token`-д rate limit, CORS нээлттэй (cookie-гүй
endpoint-ууд), consent cookie-тэй тул CSRF хамгаалалттай. Токен/код/түлхүүр
хүснэгтүүд зөвхөн `nexus_auth` DB role-д — модулийн SQL хүрэхгүй.

## 2. nexus-mini Google / өөр OIDC-ээр нэвтрэх (relying party)

Env (portal + API нэг дор):

```
GOOGLE_CLIENT_ID=…            # console.cloud.google.com → OAuth client (Web)
GOOGLE_CLIENT_SECRET=…        # redirect URI: <PORTAL_URL>/api/auth/sso/google/callback
SSO_ISSUER=https://idp.example.com     # дурын OIDC (discovery-тэй)
SSO_CLIENT_ID=… SSO_CLIENT_SECRET=… SSO_NAME="Компанийн SSO"
SSO_AUTO_SIGNUP=false         # true: танигдаагүй имэйлд данс үүсгэнэ (JIT)
```

Тохируулсан provider бүрт login хуудсанд товч гарна. Урсгал PKCE + state +
nonce (HMAC-тэй 10 мин cookie), id_token-ийг issuer-ийн JWKS-ээр (RS256)
шалгана, `iss/aud/exp/nonce` тулгана. Хэрэглэгчийг таних (`sso_identities`):
1. `(iss, sub)` холбоос байвал → тэр данс (email_verified-ээс хамаарахгүй).
2. Холбоосгүй эхний нэвтрэлт: имэйлээр бүртгэлтэй данс руу **зөвхөн
   `email_verified=true`** үед холбоно. Баталгаажаагүй имэйл (nexus-mini
   өөрөө provider болохдоо үргэлж `false` өгдөг) + байгаа данс → татгалзана,
   хэрэглэгч нууц үгээрээ нэвтэрнэ. Өөр issuer дээр хохирогчийн имэйлээр
   бүртгүүлж дансыг нь авах боломжгүй.
3. Бүртгэлгүй имэйл → `SSO_AUTO_SIGNUP` байвал нууц үггүй данс (зөвхөн
   SSO-оор нэвтэрнэ), холбоос бичигдэнэ; үгүй бол татгалзана.

## 3. Federation — хоёр nexus-mini

Bold-ын instance Таны instance-ээр нэвтрэх: Танай portal-д Bold-ын instance-ийг
SSO клиент болгон бүртгэнэ (redirect `https://nexus.bold.mn/api/auth/sso/sso/callback`),
Bold `SSO_ISSUER=https://nexus.tanai.mn/api/oauth2` + client_id/secret тавина.
Bold-ын хэрэглэгч «Tanai SSO» товч → танай consent → буцаад Bold дээр
нэвтэрнэ (имэйл таарсан данс эсвэл JIT). Урвуу чиглэл нь яг ижил.

## 4. Roadmap (идэвхгүй)
MFA (TOTP) — одоогийн session/lockout дээр нэмэгдэнэ; eID/ДАН — identity rail
модуль хэлбэрээр; dynamic client registration — хэрэгцээ гарвал.
