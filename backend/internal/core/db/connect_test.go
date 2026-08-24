package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConnectAndClose(t *testing.T) {
	app, admin := os.Getenv("NEXUS_TEST_DATABASE_URL"), os.Getenv("NEXUS_TEST_DATABASE_URL_ADMIN")
	auth := os.Getenv("NEXUS_TEST_DATABASE_URL_AUTH")
	if app == "" || admin == "" || auth == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL / _ADMIN / _AUTH шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pools, err := Connect(ctx, app, admin, auth)
	if err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]interface{ Ping(context.Context) error }{
		"app": pools.App, "admin": pools.Admin, "auth": pools.Auth,
	} {
		if err := p.Ping(ctx); err != nil {
			t.Fatalf("%s pool ping: %v", name, err)
		}
	}
	pools.Close()

	// Аль нэг URL буруу бол бүгд хаагдана (нөөц алдагдахгүй).
	if _, err := Connect(ctx, app, admin, "postgres://байхгүй:1/x?connect_timeout=1"); err == nil {
		t.Fatal("буруу auth URL хүлээн авагдав")
	}
	if _, err := Connect(ctx, "тийм-биш://", admin, auth); err == nil {
		t.Fatal("буруу app URL хүлээн авагдав")
	}
}
