// Copyright 2023 Northern.tech AS
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
	"encoding/json"
	"os"
	"os/signal"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"

	"github.com/mendersoftware/workflows/client/nats"
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
func InitAndRun(
	conf config.Reader,
	workflows Workflows,
	dataStore store.DataStore,
	natsClient nats.Client,
) error {
	ctx, cancel := context.WithCancel(context.Background())
	// Calling cancel() before returning should shut down
	// all workers. However, the new driver is not
	// particularly good at listening to the context in the
	// current state, but it'll be forced to shut down
	// eventually.
	defer cancel()

	if conf.GetBool(dconfig.SettingDebugLog) {
		log.Setup(true)
	}
	l := log.FromContext(ctx)

	err := dataStore.LoadWorkflows(ctx, l)
	if err != nil {
		return errors.Wrap(err, "failed to load workflows")
	}

	durableName := config.Config.GetString(dconfig.SettingNatsSubscriberDurable)
	concurrency := conf.GetInt(dconfig.SettingConcurrency)

	channel := make(chan []byte, concurrency)
	sub, err := natsClient.Subscribe(
		ctx,
		durableName,
		channel,
	)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to the nats JetStream")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, unix.SIGINT, unix.SIGTERM)

	workerError := make(chan error, 1)
	var errorOnce sync.Once

	// Spawn worker pool
	var wg = new(sync.WaitGroup)
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		workerID := i
		go func() (err error) {
			defer func() {
				wg.Done()
				errorOnce.Do(func() {
					workerError <- err
				})
			}()
			wl := l.F(log.Ctx{"worker_id": workerID})
			wCtx := log.WithContext(ctx, wl)

			wl.Info("worker starting up")
			err = workerMain(wCtx, channel, natsClient, dataStore)
			if err != nil {
				wl.Errorf("worker shut down with error: %s", err)
			} else {
				wl.Info("worker shut down")
			}
			return err
		}() // nolint:errcheck
	}

	go func() {
		err = sub.ListenAndServe()
		if err != nil {
			l.Errorf("subscription closed with error: %s", err)
		}
	}()

	// Run until a SIGTERM or SIGINT is received or one of the workers
	// stops unexpectedly
	select {
	case sig := <-quit:
		l.Errorf("received signal %s: terminating gracefully", sig)
		err = sub.Close()
	case err = <-workerError:
		sub.Close()
	case <-ctx.Done():
		err = ctx.Err()
	}

	// Wait for all workers to finish their current task
	wg.Wait()
	return err
}

func workerMain(
	ctx context.Context,
	msgIn <-chan []byte,
	nc nats.Client,
	ds store.DataStore,
) (err error) {
	l := log.FromContext(ctx)
	done := ctx.Done()

	var isOpen bool
	for {
		var msg []byte
		select {
		case <-done:
			return ctx.Err()
		case msg, isOpen = <-msgIn:
			if !isOpen {
				return nil
			}
		}

		job := &model.Job{}
		err = json.Unmarshal(msg, job)
		if err != nil {
			l.Error(errors.WithMessage(err, "failed to unmarshall message"))
			continue
		}
		// process the job
		l.Infof("Worker: processing job %s workflow %s", job.ID, job.WorkflowName)
		err = processJob(ctx, job, ds, nc)
		if err != nil {
			l.Errorf("error processing job: %s", err)
		}
	}
}
