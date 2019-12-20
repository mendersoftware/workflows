// Copyright 2019 Northern.tech AS
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
	"time"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/urfave/cli"

	"github.com/mendersoftware/workflows/app/server"
	"github.com/mendersoftware/workflows/app/worker"
	dconfig "github.com/mendersoftware/workflows/config"
	store "github.com/mendersoftware/workflows/store/mongo"
)

func main() {
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

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdServer(args *cli.Context) error {
	ctx := context.Background()
	dbClient, err := store.NewMongoClient(ctx, config.Config)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to connect to db: %v", err),
			3)
	}
	defer disconnectClient(ctx, dbClient)
	err = doMigrations(ctx, dbClient, args.Bool("automigrate"))
	if err != nil {
		return err
	}
	dataStore := store.NewDataStoreWithClient(dbClient, config.Config)

	return server.InitAndRun(config.Config, dataStore)
}

func cmdWorker(args *cli.Context) error {
	ctx := context.Background()
	dbClient, err := store.NewMongoClient(ctx, config.Config)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to connect to db: %v", err),
			3)
	}
	defer disconnectClient(ctx, dbClient)
	err = doMigrations(ctx, dbClient, args.Bool("automigrate"))
	if err != nil {
		return err
	}
	dataStore := store.NewDataStoreWithClient(dbClient, config.Config)

	return worker.InitAndRun(config.Config, dataStore)
}

func cmdMigrate(args *cli.Context) error {
	ctx := context.Background()
	dbClient, err := store.NewMongoClient(ctx, config.Config)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to connect to db: %v", err),
			3)
	}
	defer disconnectClient(ctx, dbClient)
	err = doMigrations(ctx, dbClient, true)
	if err != nil {
		return err
	}

	return nil
}

func doMigrations(ctx context.Context, client *store.MongoClient,
	automigrate bool) error {
	db := config.Config.GetString(dconfig.SettingDbName)
	err := store.Migrate(ctx, db, store.DbVersion, client, automigrate)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to run migrations: %v", err),
			3)
	}

	return nil
}

func disconnectClient(parentCtx context.Context, client *store.MongoClient) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	client.Disconnect(ctx)
	cancel()
}
