// Package migrate — цөм + бүртгэгдсэн модуль бүрийн goose миграцыг
// owner холболтоор ажиллуулна. CLI-ийн migrate дуудна.
package migrate

import (
	"database/sql"
	"fmt"

	coredb "github.com/gerege-systems/nexus-mini/backend/db"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunAll — ownerURL дээр цөмийн дараа бүртгэгдсэн модулиудын миграцыг
// ажиллуулна. Модуль бүр өөрийн goose хүснэгттэй (goose_<shortid>).
func RunAll(ownerURL string, logf func(format string, args ...any)) error {
	return Run(ownerURL, logf, nexus.Registered()...)
}

// Run — цөм + өгөгдсөн модулиудын миграц. Тест DB-гээ өөрсдөө бэлтгэдэг
// модулийн тестүүд бүртгэлгүйгээр дуудна (Register давхардал, глобал төлөв үгүй).
func Run(ownerURL string, logf func(format string, args ...any), modules ...nexus.Module) error {
	db, err := sql.Open("pgx", ownerURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("owner холболт: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	goose.SetBaseFS(coredb.Migrations)
	goose.SetTableName("goose_db_version")
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("цөмийн миграц: %w", err)
	}
	logf("миграц: цөм ok")

	for _, m := range modules {
		goose.SetBaseFS(m.Migrations())
		goose.SetTableName("goose_" + m.ShortID())
		if err := goose.Up(db, "migrations"); err != nil {
			return fmt.Errorf("%s миграц: %w", m.ID(), err)
		}
		logf("миграц: %s ok", m.ShortID())
	}
	return nil
}
