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

package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli"

	"github.com/mendersoftware/go-lib-micro/config"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/store"
	"github.com/mendersoftware/workflows/workflow"
)

// Workflows maps active workflow names and Workflow structs
var Workflows map[string]*workflow.Workflow

// InitAndRun initializes the server and runs it
func InitAndRun(conf config.Reader, dataStore store.DataStoreInterface) error {
	var workflowsPath string = conf.GetString(dconfig.SettingWorkflowsPath)
	if workflowsPath == "" {
		return cli.NewExitError(
			"Please specify the workflows path in the configuration file",
			1)
	}
	Workflows = workflow.GetWorkflowsFromPath(workflowsPath)

	var listen = conf.GetString(dconfig.SettingListen)
	var router = NewRouter(dataStore)
	srv := &http.Server{
		Addr:    listen,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown: ", err)
	}

	return nil
}
