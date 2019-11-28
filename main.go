package main

import (
	"log"
	"os"

	"github.com/mendersoftware/workflows/server"
	"github.com/urfave/cli"
)

var workflowsPath string

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "workflows",
				Value:       "",
				Usage:       "Directory containing the definitions of workflows",
				Destination: &workflowsPath,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "server",
				Usage:  "Run the service as a server",
				Action: cmdServer,
			},
		},
	}
	app.Usage = "Workflows"
	app.Version = "1.0.0"

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdServer(args *cli.Context) error {
	if workflowsPath == "" {
		return cli.NewExitError(
			"Please specify the --workflows option",
			1)
	}

	server.InitAndRun(workflowsPath)
	return nil
}
