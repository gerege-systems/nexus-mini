// Package tenantstate — байгууллагын түдгэлзүүлсэн / зөвхөн-унших төлөв,
// хүсэлт бүрд шалгагддаг тул 30с кэштэй (bus-аар invalidate).
package tenantstate

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ttl = 30 * time.Second

type State struct {
	Suspended bool   `json:"suspended"`
	Reason    string `json:"reason,omitempty"`
	ReadOnly  bool   `json:"read_only"`
}

type entry struct {
	st State
	at time.Time
}

type Store struct {
	pool   *pgxpool.Pool
	mu     sync.Mutex
	cache  map[string]entry
	notify func(tenantID string)
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool, cache: map[string]entry{}} }

func (s *Store) SetNotifier(fn func(tenantID string)) { s.notify = fn }

// Get — кэшээс, хуучирсан бол DB-ээс (definer функц, RLS-ээс чөлөөтэй).
func (s *Store) Get(ctx context.Context, tenantID string) (State, error) {
	s.mu.Lock()
	if e, ok := s.cache[tenantID]; ok && time.Since(e.at) < ttl {
		s.mu.Unlock()
		return e.st, nil
	}
	s.mu.Unlock()
	var st State
	var reason *string
	err := s.pool.QueryRow(ctx,
		`SELECT suspended, reason, read_only FROM tenant_state($1::uuid)`, tenantID).
		Scan(&st.Suspended, &reason, &st.ReadOnly)
	if err != nil {
		return State{}, err
	}
	if reason != nil {
		st.Reason = strings.TrimSpace(*reason)
	}
	s.mu.Lock()
	if len(s.cache) >= 10000 {
		s.cache = map[string]entry{}
	}
	s.cache[tenantID] = entry{st: st, at: time.Now()}
	s.mu.Unlock()
	return st, nil
}

func (s *Store) Invalidate(tenantID string) {
	s.InvalidateLocal(tenantID)
	if s.notify != nil {
		s.notify(tenantID)
	}
}

func (s *Store) InvalidateLocal(tenantID string) {
	s.mu.Lock()
	delete(s.cache, tenantID)
	s.mu.Unlock()
}
