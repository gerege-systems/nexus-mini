package nexus

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDecodeLimitsAndUnknownFields(t *testing.T) {
	type in struct {
		Name string `json:"name"`
	}
	cases := []struct {
		name, body string
		wantOK     bool
	}{
		{"зөв", `{"name":"а"}`, true},
		{"үл мэдэх талбар", `{"name":"а","hack":1}`, false},
		{"эвдэрсэн JSON", `{`, false},
		{"хэт том бие", `{"name":"` + strings.Repeat("a", 2<<20) + `"}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var v in
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(c.body))
			if got := Decode(w, r, &v); got != c.wantOK {
				t.Fatalf("Decode = %v, хүлээсэн %v (код %d)", got, c.wantOK, w.Code)
			}
			if !c.wantOK && w.Code < 400 {
				t.Fatalf("алдаатай бие дээр код = %d", w.Code)
			}
		})
	}
}

func TestUUIDHelpers(t *testing.T) {
	good := "3f8d0c62-3a5e-4a1b-9a2f-1b2c3d4e5f60"
	for _, s := range []string{good, strings.ToUpper(good)} {
		if !IsUUID(s) {
			t.Fatalf("IsUUID(%q) = false", s)
		}
	}
	for _, s := range []string{"", "not-a-uuid", good + "x", "3f8d0c62-3a5e-4a1b-9a2f-1b2c3d4e5f6", "'; DROP TABLE users--"} {
		if IsUUID(s) {
			t.Fatalf("IsUUID(%q) = true", s)
		}
	}
	r := chi.NewRouter()
	var seen string
	var status int
	r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
		v, ok := UUIDParam(w, req, "id")
		seen, status = v, 0
		if !ok {
			status = 1
		}
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/"+good, nil))
	if seen != good || status != 0 || w.Code != 200 {
		t.Fatalf("зөв uuid: %q ok=%d код=%d", seen, status, w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/буруу", nil))
	if status != 1 || w.Code != http.StatusBadRequest {
		t.Fatalf("буруу uuid: код = %d (400 байх ёстой)", w.Code)
	}
}

func TestDBErrorMapping(t *testing.T) {
	unique := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint \"users_email_key\""}
	fk := &pgconn.PgError{Code: "23503"}
	other := &pgconn.PgError{Code: "42601", Message: "syntax error at or near SELECT"}
	if !IsUniqueViolation(unique) || IsUniqueViolation(fk) || IsUniqueViolation(errors.New("x")) {
		t.Fatal("IsUniqueViolation буруу")
	}
	if !IsFKViolation(fk) || IsFKViolation(unique) {
		t.Fatal("IsFKViolation буруу")
	}
	w := httptest.NewRecorder()
	DBError(w, unique, "давхардлаа")
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "давхардлаа") {
		t.Fatalf("unique → %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	DBError(w, other, "давхардлаа")
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "syntax error") {
		t.Fatalf("DB-ийн текст клиентэд алдагдав: %d %s", w.Code, w.Body.String())
	}
}

func TestJSONAndError(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusCreated, map[string]string{"a": "б"})
	if w.Code != 201 || w.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		!strings.Contains(w.Body.String(), `"б"`) {
		t.Fatalf("JSON: %d %s %s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
	w = httptest.NewRecorder()
	Error(w, http.StatusForbidden, "болохгүй")
	if w.Code != 403 || !strings.Contains(w.Body.String(), `"error":"болохгүй"`) {
		t.Fatalf("Error: %d %s", w.Code, w.Body.String())
	}
}
