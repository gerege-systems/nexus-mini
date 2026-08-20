// Package db — цөмийн миграцуудын embed.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
