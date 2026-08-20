package appstore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeDB — Gate-ийн "суусан юу" query-д хариулдаг хамгийн жижиг nexus.DB.
type fakeDB struct{ enabled bool }

type fakeRow struct{ vals []any }

func (r fakeRow) Scan(dest ...any) error {
	for i := range dest {
		if b, ok := dest[i].(*bool); ok {
			*b = r.vals[i].(bool)
		}
	}
	return nil
}

func (f fakeDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (f fakeDB) QueryRow(context.Context, string, ...any) pgx.Row        { return fakeRow{[]any{f.enabled}} }
func (f fakeDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (f fakeDB) Tx(context.Context, func(tx pgx.Tx) error) error { return nil }

func gateReq(t *testing.T, g *Gate) int {
	t.Helper()
	h := g.Middleware("t.app")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(nexus.WithIdentity(req.Context(), "t1", "u1"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestGate(t *testing.T) {
	// Суулгаагүй → 403.
	if code := gateReq(t, NewGate(fakeDB{enabled: false})); code != http.StatusForbidden {
		t.Fatalf("суулгаагүй апп нэвтэрлээ: %d", code)
	}
	// Суусан → 200.
	g := NewGate(fakeDB{enabled: true})
	if code := gateReq(t, g); code != http.StatusOK {
		t.Fatalf("суусан апп хаагдлаа: %d", code)
	}
	// Кэш: DB false болсон ч TTL дотор true хэвээр.
	g.db = fakeDB{enabled: false}
	if code := gateReq(t, g); code != http.StatusOK {
		t.Fatalf("кэш ажилласангүй: %d", code)
	}
	// Invalidate → шинэ утга.
	g.Invalidate("t1")
	if code := gateReq(t, g); code != http.StatusForbidden {
		t.Fatalf("invalidate ажилласангүй: %d", code)
	}
}
