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

package worker

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"

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
func InitAndRun(conf config.Reader, workflows Workflows, dataStore store.DataStore, nats nats.Client) error {
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
	channel, err := nats.JetStreamSubscribe(ctx, subject, durableName)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to the nats JetStream")
	}

	var msg interface{}
	concurrency := conf.GetInt(dconfig.SettingConcurrency)
	sem := make(chan bool, concurrency)
	quit := make(chan os.Signal, 1)
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
		switch msg := msg.(type) {
		case *natsio.Msg:
			job := &model.Job{}
			err := json.Unmarshal(msg.Data, job)
			if err != nil {
				l.Error(errors.Wrap(err, "failed to unmarshall message"))
				if err := msg.Term(); err != nil {
					l.Error(errors.Wrap(err, "failed to term the message"))
				}
				continue
			}
			sem <- true
			go func(ctx context.Context,
				job *model.Job, dataStore store.DataStore) {
				defer func() { <-sem }()
				l.Infof("Worker: processing job %s workflow %s", job.ID, job.WorkflowName)
				err := processJob(ctx, job, dataStore)
				if err != nil {
					l.Errorf("error: %v", err)
				}
				if err := msg.AckSync(); err != nil {
					l.Error(errors.Wrap(err, "failed to ack the message"))
				}
			}(ctx, job, dataStore)

		case error:
			return msg
		}
	}

	return nil
}
