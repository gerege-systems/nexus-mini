// Package rbac — эрхийн шийдвэр (docs/02-rbac.md).
//
// Үр дүнтэй эрх = гишүүнчлэлийн role-ууд + тэдгээрийн implies гинж →
// role_permissions → map[code]Grant. Нэг код олон замаар ирвэл өргөн
// scope (all) нь ялна. 30 секунд кэшлэнэ.
package rbac

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

const grantTTL = 30 * time.Second

type cacheEntry struct {
	grants map[string]nexus.Grant
	at     time.Time
}

type Store struct {
	db     nexus.DB // TenantDB — RLS context-оо өөрөө тохируулдаг
	mu     sync.Mutex
	cache  map[string]cacheEntry
	notify func(tenantID string) // бусад процесст мэдэгдэх (bus); nil бол зөвхөн локал
}

// SetNotifier — Invalidate бүрд дуудагдах cross-process мэдэгдэгч.
func (s *Store) SetNotifier(fn func(tenantID string)) { s.notify = fn }

func NewStore(db nexus.DB) *Store {
	return &Store{db: db, cache: make(map[string]cacheEntry)}
}

var _ nexus.PermissionStore = (*Store)(nil)

// Invalidate — tenant-ийн бүх кэшийг унагаана (role/оноолт өөрчлөгдөхөд)
// ба бусад процесст мэдэгдэнэ.
func (s *Store) Invalidate(tenantID string) {
	s.InvalidateLocal(tenantID)
	if s.notify != nil {
		s.notify(tenantID)
	}
}

// InvalidateLocal — зөвхөн энэ процессын кэш (bus-аас ирсэн мэдэгдэлд).
func (s *Store) InvalidateLocal(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.cache {
		if strings.HasPrefix(k, tenantID+":") {
			delete(s.cache, k)
		}
	}
}

func (s *Store) UserGrants(ctx context.Context, tenantID, userID string) (map[string]nexus.Grant, error) {
	if tenantID == "" || userID == "" {
		return map[string]nexus.Grant{}, nil
	}
	key := tenantID + ":" + userID
	s.mu.Lock()
	if e, ok := s.cache[key]; ok && time.Since(e.at) < grantTTL {
		s.mu.Unlock()
		return clone(e.grants), nil
	}
	s.mu.Unlock()

	// Гишүүнчлэлийн шууд role-уудаас implies гинжээр өвлөгдсөн role хүртэл
	// (гүн ≤ 5 — санамсаргүй мөчлөгөөс хамгаална), дараа нь grant-ууд.
	// TenantDB нь ctx-ийн identity-гээр RLS context тохируулдаг тул энд
	// шийдэж буй хосыг нь ctx-д тавьж өгнө.
	qctx := identity.With(ctx, tenantID, userID)
	rows, err := s.db.Query(qctx, `
		WITH RECURSIVE chain AS (
		  SELECT r.id, r.code, r.implies, 0 AS depth
		    FROM memberships m
		    JOIN membership_roles mr ON mr.membership_id = m.id
		    JOIN roles r ON r.id = mr.role_id AND r.tenant_id = $1::uuid AND r.active
		   WHERE m.tenant_id = $1::uuid AND m.user_id = $2::uuid
		  UNION ALL
		  SELECT r.id, r.code, r.implies, c.depth + 1
		    FROM chain c
		    JOIN roles r ON r.tenant_id = $1::uuid AND r.code = c.implies AND r.active
		   WHERE c.depth < 5
		)
		SELECT rp.permission_code, rp.scope
		  FROM (SELECT DISTINCT id FROM chain) cr
		  JOIN role_permissions rp ON rp.role_id = cr.id`,
		tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grants := make(map[string]nexus.Grant)
	for rows.Next() {
		var code, scope string
		if err := rows.Scan(&code, &scope); err != nil {
			return nil, err
		}
		g := nexus.Grant{Allowed: true, Scope: nexus.ScopeKind(scope)}
		if prev, ok := grants[code]; ok && prev.Scope == nexus.ScopeAll {
			continue // өргөн scope аль хэдийн байна
		}
		grants[code] = g
	}
	if err := rows.Err(); err != nil {
		// Тасарсан урсгал = дутуу эрхийн жагсаалт; кэшлэвэл 30 секунд
		// хүнийг ажлаас нь хаана — кэшлэхгүй, алдаагаар буцаана.
		return nil, err
	}

	s.mu.Lock()
	if len(s.cache) >= 10000 {
		// Хязгааргүй өсөхөөс сэргийлсэн бүдүүлэг тааз — TTL богино тул
		// бүхэлд нь хаясан ч дараагийн хүсэлтүүд сэргээнэ.
		s.cache = make(map[string]cacheEntry)
	}
	s.cache[key] = cacheEntry{grants: grants, at: time.Now()}
	s.mu.Unlock()
	return clone(grants), nil
}

func clone(m map[string]nexus.Grant) map[string]nexus.Grant {
	out := make(map[string]nexus.Grant, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
