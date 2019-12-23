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

package worker

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

// InitAndRun initializes the worker and runs it
func InitAndRun(conf config.Reader, dataStore store.DataStore) error {
	ctx := context.Background()
	channel := dataStore.GetJobs(ctx)
	l := log.FromContext(ctx)

	var job *model.Job
	concurrency := conf.GetInt(dconfig.SettingConcurrency)
	sem := make(chan bool, concurrency)
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case job = <-channel:

		case <-quit:
			l.Info("Shutdown Worker ...")
			dataStore.Shutdown()
			return nil
		}
		if job == nil {
			break
		}
		sem <- true
		go func(ctx context.Context, job *model.Job, dataStore store.DataStore) {
			defer func() { <-sem }()
			processJob(ctx, job, dataStore)
		}(ctx, job, dataStore)
	}

	return nil
}
