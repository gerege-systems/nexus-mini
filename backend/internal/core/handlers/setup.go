package handlers

// Setup — анхны тохиргооны шидтэн (open-gerege-nexus-ийн internal/operator/setup-
// ийн загвар). env ADMIN_* + `make migrate` замын ХАЖУУД оршино: тэр нь
// терминал + env файлтай хүнд, энэ нь хөтчийн өмнө зогсож буй хүнд.
//
// Нээлттэй /setup-ыг public хост дээр эхлээд хүрсэн хүн эзэмшдэг тул
// токеноор зэвсэглэнэ: 256 бит, платформ админгүй үед л API асахад санах ойд
// үүсгэж, операторын уншиж буй ЛОГТ нэг удаа бичнэ, хаана ч хадгалахгүй, админ
// үүсмэгц устгана. Restart → шинэ токен. Буруу токенд 401 биш 404 — таах
// токен байгааг ч мэдэгдэхгүй. DB тал (auth_setup_admin) хоёр дахь админыг
// өөрөө татгалздаг тул handler-ийн алдаа ч хоёр дахь хаалга болохгүй.

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
)

type Setup struct {
	Auth      *Auth
	PortalURL string

	mu    sync.Mutex
	token string // энэ service-ийн бүх төлөв; мөр биш — процессоос хэтэрч амьдрахгүй
}

func NewSetup(a *Auth, portalURL string) *Setup {
	return &Setup{Auth: a, PortalURL: strings.TrimRight(portalURL, "/")}
}

// Arm — платформ админ байхгүй бол токен үүсгэж логт бичнэ. Байвал юу ч
// хийхгүй тул асах бүрд дуудахад аюулгүй.
func (s *Setup) Arm(ctx context.Context) {
	var exists bool
	if err := s.Auth.Pool.QueryRow(ctx, `SELECT auth_platform_admin_exists()`).Scan(&exists); err != nil {
		log.Printf("setup: платформ админ байгаа эсэхийг шалгаж чадсангүй: %v", err)
		return
	}
	if exists {
		return
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("setup: токен үүсгэж чадсангүй (%v) — env ADMIN_* + make migrate ашигла", err)
		return
	}
	tok := hex.EncodeToString(buf)
	s.mu.Lock()
	s.token = tok
	s.mu.Unlock()
	log.Printf("АНХААР: платформын админ байхгүй — хэн ч нэвтэрч чадахгүй. Шидтэн: %s/setup?token=%s"+
		"  (эсвэл nexus-mini.env-д ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD бичээд make migrate)", s.PortalURL, tok)
}

// Token — идэвхтэй токен (тест, оношилгоо). Зэвсэглээгүй бол хоосон.
func (s *Setup) Token() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

func (s *Setup) disarm() {
	s.mu.Lock()
	s.token = ""
	s.mu.Unlock()
}

func (s *Setup) authorised(r *http.Request) bool {
	want := s.Token()
	if want == "" {
		return false
	}
	got := strings.TrimSpace(r.Header.Get("X-Setup-Token"))
	if got == "" {
		got = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// GET /api/setup/status — хөтөч юу ч зурахаасаа өмнө асууна. Токенгүй цорын
// ганц зам: "админ байхгүй" гэдэг нэг бит нь нэвтрэх гэж оролдоход угаас
// харагддаг.
func (s *Setup) Status(w http.ResponseWriter, r *http.Request) {
	var exists bool
	if err := s.Auth.Pool.QueryRow(r.Context(), `SELECT auth_platform_admin_exists()`).Scan(&exists); err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, "өгөгдлийн санд хүрсэнгүй")
		return
	}
	if exists {
		s.disarm()
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"required": !exists,
		// Шидтэнийг бодитоор ашиглаж болох эсэх (шаардлагатай эсэхээс тусдаа):
		// restart-аар токен алдагдсан бол дэлгэц "логоо хар" гэнэ.
		"armed": !exists && s.Token() != "",
	})
}

// POST /api/setup/complete {admin:{email,name,password}, organisation?:{name,slug}}
// Токентой. Админ (DB талд хамгаалагдсан) → заавал биш анхны байгууллага
// (signup-тай ижил createTenantTx). Байгууллага бүтэхгүй бол админ аль хэдийн
// үүссэн тул 201 + tenant_error — хэрэглэгч нэвтэрч /org/new-ээс үүсгэнэ.
func (s *Setup) Complete(w http.ResponseWriter, r *http.Request) {
	if !s.authorised(r) {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	var in struct {
		Admin struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
		} `json:"admin"`
		Organisation struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"organisation"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Admin.Email))
	name := strings.TrimSpace(in.Admin.Name)
	orgName := strings.TrimSpace(in.Organisation.Name)
	orgSlug := strings.ToLower(strings.TrimSpace(in.Organisation.Slug))
	if !emailRe.MatchString(email) || len(email) > 255 || name == "" || len(name) > 120 {
		httpx.Error(w, http.StatusBadRequest, "админы имэйл (≤255), нэр (≤120) шаардлагатай")
		return
	}
	if err := password.Validate(in.Admin.Password); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	withOrg := orgName != "" || orgSlug != ""
	if withOrg && (orgName == "" || len(orgName) > 160 || !slugRe.MatchString(orgSlug)) {
		httpx.Error(w, http.StatusBadRequest, "байгууллагын нэр (≤160) ба "+slugHint)
		return
	}
	hash, err := password.Hash(in.Admin.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "hash failed")
		return
	}

	var uid *string
	err = s.Auth.Pool.QueryRow(r.Context(),
		`SELECT auth_setup_admin($1::varchar(255), $2::varchar(255), $3::varchar(120))`,
		email, hash, name).Scan(&uid)
	switch {
	case err == nil && uid == nil:
		// Хаалга аль хэдийн хаагдсан — хариу бичихээс өмнө токеноо ч хаяна.
		s.disarm()
		httpx.Error(w, http.StatusConflict, "платформын админ аль хэдийн бий")
		return
	case nexus.IsUniqueViolation(err):
		httpx.Error(w, http.StatusConflict, "энэ имэйлтэй данс бүртгэлтэй байна")
		return
	case err != nil:
		log.Printf("setup admin: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "тохиргоо амжилтгүй боллоо")
		return
	}
	// Эхний хүн орж ирмэгц хаалга хаагдана — хариунаас өмнө.
	s.disarm()
	log.Printf("setup: платформын админ %s шидтэнээр үүсэв", email)

	out := map[string]any{"user_id": *uid}
	if withOrg {
		var tenantID string
		err := s.Auth.DB.Tx(identity.With(r.Context(), "", ""), func(tx pgx.Tx) error {
			var err error
			tenantID, err = createTenantTx(r.Context(), tx, *uid, orgName, orgSlug)
			return err
		})
		switch {
		case err == nil:
			s.Auth.Audit.RecordAs(r.Context(), tenantID, *uid, "tenant.create", orgSlug, map[string]any{"name": orgName, "setup": true})
			out["tenant_id"] = tenantID
		case nexus.IsUniqueViolation(err):
			out["tenant_error"] = "slug давхардаж байна"
		default:
			log.Printf("setup tenant: %v", err)
			out["tenant_error"] = "байгууллага үүсгэж чадсангүй"
		}
	}
	httpx.JSON(w, http.StatusCreated, out)
}
