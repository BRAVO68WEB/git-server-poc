package commands

import "github.com/urfave/cli/v3"

func ServeCommand() *cli.Command {
	return &cli.Command{
		Name:   "serve",
		Usage:  "Start the Git server",
		Action: nil,
	}
}
