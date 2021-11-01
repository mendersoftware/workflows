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

package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	natsio "github.com/nats-io/nats.go"
)

const (
	// Set reconnect buffer size in bytes (10 MB)
	reconnectBufSize = 10 * 1024 * 1024
	// Set reconnect interval to 1 second
	reconnectWaitTime = 1 * time.Second
	// Set the number of redeliveries for a message
	maxDeliver = 3
	// Set the ACK wait
	ackWait = 30 * time.Second
)

// Client is the nats client
//go:generate ../../utils/mockgen.sh
type Client interface {
	WithStreamName(streamName string) Client
	StreamName() string
	IsConnected() bool
	JetStreamCreateStream(streamName string) error
	JetStreamSubscribe(ctx context.Context, subj, durable string) (<-chan interface{}, error)
	JetStreamPublish(string, []byte) error
}

// NewClient returns a new nats client
func NewClient(url string, opts ...natsio.Option) (Client, error) {
	natsClient, err := natsio.Connect(url, opts...)
	if err != nil {
		return nil, err
	}
	js, err := natsClient.JetStream()
	if err != nil {
		return nil, err
	}
	return &client{
		nats: natsClient,
		js:   js,
	}, nil
}

// NewClient returns a new nats client with default options
func NewClientWithDefaults(url string) (Client, error) {
	natsClient, err := NewClient(url,
		natsio.ReconnectBufSize(reconnectBufSize),
		natsio.ReconnectWait(reconnectWaitTime),
	)
	if err != nil {
		return nil, err
	}
	return natsClient, nil
}

type client struct {
	nats       *natsio.Conn
	js         natsio.JetStreamContext
	streamName string
}

// IsConnected returns true if the client is connected to nats
func (c *client) WithStreamName(streamName string) Client {
	c.streamName = streamName
	return c
}

// IsConnected returns true if the client is connected to nats
func (c *client) StreamName() string {
	return c.streamName
}

// IsConnected returns true if the client is connected to nats
func (c *client) IsConnected() bool {
	return c.nats.IsConnected()
}

// JetStreamCreateStream creates a stream
func (c *client) JetStreamCreateStream(streamName string) error {
	stream, err := c.js.StreamInfo(streamName)
	if err != nil && err != natsio.ErrStreamNotFound {
		return err
	}
	if stream == nil {
		_, err = c.js.AddStream(&nats.StreamConfig{
			Name:      streamName,
			NoAck:     false,
			MaxAge:    24 * time.Hour,
			Retention: nats.WorkQueuePolicy,
			Storage:   nats.FileStorage,
			Subjects:  []string{streamName + ".*"},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// JetStreamSubscribe subscribes to messages from the given subject with a durable subscriber
func (c *client) JetStreamSubscribe(ctx context.Context, subj, durable string) (<-chan interface{}, error) {
	var retChan = make(chan interface{})
	var channel = make(chan interface{})

	go func() {
		sub, err := c.js.QueueSubscribe(subj, durable, func(msg *natsio.Msg) {
			channel <- msg
		},
			natsio.AckExplicit(),
			natsio.AckWait(ackWait),
			natsio.ManualAck(),
			natsio.MaxDeliver(maxDeliver),
		)
		if err != nil {
			retChan <- err
			return
		}
		defer func() {
			_ = sub.Unsubscribe()
		}()

		retChan <- nil
		<-ctx.Done()
	}()

	ret := <-retChan
	switch ret := ret.(type) {
	case error:
		return nil, ret
	default:
		return channel, nil
	}
}

// JetStreamPublish publishes a message to the given subject
func (c *client) JetStreamPublish(subj string, data []byte) error {
	_, err := c.js.Publish(subj, data)
	return err
}