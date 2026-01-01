package main

import (
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
	"github.com/bravo68web/stasis/internal/domain/models"
)

func main() {
	stmts, err := gormschema.New("postgres").Load(
		&models.User{},
		&models.Repository{},
		&models.SSHKey{},
		&models.Token{},
		&models.CIJob{},
		&models.CIArtifact{},
		&models.CIJobLog{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}

	io.WriteString(os.Stdout, stmts)
}
