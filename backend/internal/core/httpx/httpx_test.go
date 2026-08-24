package httpx

// httpx нь pkg/nexus-ийн нимгэн бүрхүүл — дамжуулалт зөв эсэхийг барина.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestWrappers(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusAccepted, map[string]int{"a": 1})
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"a":1`) {
		t.Fatalf("JSON = %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	Error(w, http.StatusTeapot, "алдаа")
	if w.Code != http.StatusTeapot || !strings.Contains(w.Body.String(), "алдаа") {
		t.Fatalf("Error = %d %s", w.Code, w.Body.String())
	}
	var v struct {
		A int `json:"a"`
	}
	w = httptest.NewRecorder()
	if !Decode(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":2}`)), &v) || v.A != 2 {
		t.Fatalf("Decode = %+v", v)
	}
	w = httptest.NewRecorder()
	if Decode(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`)), &v) {
		t.Fatal("эвдэрсэн JSON хүлээн авагдав")
	}
	w = httptest.NewRecorder()
	DBError(w, &pgconn.PgError{Code: "23505"}, "давхардлаа")
	if w.Code != http.StatusConflict {
		t.Fatalf("DBError = %d", w.Code)
	}
}
