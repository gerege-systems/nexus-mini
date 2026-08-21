package handlers

import (
	"net/http"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
)

// Байгууллагын профайл (OGN-ийн tenant_profiles-оос авсан): хуулийн нэр,
// регистр, ТТД, хаяг, холбоо барих. Унших — гишүүн бүр; бичих —
// core.settings.manage.

type tenantProfile struct {
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	LegalName          string `json:"legal_name"`
	RegistrationNumber string `json:"registration_number"`
	TaxNumber          string `json:"tax_number"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	Website            string `json:"website"`
}

func (p *tenantProfile) clean() bool {
	p.Name = strings.TrimSpace(p.Name)
	for _, f := range []*string{&p.LegalName, &p.RegistrationNumber, &p.TaxNumber, &p.Address, &p.Phone, &p.Email, &p.Website} {
		*f = strings.TrimSpace(*f)
	}
	if p.Email != "" && !emailRe.MatchString(p.Email) {
		return false
	}
	return p.Name != "" && len(p.Name) <= 160 && len(p.LegalName) <= 200 &&
		len(p.RegistrationNumber) <= 32 && len(p.TaxNumber) <= 32 && len(p.Address) <= 500 &&
		len(p.Phone) <= 32 && len(p.Email) <= 255 && len(p.Website) <= 255
}

// GET /api/tenant/profile
func (h *Auth) TenantProfile(w http.ResponseWriter, r *http.Request) {
	var p tenantProfile
	err := h.DB.QueryRow(r.Context(), `
		SELECT t.name, t.slug,
		       coalesce(p.legal_name, ''), coalesce(p.registration_number, ''), coalesce(p.tax_number, ''),
		       coalesce(p.address, ''), coalesce(p.phone, ''), coalesce(p.email, ''), coalesce(p.website, '')
		  FROM tenants t LEFT JOIN tenant_profiles p ON p.tenant_id = t.id
		 WHERE t.id = $1::uuid`, nexus.TenantID(r.Context())).
		Scan(&p.Name, &p.Slug, &p.LegalName, &p.RegistrationNumber, &p.TaxNumber, &p.Address, &p.Phone, &p.Email, &p.Website)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "profile query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// PUT /api/tenant/profile — нэр + профайл (upsert). Slug өөрчлөгдөхгүй.
func (h *Auth) UpdateTenantProfile(w http.ResponseWriter, r *http.Request) {
	var p tenantProfile
	if !httpx.Decode(w, r, &p) {
		return
	}
	if !p.clean() {
		httpx.Error(w, http.StatusBadRequest, "нэр шаардлагатай; талбарын урт/имэйл буруу")
		return
	}
	tenantID := nexus.TenantID(r.Context())
	err := h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(),
			`UPDATE tenants SET name = $2::varchar(160) WHERE id = $1::uuid`, tenantID, p.Name); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO tenant_profiles (tenant_id, legal_name, registration_number, tax_number, address, phone, email, website)
			VALUES ($1::uuid, $2::varchar(200), $3::varchar(32), $4::varchar(32), $5::varchar(500), $6::varchar(32), $7::varchar(255), $8::varchar(255))
			ON CONFLICT (tenant_id) DO UPDATE SET
			  legal_name = excluded.legal_name, registration_number = excluded.registration_number,
			  tax_number = excluded.tax_number, address = excluded.address, phone = excluded.phone,
			  email = excluded.email, website = excluded.website, updated_at = now()`,
			tenantID, p.LegalName, p.RegistrationNumber, p.TaxNumber, p.Address, p.Phone, p.Email, p.Website)
		return err
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "profile save failed")
		return
	}
	h.Audit.Record(r.Context(), "tenant.profile", p.Slug, map[string]any{"name": p.Name})
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
