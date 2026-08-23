package appstore

// Каталогийн "татаж авах боломжтой" (компиллогдоогүй) апп boot-ыг унагаахгүй
// байх ёстой — app_releases.app_id нь apps руу FK-тай тул дараалал чухал.
// Бодит DB шаардана (make check-db).

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSyncWithUninstallableCatalogApp(t *testing.T) {
	adminURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_ADMIN"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if adminURL == "" || ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL_ADMIN / _OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	owner, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	// Өмнөх ажиллалтын үлдэгдэл байвал apps мөр аль хэдийн байж FK-ийн замыг
	// далдалдаг — эхлэхдээ ч цэвэрлэнэ.
	clean := func() {
		_, _ = owner.Exec(ctx, `DELETE FROM app_releases WHERE app_id LIKE 'mn.itest.%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM apps WHERE id LIKE 'mn.itest.%'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")
	// Компиллогдоогүй апп: apps хүснэгтэд урьд нь байхгүй.
	body := `{"generated_at":"2026-01-01T00:00:00Z","apps":[{"id":"mn.itest.remote","short_id":"itestremote",
	 "name":"Татаж авах апп","version":"2.1.0","go_module":"example.com/itest","import":"example.com/itest",
	 "description":"тест","publisher":"itest","permissions":[{"code":"itestremote.read","name":"харах"}]}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Sync(ctx, admin, Source{URL: "off", FilePath: path}); err != nil {
		t.Fatalf("Sync компиллогдоогүй аппад унав: %v", err)
	}
	var compiled bool
	var version string
	if err := owner.QueryRow(ctx, `SELECT compiled, version FROM apps WHERE id = 'mn.itest.remote'`).Scan(&compiled, &version); err != nil {
		t.Fatalf("apps мөр алга: %v", err)
	}
	if compiled || version != "2.1.0" {
		t.Fatalf("compiled=%v version=%s", compiled, version)
	}
	var releases int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM app_releases WHERE app_id = 'mn.itest.remote'`).Scan(&releases); err != nil {
		t.Fatal(err)
	}
	if releases != 1 {
		t.Fatalf("app_releases = %d, 1 байх ёстой", releases)
	}
}
