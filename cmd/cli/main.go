package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "git-server",
		Usage: "A simple Git server application",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cmd.Writer.Write([]byte("Git server CLI\n"))
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
