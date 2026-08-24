package identity

import (
	"context"
	"testing"
)

func TestIdentityContext(t *testing.T) {
	ctx := context.Background()
	if TenantID(ctx) != "" || UserID(ctx) != "" || Impersonator(ctx) != "" {
		t.Fatal("хоосон ctx-д утга гарав")
	}
	ctx = With(ctx, "t1", "u1")
	if TenantID(ctx) != "t1" || UserID(ctx) != "u1" {
		t.Fatalf("With = %q %q", TenantID(ctx), UserID(ctx))
	}
	if Impersonator(ctx) != "" {
		t.Fatal("impersonator автоматаар тавигдав")
	}
	ctx = WithImpersonator(ctx, "admin-1")
	if Impersonator(ctx) != "admin-1" || TenantID(ctx) != "t1" {
		t.Fatalf("WithImpersonator = %q, tenant %q", Impersonator(ctx), TenantID(ctx))
	}
	// Дарж бичих.
	ctx2 := With(ctx, "t2", "u2")
	if TenantID(ctx2) != "t2" || Impersonator(ctx2) != "admin-1" {
		t.Fatalf("дарж бичилт = %q / %q", TenantID(ctx2), Impersonator(ctx2))
	}
}
