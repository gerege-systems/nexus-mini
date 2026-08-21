// Package core — nexus-mini платформын цөм, САН хэлбэрээр. Дистрибуц
// (энэ репогийн cmd/nexus-mini, эсвэл гадны компанийн өөрийн main.go)
// модулиудаа өгөөд Main-ийг дуудна:
//
//	func main() { core.Main(inventory.New(), payroll.New()) }
//
// Ингэснээр цөмийг fork хийхгүй — go.mod дахь хувилбарыг өсгөөд л
// шинэчлэлтийг авна. Коммандууд: migrate (миграц + env-д ADMIN_* байвал
// анхны админ), serve (API сервер). Зөвхөн Makefile-аар дуудагдана.
package core

import (
	"fmt"
	"os"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/envfile"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

const usage = `nexus-mini — платформын командын хэрэгсэл

Хэрэглээ (репогийн язгуураас, зөвхөн Makefile-аар):
  make migrate   Цөм + модулиудын миграцыг ажиллуулна; env-д
                 ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD байгаа бөгөөд
                 платформын админ хараахан байхгүй бол үүсгэнэ
  make serve     API серверийг асаана

Тохиргоо: backend/nexus-mini.env (эсвэл ENV_FILE=<зам>) файлыг уншина;
орчны хувьсагч файлаас дээгүүр үйлчилнэ. Загвар нь репогийн .env.example —
хуулж бөглөөд л болно.
`

// Main — CLI-ийн бүрэн оролт: модулиудаа бүртгээд os.Args-аар коммандаа
// ажиллуулна; алдаа бол stderr + exit 1. Дистрибуцийн main() үүнийг л дуудна.
func Main(modules ...nexus.Module) {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}
	cmd, args := os.Args[1], os.Args[2:]

	// Модулиуд бүх коммандад хэрэгтэй (миграц, permission sync). Register
	// нь буруу ID/permission/prefix дээр panic хийдэг — дистрибуц эхлэхдээ
	// л унана, ажиллаж байгаад биш.
	for _, m := range modules {
		nexus.Register(m)
	}

	var err error
	switch cmd {
	case "migrate":
		err = withEnv(args, cmdMigrate)
	case "serve":
		err = withEnv(args, cmdServe)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "үл мэдэх комманд: %s\n\n%s", cmd, usage)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "алдаа: %v\n", err)
		os.Exit(1)
	}
}

// withEnv — --env флагийг (default: ./nexus-mini.env) уншиж орчинд
// ачаалаад коммандаа ажиллуулна.
func withEnv(args []string, fn func(args []string) error) error {
	path := "nexus-mini.env"
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--env" && i+1 < len(args) {
			path = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	if err := envfile.Load(path); err != nil {
		return fmt.Errorf("%s уншиж чадсангүй: %w", path, err)
	}
	return fn(rest)
}
