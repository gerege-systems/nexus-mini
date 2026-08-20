package devices

import "strings"

// deviceRow — жагсаалт/хариултын мөр.
type deviceRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Serial    string `json:"serial"`
	Status    string `json:"status"`
	Note      string `json:"note"`
	CreatedBy string `json:"created_by"`
	OwnerName string `json:"owner_name"`
	CreatedAt string `json:"created_at"`
}

// deviceInput — үүсгэх/засах хүсэлтийн бие.
type deviceInput struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Serial string `json:"serial"`
	Status string `json:"status"`
	Note   string `json:"note"`
}

// valid — хилийн validation: цэвэрлээд дүрмээ шалгана.
func (in *deviceInput) valid() bool {
	in.Name = strings.TrimSpace(in.Name)
	in.Serial = strings.TrimSpace(in.Serial)
	if in.Status == "" {
		in.Status = "active"
	}
	switch in.Status {
	case "active", "repair", "lost", "retired":
	default:
		return false
	}
	return in.Name != "" && in.Serial != ""
}
