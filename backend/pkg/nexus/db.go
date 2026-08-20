package nexus

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB — модулиудын өгөгдлийн сангийн гэрээ. Платформын хэрэгжүүлэлт дуудлага
// бүрийг гүйлгээнд ороож, context-ийн tenant/user-ийг RLS-ийн
// `app.tenant_id` / `app.user_id` тохиргоонд SET LOCAL хийдэг — модуль
// WHERE tenant_id гэж бичихээ мартсан ч RLS хамгаална (бичих нь зөв хэвээр:
// индекс ашиглалт сайжирна).
//
// pgx-ийн төрлүүд энд санаатай ил байгаа: pgx бол платформын адислагдсан
// драйвер, түүнийг нуусан хийсвэрлэл модульд юу ч өгөхгүй.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	// Tx нь олон үйлдлийг нэг гүйлгээнд (tenant context тохируулсан) ажиллуулна.
	Tx(ctx context.Context, fn func(tx pgx.Tx) error) error
}
