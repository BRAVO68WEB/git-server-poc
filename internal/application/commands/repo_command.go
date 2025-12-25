package commands

import "github.com/urfave/cli/v3"

func RepoCommands() *cli.Command {
	return &cli.Command{
		Name:  "repo",
		Usage: "Manage repositories",
		Commands: []*cli.Command{
			Create(),
			Delete(),
		},
	}
}

func Create() *cli.Command {
	return &cli.Command{
		Name: "create",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Name of the repository",
				Required: true,
				Action:   nil,
			},
			&cli.StringFlag{
				Name:     "description",
				Aliases:  []string{"d"},
				Usage:    "Description of the repository",
				Required: false,
				Action:   nil,
			},
			&cli.BoolFlag{
				Name:   "isPrivate",
				Usage:  "Set repository as private",
				Value:  false,
				Action: nil,
			},
		},
	}
}

func Delete() *cli.Command {
	return &cli.Command{
		Name: "delete",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Name of the repository to delete",
				Required: true,
				Action:   nil,
			},
		},
	}
}
