package organisation

import (
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

type handler struct{ deps nexus.Deps }

type departmentRow struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parent_id"`
	ManagerID   *string `json:"manager_membership_id"`
	ManagerName string  `json:"manager_name"`
	Active      bool    `json:"active"`
	People      int     `json:"people"`
}

type departmentInput struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	ManagerID *string `json:"manager_membership_id"`
	Active    *bool   `json:"active"`
}

func (in *departmentInput) valid() bool {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if in.ParentID != nil && *in.ParentID == "" {
		in.ParentID = nil
	}
	if in.ManagerID != nil && *in.ManagerID == "" {
		in.ManagerID = nil
	}
	if (in.ParentID != nil && !nexus.IsUUID(*in.ParentID)) || (in.ManagerID != nil && !nexus.IsUUID(*in.ManagerID)) {
		return false
	}
	return in.Code != "" && len(in.Code) <= 32 && in.Name != "" && len(in.Name) <= 120
}

type personRow struct {
	MembershipID   string  `json:"membership_id"`
	UserID         string  `json:"user_id"`
	Name           string  `json:"name"`
	DepartmentID   *string `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	JobTitle       string  `json:"job_title"`
}

type positionInput struct {
	DepartmentID *string `json:"department_id"`
	JobTitle     string  `json:"job_title"`
}

func (in *positionInput) valid() bool {
	in.JobTitle = strings.TrimSpace(in.JobTitle)
	if in.DepartmentID != nil && *in.DepartmentID == "" {
		in.DepartmentID = nil
	}
	if in.DepartmentID != nil && !nexus.IsUUID(*in.DepartmentID) {
		return false
	}
	return len(in.JobTitle) <= 120
}
