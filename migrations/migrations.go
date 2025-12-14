package migrations

import (
	"embed"
)

//go:embed *.sql
var SqlMigrations embed.FS
