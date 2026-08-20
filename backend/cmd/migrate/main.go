// cmd/migrate — цөм + модуль бүрийн миграцыг ажиллуулна.
// DATABASE_URL_OWNER (nexus_owner)-ээр холбогдоно; модуль бүр өөрийн
// goose хүснэгттэй (goose_<shortid>) тул цөмтэй мөргөлдөхгүй.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	coredb "github.com/gerege-systems/nexus-mini/backend/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/modules"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	url := os.Getenv("DATABASE_URL_OWNER")
	if url == "" {
		log.Fatal("DATABASE_URL_OWNER тохируулаагүй байна")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	goose.SetDialect("postgres")

	// Цөм.
	goose.SetBaseFS(coredb.Migrations)
	goose.SetTableName("goose_db_version")
	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatalf("цөмийн миграц: %v", err)
	}
	fmt.Println("цөм: ok")

	// Модулиуд.
	modules.RegisterAll()
	for _, m := range nexus.Registered() {
		goose.SetBaseFS(m.Migrations())
		goose.SetTableName("goose_" + m.ShortID())
		if err := goose.Up(db, "migrations"); err != nil {
			log.Fatalf("%s миграц: %v", m.ID(), err)
		}
		fmt.Printf("%s: ok\n", m.ShortID())
	}
}
