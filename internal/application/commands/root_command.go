package commands

import (
	"context"

	"github.com/urfave/cli/v3"
)

type CommandRegistry struct {
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{}
}

func (*CommandRegistry) RegisterCLI() *cli.Command {
	return &cli.Command{
		Name:                  "dekha-jayega",
		Suggest:               true,
		EnableShellCompletion: true,
		Action:                RootCommand(),
		Commands: []*cli.Command{
			RepoCommands(),
			ServeCommand(),
		},
	}
}

func RootCommand() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		cmd.Writer.Write([]byte("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		cmd.Writer.Write([]byte("Welcome to Git-Server CLI!\n"))
		cmd.Writer.Write([]byte("Use 'Git-Server --help' to see available commands.\n"))
		cmd.Writer.Write([]byte("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		return nil
	}
}
