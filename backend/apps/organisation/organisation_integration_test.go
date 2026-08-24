package organisation_test

// Байгууллагын бүтэц: хэлтсийн мод, мөчлөгийн хамгаалалт, өөр tenant-ийн
// холбоос (same-tenant trigger), ажилтны байршил, эрхийн хил.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/apps/organisation"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	coredb "github.com/gerege-systems/nexus-mini/backend/internal/core/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fx struct {
	router    http.Handler
	owner     *pgxpool.Pool
	tid, tid2 string
	manager   string
	viewer    string
	member2   string // өөр tenant-ийн гишүүнчлэл
}

func setup(t *testing.T) *fx {
	t.Helper()
	appURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if appURL == "" || ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL / _OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	app, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	f := &fx{owner: owner}
	clean := func() {
		_, _ = owner.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'otest%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM users WHERE email LIKE 'otest%'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		app.Close()
		owner.Close()
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('otest','О') RETURNING id`).Scan(&f.tid))
	must(owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('otest2','О2') RETURNING id`).Scan(&f.tid2))
	must(owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('otest-mgr@x.mn','x','Менежер') RETURNING id`).Scan(&f.manager))
	must(owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('otest-view@x.mn','x','Харагч') RETURNING id`).Scan(&f.viewer))
	var other string
	must(owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('otest-other@x.mn','x','Гадны') RETURNING id`).Scan(&other))
	for _, u := range []string{f.manager, f.viewer} {
		_, err := owner.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2)`, f.tid, u)
		must(err)
	}
	must(owner.QueryRow(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2) RETURNING id`, f.tid2, other).Scan(&f.member2))

	m := organisation.New()
	for _, p := range m.Permissions() {
		_, err := owner.Exec(ctx, `
			INSERT INTO permissions (code, module_id, name, own_scope) VALUES ($1,$2,$3,$4)
			ON CONFLICT (code) DO NOTHING`, p.Code, m.ID(), p.Name, p.OwnScope)
		must(err)
	}
	grant := func(code, user string, perms map[string]string) {
		var roleID, memberID string
		must(owner.QueryRow(ctx, `INSERT INTO roles (tenant_id, code, name) VALUES ($1,$2,$2) RETURNING id`, f.tid, code).Scan(&roleID))
		must(owner.QueryRow(ctx, `SELECT id FROM memberships WHERE tenant_id=$1::uuid AND user_id=$2::uuid`, f.tid, user).Scan(&memberID))
		_, err := owner.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1,$2)`, memberID, roleID)
		must(err)
		for c, s := range perms {
			_, err := owner.Exec(ctx, `INSERT INTO role_permissions (role_id, permission_code, scope) VALUES ($1,$2,$3)`, roleID, c, s)
			must(err)
		}
	}
	grant("o_mgr", f.manager, map[string]string{"organisation.read": "all", "organisation.manage": "all"})
	grant("o_view", f.viewer, map[string]string{"organisation.read": "all"})

	tdb := coredb.NewTenantDB(app)
	deps := nexus.Deps{DB: tdb, Perms: rbac.NewStore(tdb), Audit: audit.NewRecorder(tdb)}
	r := chi.NewRouter()
	r.Route("/api/apps/organisation", func(sub chi.Router) {
		sub.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				uid := req.Header.Get("X-Test-User")
				ctx := identity.With(req.Context(), f.tid, uid)
				ctx = nexus.WithIdentity(ctx, f.tid, uid)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		m.RegisterRoutes(sub, deps)
	})
	f.router = r
	return f
}

func (f *fx) req(t *testing.T, user, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, target, &buf)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Test-User", user)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	return w
}

func (f *fx) create(t *testing.T, body map[string]any) string {
	t.Helper()
	w := f.req(t, f.manager, http.MethodPost, "/api/apps/organisation/departments", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("хэлтэс үүсгэх = %d: %s", w.Code, w.Body.String())
	}
	var out map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out["id"]
}

