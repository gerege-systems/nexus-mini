// nexus-mini — платформын командын хэрэгсэл.
//
//	nexus-mini migrate   миграц + (env-д ADMIN_* байвал) анхны админ
//	nexus-mini admin     платформын админ үүсгэх / өргөмжлөх
//	nexus-mini serve     API сервер асаах
package main

import (
	"fmt"
	"os"

	"github.com/gerege-systems/nexus-mini/backend/internal/modules"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/envfile"
)

const usage = `nexus-mini — платформын командын хэрэгсэл

Хэрэглээ:
  nexus-mini migrate   Цөм + модулиудын миграцыг ажиллуулна; env-д
                       ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD байгаа бөгөөд
                       платформын админ хараахан байхгүй бол үүсгэнэ
  nexus-mini admin     Платформын админ үүсгэх/өргөмжлөх
                       (--email --name --password эсвэл интерактив)
  nexus-mini serve     API серверийг асаана

Тохиргоо: коммандууд ажлын хавтаснаас nexus-mini.env (эсвэл --env <зам>)
файлыг уншина; орчны хувьсагч файлаас дээгүүр үйлчилнэ. Загвар нь
репогийн .env.example — хуулж бөглөөд л болно.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}
	cmd, args := os.Args[1], os.Args[2:]

	// Модулиуд бүх коммандад хэрэгтэй (миграц, permission sync).
	modules.RegisterAll()

	var err error
	switch cmd {
	case "migrate":
		err = withEnv(args, cmdMigrate)
	case "admin":
		err = withEnv(args, cmdAdmin)
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
