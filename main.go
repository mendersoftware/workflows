// Copyright 2021 Northern.tech AS
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
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/mendersoftware/workflows/app/server"
	"github.com/mendersoftware/workflows/app/worker"
	"github.com/mendersoftware/workflows/client/nats"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
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
				Name: "config",
				Usage: "Configuration `FILE`." +
					" Supports JSON, TOML, YAML and HCL formatted configs.",
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
			{
				Name:   "list-jobs",
				Usage:  "List jobs",
				Action: cmdListJobs,
				Flags: []cli.Flag{
					cli.Int64Flag{
						Name:  "page",
						Usage: "page number to show",
					},
					cli.Int64Flag{
						Name:  "perPage",
						Usage: "number of results per page",
					},
				},
			},
		},
	}
	app.Usage = "Workflows"
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

func getNatsClient() (nats.Client, error) {
	natsURI := config.Config.GetString(dconfig.SettingNatsURI)
	nats, err := nats.NewClientWithDefaults(natsURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to nats")
	}

	streamName := config.Config.GetString(dconfig.SettingNatsStreamName)
	nats = nats.WithStreamName(streamName)
	err = nats.JetStreamCreateStream(nats.StreamName())
	if err != nil {
		return nil, err
	}
	return nats, nil
}

func cmdServer(args *cli.Context) error {
	dataStore, err := store.SetupDataStore(args.Bool("automigrate"))
	if err != nil {
		return err
	}
	defer dataStore.Close()

	nats, err := getNatsClient()
	if err != nil {
		return err
	}
	defer nats.Close()

	return server.InitAndRun(config.Config, dataStore, nats)
}

func cmdWorker(args *cli.Context) error {
	dataStore, err := store.SetupDataStore(args.Bool("automigrate"))
	if err != nil {
		return err
	}
	defer dataStore.Close()

	nats, err := getNatsClient()
	if err != nil {
		return err
	}
	defer nats.Close()

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
	return worker.InitAndRun(config.Config, workflows, dataStore, nats)
}

func cmdMigrate(args *cli.Context) error {
	_, err := store.SetupDataStore(true)
	if err != nil {
		return err
	}
	return nil
}

func cmdListJobs(args *cli.Context) error {
	dataStore, err := store.SetupDataStore(false)
	if err != nil {
		return err
	}
	defer dataStore.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var page int64
	var perPage int64
	page = args.Int64("page")
	perPage = args.Int64("perPage")

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 75
	}
	jobs, count, _ := dataStore.GetAllJobs(ctx, page, perPage)
	fmt.Printf("all jobs: %d; page: %d/%d perPage:%d\n%29s %24s %10s %s\n",
		count, page, count/perPage, perPage, "insert time", "id", "status", "workflow")
	for _, j := range jobs {
		format := "Mon, 2 Jan 2006 15:04:05 MST"
		fmt.Printf(
			"%29s %24s %10s %s\n",
			j.InsertTime.Format(format),
			j.ID, model.StatusToString(j.Status),
			j.WorkflowName,
		)
	}
	fmt.Printf("all jobs: %d; page: %d/%d\n", count, page, count/perPage)

	return nil
}
