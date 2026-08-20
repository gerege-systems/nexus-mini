package nexus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeStore struct {
	grants map[string]Grant
	err    error
}

func (f fakeStore) UserGrants(context.Context, string, string) (map[string]Grant, error) {
	return f.grants, f.err
}

func doReq(t *testing.T, store PermissionStore, code string) (*httptest.ResponseRecorder, ScopeKind) {
	t.Helper()
	var seen ScopeKind
	h := RequirePermission(store, code)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = Scope(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(WithIdentity(req.Context(), "t1", "u1"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, seen
}

func TestRequirePermission(t *testing.T) {
	store := fakeStore{grants: map[string]Grant{
		"m.read":   {Allowed: true, Scope: ScopeAll},
		"m.manage": {Allowed: true, Scope: ScopeOwn},
	}}

	if rec, scope := doReq(t, store, "m.read"); rec.Code != 200 || scope != ScopeAll {
		t.Fatalf("all grant: code=%d scope=%s", rec.Code, scope)
	}
	if rec, scope := doReq(t, store, "m.manage"); rec.Code != 200 || scope != ScopeOwn {
		t.Fatalf("own grant: code=%d scope=%s", rec.Code, scope)
	}
	if rec, _ := doReq(t, store, "m.delete"); rec.Code != http.StatusForbidden {
		t.Fatalf("grant-гүй permission нэвтэрлээ: %d", rec.Code)
	}
	if rec, _ := doReq(t, fakeStore{err: errors.New("db down")}, "m.read"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("store алдаа 500 биш: %d", rec.Code)
	}
}
