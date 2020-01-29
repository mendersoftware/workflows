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

package worker

import (
	"context"
	"os"
	"os/signal"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

// Workflows filters workflows executed by a worker
type Workflows struct {
	Included []string
	Excluded []string
}

// InitAndRun initializes the worker and runs it
func InitAndRun(conf config.Reader, workflows Workflows, dataStore store.DataStore) error {
	ctx, cancel := context.WithCancel(context.Background())
	// Calling cancel() before returning should shut down
	// all workers. However, the new driver is not
	// particularly good at listening to the context in the
	// current state, but it'll be forced to shut down
	// eventually.
	defer cancel()

	dataStore.LoadWorkflows(ctx)

	channel, err := dataStore.GetJobs(ctx, workflows.Included, workflows.Excluded)
	if err != nil {
		return errors.Wrap(err, "Failed to start job scheduler")
	}
	l := log.FromContext(ctx)

	var msg interface{}
	concurrency := conf.GetInt(dconfig.SettingConcurrency)
	sem := make(chan bool, concurrency)
	quit := make(chan os.Signal)
	signal.Notify(quit, unix.SIGINT, unix.SIGTERM)
	for {
		select {
		case msg = <-channel:

		case <-quit:
			signal.Stop(quit)
			l.Info("Shutdown Worker ...")
			return err
		}
		if msg == nil {
			break
		}
		switch msg.(type) {
		case *model.Job:
			job := msg.(*model.Job)
			sem <- true
			go func(ctx context.Context,
				job *model.Job, dataStore store.DataStore) {
				defer func() { <-sem }()
				processJob(ctx, job, dataStore)
			}(ctx, job, dataStore)

		case error:
			return msg.(error)
		}
	}

	return err
}
