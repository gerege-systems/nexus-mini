// Package audit — append-only, hash chain-тэй үйлдлийн бүртгэл.
// Бичилт audit_append() SECURITY DEFINER функцээр л явна; hash, цаг
// хоёулаа DB дотор тооцогдоно (docs/01-lessons.md #2).
package audit

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Recorder struct {
	pool *pgxpool.Pool
}

func NewRecorder(pool *pgxpool.Pool) *Recorder { return &Recorder{pool: pool} }

var _ nexus.AuditRecorder = (*Recorder)(nil)

// Record — context-ийн tenant/user-ээр бичнэ. Audit бичигдэхгүй байх нь
// үндсэн үйлдлийг унагаах шалтгаан биш тул алдааг логлоод залгина.
func (r *Recorder) Record(ctx context.Context, action, object string, details map[string]any) {
	tenantID := nexus.TenantID(ctx)
	if tenantID == "" {
		log.Printf("audit: tenant-гүй context дээр %s орхигдов", action)
		return
	}
	r.RecordAs(ctx, tenantID, nexus.UserID(ctx), action, object, details)
}

// RecordAs — tenant/user-ийг ил зааж бичнэ (signup гэх мэт context-д
// identity нь хараахан суугаагүй урсгалд).
func (r *Recorder) RecordAs(ctx context.Context, tenantID, userID, action, object string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	dj, err := json.Marshal(details)
	if err != nil {
		log.Printf("audit: details marshal: %v", err)
		dj = []byte("{}")
	}
	var uid any
	if userID != "" {
		uid = userID
	}
	_, err = r.pool.Exec(ctx,
		`SELECT audit_append($1::uuid, $2::uuid, $3::varchar(128), $4::varchar(255), $5::jsonb)`,
		tenantID, uid, action, object, dj)
	if err != nil {
		log.Printf("audit: %s бичигдсэнгүй: %v", action, err)
	}
}
