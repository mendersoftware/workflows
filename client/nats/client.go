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

package nats

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mendersoftware/go-lib-micro/log"
)

var (
	ErrNoConsumer           = errors.New("nats: consumer does not exist")
	ErrConsumerExist        = errors.New("nats: consumer already exist")
	ErrConsumerIncompatible = errors.New("nats: consumer configuration is incompatible")
)

const (
	// Set reconnect buffer size in bytes (10 MB)
	reconnectBufSize = 10 * 1024 * 1024
	// Set reconnect interval to 1 second
	reconnectWaitTime = 1 * time.Second
)

// Client is the nats client
//
//go:generate ../../utils/mockgen.sh
type Client interface {
	Close()
	IsConnected() bool
	StreamName() string
	CreateStream() error
	CreateConsumer(consumerName string, cfg ConsumerConfig) error
	Subscribe(
		ctx context.Context,
		consumerName string,
		q chan<- []byte,
	) (Subscription, error)
	Publish(string, []byte) error
}

// NewClient returns a new nats client
func NewClient(url string, streamName string, opts ...nats.Option) (Client, error) {
	natsClient, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}
	js, _ := natsClient.JetStream()
	return &client{
		nats:       natsClient,
		js:         js,
		streamName: streamName,
	}, nil
}

// NewClient returns a new nats client with default options
func NewClientWithDefaults(url string, streamName string) (Client, error) {
	ctx := context.Background()
	l := log.FromContext(ctx)

	natsClient, err := NewClient(url, streamName,
		func(o *nats.Options) error {
			o.AllowReconnect = true
			o.MaxReconnect = -1
			o.ReconnectBufSize = reconnectBufSize
			o.ReconnectWait = reconnectWaitTime
			o.RetryOnFailedConnect = true
			o.ClosedCB = func(_ *nats.Conn) {
				l.Info("nats client closed the connection")
			}
			o.DisconnectedErrCB = func(_ *nats.Conn, e error) {
				if e != nil {
					l.Warnf("nats client disconnected, err: %v", e)
				}
			}
			o.ReconnectedCB = func(_ *nats.Conn) {
				l.Warn("nats client reconnected")
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return natsClient, nil
}

type client struct {
	nats       *nats.Conn
	js         nats.JetStreamContext
	streamName string
}

func (c *client) StreamName() string {
	return c.streamName
}

// Close closes the connection to nats
func (c *client) Close() {
	c.nats.Close()
}

// IsConnected returns true if the client is connected to nats
func (c *client) IsConnected() bool {
	return c.nats.IsConnected()
}

type ConsumerConfig struct {
	AutoReplace bool

	// Filter expression for which topics this consumer covers.
	Filter string
	// MaxPending messages in the work queue.
	// NOTE: This sets an upper limit on the horizontal scalability of the
	// service.
	MaxPending int
	// MaxWaiting sets the maximum number of pull consumers (clients) to wait
	// for messages.
	MaxWaiting int
	// MaxDeliver sets the maximum amount of time the message will be
	// (re-) delivered.
	MaxDeliver int
	// AckWait sets the time to wait for message acknowledgement before
	// resending the message.
	AckWait time.Duration
}

func (cfg ConsumerConfig) Validate() error {
	if cfg.AckWait < time.Second {
		return fmt.Errorf(
			"invalid consumer configuration AckWait: %s < 1s",
			cfg.AckWait)
	}
	if cfg.MaxDeliver < 1 {
		return fmt.Errorf(
			"invalid consumer configuration MaxDeliver: %d < 1",
			cfg.MaxDeliver)
	}
	if cfg.MaxPending < 1 {
		return fmt.Errorf(
			"invalid consumer configuration MaxPending: %d < 1",
			cfg.MaxPending)
	}
	if cfg.MaxWaiting < 1 {
		return fmt.Errorf(
			"invalid consumer configuration MaxWaiting: %d < 1",
			cfg.MaxWaiting)
	}
	return nil
}

const consumerVersionString = "workflows/v2"

func (cfg ConsumerConfig) toNats(name string) *nats.ConsumerConfig {
	return &nats.ConsumerConfig{
		Description:   consumerVersionString,
		FilterSubject: cfg.Filter,
		Name:          name,
		Durable:       name,

		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       cfg.AckWait,
		DeliverPolicy: nats.DeliverAllPolicy,
		MaxAckPending: cfg.MaxPending,
		MaxDeliver:    cfg.MaxDeliver,
		MaxWaiting:    cfg.MaxWaiting,
	}
}

func (c *client) CreateConsumer(consumerName string, config ConsumerConfig) error {
	consumerInfo, err := c.js.ConsumerInfo(c.streamName, consumerName)
	if errors.Is(err, nats.ErrConsumerNotFound) {
		_, err := c.js.AddConsumer(c.streamName, config.toNats(consumerName))
		if errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
		} else {
			return ErrConsumerExist
		}
	} else if err != nil {
		return fmt.Errorf("nats: error getting consumer info: %w", err)
	}

	if consumerInfo.Config.Description == consumerVersionString {
		return nil
	}
	if config.AutoReplace {
		_, err = c.js.UpdateConsumer(c.streamName, config.toNats(consumerName))
		if err == nil {
			return nil
		}
		err = c.js.DeleteConsumer(c.streamName, consumerName)
		if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
			return err
		}
		_, err = c.js.AddConsumer(c.streamName, config.toNats(consumerName))
		if err != nil {
			err = fmt.Errorf("nats: failed to recreate consumer configuration: %w", err)
		}
	} else {
		err = ErrConsumerIncompatible
	}
	return err
}

// JetStreamCreateStream creates a stream
func (c *client) CreateStream() error {
	stream, err := c.js.StreamInfo(c.streamName)
	if err != nil && err != nats.ErrStreamNotFound {
		return err
	}
	if stream == nil {
		_, err = c.js.AddStream(&nats.StreamConfig{
			Name:                 c.streamName,
			Description:          "",
			Subjects:             []string{c.streamName + ".>"},
			Retention:            nats.WorkQueuePolicy,
			MaxConsumers:         0,
			MaxMsgs:              0,
			MaxBytes:             0,
			Discard:              nats.DiscardOld,
			DiscardNewPerSubject: false,
			MaxAge:               24 * time.Hour,
			MaxMsgsPerSubject:    0,
			MaxMsgSize:           0,
			Storage:              nats.FileStorage,
			Replicas:             1,
			NoAck:                false,
			Template:             "",
			Duplicates:           0,
			Placement:            nil,
			Mirror:               nil,
			Sources:              nil,
			Sealed:               false,
			DenyDelete:           false,
			DenyPurge:            false,
			AllowRollup:          false,
			RePublish:            nil,
			AllowDirect:          false,
			MirrorDirect:         false,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Subscribe to a work queue consumer.
func (c *client) Subscribe(
	ctx context.Context,
	consumerName string,
	q chan<- []byte,
) (Subscription, error) {
	info, err := c.js.ConsumerInfo(c.streamName, consumerName)
	if errors.Is(err, nats.ErrConsumerNotFound) {
		return nil, ErrNoConsumer
	} else if err != nil {
		return nil, err
	}
	sub, err := c.js.PullSubscribe(
		info.Config.FilterSubject, "",
		nats.Bind(c.streamName, consumerName),
		nats.Context(ctx),
	)
	if err != nil {
		return nil, err
	}

	return &subscription{
		dst: q,
		sub: sub,

		done: make(chan struct{}),
	}, nil
}

type Subscription interface {
	ListenAndServe() error
	Close() error
}

type subscription struct {
	dst chan<- []byte
	sub *nats.Subscription

	mu   sync.Mutex
	done chan struct{}
	err  error
}

func (s *subscription) ListenAndServe() (err error) {
	defer func() {
		close(s.dst)
		_ = s.sub.Unsubscribe()
	}()
	var (
		msgs []*nats.Msg
	)
	for err == nil {
		// TODO: If NATS server is >= 2.7.1, we can fetch up to MaxAckPending
		// documents to increase efficiency.
		msgs, err = s.sub.Fetch(1, nats.MaxWait(time.Minute))
		if errors.Is(err, nats.ErrTimeout) {
			err = nil
			time.Sleep(time.Millisecond * 20)
			continue
		} else if err != nil {
			break
		}
		for _, msg := range msgs {
			err = msg.Ack()
			if err != nil {
				break
			}
			select {
			case s.dst <- msg.Data:

			case <-s.done:
				return nil
			}
		}
	}
	return err
}

func (sub *subscription) Close() error {
	sub.mu.Lock()
	defer sub.mu.Unlock()
	select {
	case <-sub.done:
	default:
		close(sub.done)
	}
	return sub.err
}

// Publish a message to the given subject
func (c *client) Publish(subj string, data []byte) error {
	_, err := c.js.Publish(subj, data)
	return err
}