func TestDepartmentsTreeAndGuards(t *testing.T) {
	f := setup(t)
	hq := f.create(t, map[string]any{"code": "hq", "name": "Төв"})
	it := f.create(t, map[string]any{"code": "it", "name": "IT", "parent_id": hq})

	// Мөчлөг: hq-ийн эцгийг it болгох.
	if w := f.req(t, f.manager, http.MethodPut, "/api/apps/organisation/departments/"+hq,
		map[string]any{"code": "hq", "name": "Төв", "parent_id": it}); w.Code != http.StatusBadRequest {
		t.Fatalf("мөчлөг = %d (400 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// Өөрийгөө эцэг болгох.
	if w := f.req(t, f.manager, http.MethodPut, "/api/apps/organisation/departments/"+it,
		map[string]any{"code": "it", "name": "IT", "parent_id": it}); w.Code != http.StatusBadRequest {
		t.Fatalf("өөрийгөө эцэг = %d (400 хүлээсэн)", w.Code)
	}
	// Давхардсан код.
	if w := f.req(t, f.manager, http.MethodPost, "/api/apps/organisation/departments",
		map[string]any{"code": "it", "name": "IT2"}); w.Code != http.StatusConflict {
		t.Fatalf("давхардсан код = %d (409 хүлээсэн)", w.Code)
	}
	// Байхгүй эцэг.
	if w := f.req(t, f.manager, http.MethodPost, "/api/apps/organisation/departments",
		map[string]any{"code": "x", "name": "X", "parent_id": "00000000-0000-0000-0000-000000000001"}); w.Code != http.StatusBadRequest {
		t.Fatalf("байхгүй эцэг = %d (400 хүлээсэн)", w.Code)
	}
	// ӨӨР TENANT-ийн гишүүнчлэлийг менежер болгох — DB триггер барина.
	if w := f.req(t, f.manager, http.MethodPut, "/api/apps/organisation/departments/"+it,
		map[string]any{"code": "it", "name": "IT", "manager_membership_id": f.member2}); w.Code < 400 {
		t.Fatalf("өөр tenant-ийн менежер = %d (алдаа хүлээсэн): %s", w.Code, w.Body.String())
	}
	// Уншигч засаж чадахгүй, харж чадна.
	if w := f.req(t, f.viewer, http.MethodPost, "/api/apps/organisation/departments",
		map[string]any{"code": "y", "name": "Y"}); w.Code != http.StatusForbidden {
		t.Fatalf("уншигч үүсгэв = %d (403 хүлээсэн)", w.Code)
	}
	w := f.req(t, f.viewer, http.MethodGet, "/api/apps/organisation/departments", nil)
	var list struct {
		Departments []map[string]any `json:"departments"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list.Departments) != 2 {
		t.Fatalf("хэлтсийн тоо = %d", len(list.Departments))
	}
	// Устгахад харьяа нэгж дээд түвшинд гарна.
	if w := f.req(t, f.manager, http.MethodDelete, "/api/apps/organisation/departments/"+hq, nil); w.Code != 200 {
		t.Fatalf("устгах = %d", w.Code)
	}
	w = f.req(t, f.manager, http.MethodGet, "/api/apps/organisation/departments", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list.Departments) != 1 || list.Departments[0]["parent_id"] != nil {
		t.Fatalf("устгасны дараа = %v", list.Departments)
	}
	// Байхгүй хэлтэс, буруу uuid.
	if w := f.req(t, f.manager, http.MethodDelete, "/api/apps/organisation/departments/00000000-0000-0000-0000-000000000009", nil); w.Code != http.StatusNotFound {
		t.Fatalf("байхгүй хэлтэс = %d", w.Code)
	}
	if w := f.req(t, f.manager, http.MethodDelete, "/api/apps/organisation/departments/тийм-биш", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу uuid = %d", w.Code)
	}
}

func TestPeoplePositions(t *testing.T) {
	f := setup(t)
	dep := f.create(t, map[string]any{"code": "sales", "name": "Борлуулалт"})
	w := f.req(t, f.manager, http.MethodGet, "/api/apps/organisation/people", nil)
	var people struct {
		People []map[string]any `json:"people"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &people)
	if len(people.People) != 2 {
		t.Fatalf("ажилтны тоо = %d", len(people.People))
	}
	// Имэйл ГАРАХГҮЙ (organisation.read нь бүх гишүүнд байдаг).
	if _, ok := people.People[0]["email"]; ok {
		t.Fatal("people-д имэйл гарч байна")
	}
	mid := people.People[0]["membership_id"].(string)
	if w := f.req(t, f.manager, http.MethodPut, "/api/apps/organisation/people/"+mid,
		map[string]any{"department_id": dep, "job_title": "Менежер"}); w.Code != 200 {
		t.Fatalf("байршил = %d: %s", w.Code, w.Body.String())
	}
	w = f.req(t, f.manager, http.MethodGet, "/api/apps/organisation/people", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &people)
	var found bool
	for _, p := range people.People {
		if p["membership_id"] == mid && p["department_name"] == "Борлуулалт" && p["job_title"] == "Менежер" {
			found = true
		}
	}
	if !found {
		t.Fatalf("байршил хадгалагдсангүй: %v", people.People)
	}
	// Өөр tenant-ийн гишүүнчлэлд байршил өгөх — гишүүнчлэл олдохгүй.
	if w := f.req(t, f.manager, http.MethodPut, "/api/apps/organisation/people/"+f.member2,
		map[string]any{"job_title": "Hack"}); w.Code != http.StatusNotFound {
		t.Fatalf("өөр tenant-ийн ажилтан = %d (404 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// Байхгүй хэлтэс.
	if w := f.req(t, f.manager, http.MethodPut, "/api/apps/organisation/people/"+mid,
		map[string]any{"department_id": "00000000-0000-0000-0000-000000000009"}); w.Code != http.StatusBadRequest {
		t.Fatalf("байхгүй хэлтэс = %d (400 хүлээсэн)", w.Code)
	}
	// Хэт урт албан тушаал.
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	if w := f.req(t, f.manager, http.MethodPut, "/api/apps/organisation/people/"+mid,
		map[string]any{"job_title": string(long)}); w.Code != http.StatusBadRequest {
		t.Fatalf("урт албан тушаал = %d (400 хүлээсэн)", w.Code)
	}
}
