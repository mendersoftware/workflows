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
	"time"

	natsio "github.com/nats-io/nats.go"
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

	streamName := config.Config.GetString(dconfig.SettingNatsStreamName)
	topic := config.Config.GetString(dconfig.SettingNatsSubscriberTopic)
	subject := streamName + "." + topic
	durableName := config.Config.GetString(dconfig.SettingNatsSubscriberDurable)
	concurrency := conf.GetInt(dconfig.SettingConcurrency)

	cfg, err := natsClient.GetConsumerConfig(durableName)
	if err != nil {
		return err
	}
	notifyPeriod := cfg.AckWait * 8 / 10
	if notifyPeriod < time.Second {
		notifyPeriod = time.Second
	}

	channel := make(chan *natsio.Msg, concurrency)
	unsubscribe, err := natsClient.JetStreamSubscribe(
		ctx,
		subject,
		durableName,
		channel,
	)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to the nats JetStream")
	}
	defer func() {
		_ = unsubscribe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, unix.SIGINT, unix.SIGTERM)

	// Spawn worker pool
	var wg = new(sync.WaitGroup)
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		workerID := i
		go func() {
			defer func() {
				wg.Done()
				cancel() // Unblock main thread if it's waiting
			}()
			wl := l.F(log.Ctx{"worker_id": workerID})
			wCtx := log.WithContext(ctx, wl)

			wl.Info("worker starting up")
			workerMain(wCtx, channel, notifyPeriod, natsClient, dataStore)
			wl.Info("worker shut down")
		}()
	}

	// Run until a SIGTERM or SIGINT is received or one of the workers
	// stops unexpectedly
	select {
	case <-quit:
	case <-ctx.Done():
		err = ctx.Err()
	}
	// Notify workers that we're done
	close(channel)

	// Wait for all workers to finish their current task
	wg.Wait()
	return err
}

func workerMain(
	ctx context.Context,
	msgIn <-chan *natsio.Msg,
	notifyPeriod time.Duration,
	nc nats.Client,
	ds store.DataStore) {
	l := log.FromContext(ctx)
	sidecarChan := make(chan *natsio.Msg, 1)
	sidecarTimer := (*reusableTimer)(time.NewTimer(0))
	defer close(sidecarChan)
	done := ctx.Done()

	// workerSidecar is responsible for notifying the broker about slow workflows
	go workerSidecar(ctx, sidecarChan, notifyPeriod)
	var isOpen bool
	for {
		var msg *natsio.Msg
		select {
		case <-done:
			return
		case msg, isOpen = <-msgIn:
			if !isOpen {
				return
			}
			// Notify the sidecar routine about the new message
			select {
			case <-sidecarTimer.After(notifyPeriod / 8):
				l.Warn("timeout notifying sidecar routine about message")

			case sidecarChan <- msg:
			}
		}

		job := &model.Job{}
		err := json.Unmarshal(msg.Data, job)
		if err != nil {
			l.Error(errors.Wrap(err, "failed to unmarshall message"))
			if err := msg.Term(); err != nil {
				l.Error(errors.Wrap(err, "failed to term the message"))
			}
			continue
		}
		// process the job
		l.Infof("Worker: processing job %s workflow %s", job.ID, job.WorkflowName)
		err = processJob(ctx, job, ds, nc)
		if err != nil {
			l.Errorf("error: %v", err)
		}
		// Release message
		sidecarChan <- nil
		// stop the in progress ticker and ack the message
		if err := msg.AckSync(); err != nil {
			l.Error(errors.Wrap(err, "failed to ack the message"))
		}
	}
}

// workerSidecar helps notifying the NATS server about slow workflows.
// When workerMain picks up a new task, this routine is woken up and starts
// a timer that sends an "IN PROGRESS" package back to the broker if the worker
// takes too long.
func workerSidecar(ctx context.Context, msgIn <-chan *natsio.Msg, notifyPeriod time.Duration) {
	var (
		isOpen        bool
		msgInProgress *natsio.Msg
	)
	done := ctx.Done()
	t := (*reusableTimer)(time.NewTimer(0))
	for {
		select {
		case <-done:
			return
		case msgInProgress, isOpen = <-msgIn:
			if !isOpen {
				return
			}
		}
		for msgInProgress != nil {
			select {
			case <-t.After(notifyPeriod):
				_ = msgInProgress.InProgress()
			case <-done:
				return
			case msgInProgress, isOpen = <-msgIn:
				if !isOpen {
					return
				}
			}
		}
	}
}
