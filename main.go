// Copyright 2020 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/urfave/cli"

	"github.com/mendersoftware/workflows/app/server"
	"github.com/mendersoftware/workflows/app/worker"
	dconfig "github.com/mendersoftware/workflows/config"
	store "github.com/mendersoftware/workflows/store/mongo"
)

func main() {
	doMain(os.Args)
}

func doMain(args []string) {
	var configPath string

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "Configuration `FILE`. Supports JSON, TOML, YAML and HCL formatted configs.",
				Value:       "config.yaml",
				Destination: &configPath,
			},
		},
		Commands: []cli.Command{
			{
				Name:   "server",
				Usage:  "Run the HTTP API server",
				Action: cmdServer,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "automigrate",
						Usage: "Run database migrations before starting.",
					},
				},
			},
			{
				Name:   "worker",
				Usage:  "Run the worker process",
				Action: cmdWorker,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "automigrate",
						Usage: "Run database migrations before starting.",
					},
					&cli.StringFlag{
						Name:  "workflows",
						Usage: "Comma-separated list of workflows executed by this worker",
					},
					&cli.StringFlag{
						Name:  "excluded-workflows",
						Usage: "Comma-separated list of workflows NOT executed by this worker",
					},
				},
			},
			{
				Name:   "migrate",
				Usage:  "Run the migrations",
				Action: cmdMigrate,
			},
		},
	}
	app.Usage = "Workflows"
	app.Version = "1.0.0"
	app.Action = cmdServer

	app.Before = func(args *cli.Context) error {
		err := config.FromConfigFile(configPath, dconfig.Defaults)
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("error loading configuration: %s", err),
				1)
		}

		// Enable setting config values by environment variables
		config.Config.SetEnvPrefix("WORKFLOWS")
		config.Config.AutomaticEnv()
		config.Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

		return nil
	}

	err := app.Run(args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdServer(args *cli.Context) error {
	dataStore, err := store.SetupDataStore(args.Bool("automigrate"))
	if err != nil {
		return err
	}
	defer dataStore.Close()
	return server.InitAndRun(config.Config, dataStore)
}

func cmdWorker(args *cli.Context) error {
	dataStore, err := store.SetupDataStore(args.Bool("automigrate"))
	if err != nil {
		return err
	}
	defer dataStore.Close()

	var included, excluded []string

	includedWorkflows := args.String("workflows")
	if includedWorkflows != "" {
		included = strings.Split(includedWorkflows, ",")
	}

	excludedWorkflows := args.String("excluded-workflows")
	if excludedWorkflows != "" {
		excluded = strings.Split(excludedWorkflows, ",")
	}

	workflows := worker.Workflows{
		Included: included,
		Excluded: excluded,
	}
	return worker.InitAndRun(config.Config, workflows, dataStore)
}

func cmdMigrate(args *cli.Context) error {
	_, err := store.SetupDataStore(true)
	if err != nil {
		return err
	}
	return nil
}
