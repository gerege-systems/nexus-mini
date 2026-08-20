// Package audit — append-only, hash chain-тэй үйлдлийн бүртгэл.
// Бичилт audit_append() SECURITY DEFINER функцээр л явна; hash, цаг
// хоёулаа DB дотор тооцогдоно (docs/01-lessons.md #2).
package audit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

type Recorder struct {
	db nexus.DB
}

func NewRecorder(db nexus.DB) *Recorder { return &Recorder{db: db} }

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
	// Хүсэлтийн ctx цуцлагдсан ч (client тасрах, timeout) audit бичигдэх
	// ёстой — бизнес бичилт commit болчихоод audit нь алдагдвал hash chain
	// худал дүр зурагтай болно. Өөрийн 5с хугацаатай, цуцлагдашгүй ctx.
	base := context.WithoutCancel(ctx)
	wctx, cancel := context.WithTimeout(base, 5*time.Second)
	defer cancel()
	// TenantDB нь identity-ээс GUC тохируулдаг тул audit_append доторх
	// app_tenant_id() шалгалт (00005) энэ tenant-тай таарна.
	wctx = identity.With(wctx, tenantID, userID)
	_, err = r.db.Exec(wctx,
		`SELECT audit_append($1::uuid, $2::uuid, $3::varchar(128), $4::varchar(255), $5::jsonb)`,
		tenantID, uid, action, object, dj)
	if err != nil {
		log.Printf("audit: %s бичигдсэнгүй: %v", action, err)
	}
}
