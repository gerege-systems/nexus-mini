package tenantstate

// Байгууллагын төлөв: кэш, invalidate, устгалын товлол.

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStateCacheAndInvalidate(t *testing.T) {
	authURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_AUTH"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if authURL == "" || ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL_AUTH / _OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	authP, err := pgxpool.New(ctx, authURL)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	clean := func() { _, _ = owner.Exec(ctx, `DELETE FROM tenants WHERE slug = 'tstest'`) }
	clean()
	t.Cleanup(func() {
		clean()
		authP.Close()
		owner.Close()
	})
	var tid string
	if err := owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('tstest','Т') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}

	s := New(authP)
	st, err := s.Get(ctx, tid)
	if err != nil || st.Suspended || st.ReadOnly || st.DeletionAt != nil {
		t.Fatalf("шинэ tenant = %+v %v", st, err)
	}
	// DB-д өөрчилсөн ч кэш (30с) хуучныг өгнө.
	if _, err := owner.Exec(ctx, `UPDATE tenants SET suspended_at = now(), suspension_reason = 'шалтгаан', read_only = true WHERE id = $1::uuid`, tid); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Get(ctx, tid); st.Suspended {
		t.Fatal("кэш ажиллаагүй (шууд шинэчлэгдэв)")
	}
	// Invalidate → шинэ утга.
	s.InvalidateLocal(tid)
	st, err = s.Get(ctx, tid)
	if err != nil || !st.Suspended || !st.ReadOnly || st.Reason != "шалтгаан" {
		t.Fatalf("invalidate дараа = %+v %v", st, err)
	}
	// Устгалын товлол.
	if _, err := owner.Exec(ctx, `UPDATE tenants SET deletion_scheduled_at = now() + interval '30 days' WHERE id = $1::uuid`, tid); err != nil {
		t.Fatal(err)
	}
	s.InvalidateLocal(tid)
	st, _ = s.Get(ctx, tid)
	if st.DeletionAt == nil || st.DeletionAt.Before(time.Now()) {
		t.Fatalf("deletion_at = %v", st.DeletionAt)
	}
	// Notifier дуудагдана.
	var got string
	s.SetNotifier(func(id string) { got = id })
	s.Invalidate(tid)
	if got != tid {
		t.Fatalf("notifier = %q", got)
	}
	// Байхгүй tenant → алдаа.
	if _, err := s.Get(ctx, "00000000-0000-0000-0000-000000000000"); err == nil {
		t.Fatal("байхгүй tenant алдаа өгсөнгүй")
	}
}
