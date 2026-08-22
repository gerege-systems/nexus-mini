package rbac

import "github.com/gerege-systems/nexus-mini/backend/pkg/nexus"

// CorePermissions — платформын өөрийн permission-ууд ("core." prefix,
// модуль биш). Tenant үүсэхэд GrantOnInstall-аар default оноолт нь хийгдэж,
// boot дээр appstore.Sync каталогт бүртгэнэ.
func CorePermissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: "core.members.manage", Name: "Гишүүд удирдах",
			Description: "Гишүүн нэмэх, хасах, role оноох"},
		{Code: "core.roles.manage", Name: "Role удирдах",
			Description: "Role үүсгэх, permission оноох"},
		{Code: "core.apps.manage", Name: "Апп удирдах",
			Description: "App store-оос суулгах, идэвхгүй болгох"},
		{Code: "core.audit.read", Name: "Audit лог харах",
			DefaultRoles: []string{"manager"}},
		{Code: "core.sso.manage", Name: "SSO клиент удирдах",
			Description: "OAuth2/OIDC клиент бүртгэх (гадны систем энэ платформоор нэвтрэх)"},
		{Code: "core.settings.manage", Name: "Байгууллагын тохиргоо",
			Description: "Байгууллагын нэр, профайл (регистр, хаяг, холбоо барих)"},
	}
}
