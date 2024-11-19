package pubsub

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wnats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jj-style/eventpix/backend/internal/config"
	"github.com/nats-io/nats.go"
)

func NewPublisher(cfg *config.PubSub) (message.Publisher, func(), error) {
	wlogger := watermill.NopLogger{}
	switch cfg.Mode {
	case "memory":
		pubSub := gochannel.NewGoChannel(
			gochannel.Config{},
			wlogger,
		)
		return pubSub, func() { pubSub.Close() }, nil
	case "nats":
		natsCfg := cfg.Nats
		options, marshaler, jsConfig := natsConfig(natsCfg)

		publisher, err := wnats.NewPublisher(
			wnats.PublisherConfig{
				URL:         natsCfg.Url,
				NatsOptions: options,
				Marshaler:   marshaler,
				JetStream:   jsConfig,
			},
			wlogger,
		)
		if err != nil {
			return nil, func() {}, fmt.Errorf("creating publisher: %w", err)
		}
		return publisher, func() { publisher.Close() }, nil
	default:
		return nil, func() {}, fmt.Errorf("unsupported pubsub mode: %s", cfg.Mode)
	}
}

func NewSubscriber(cfg *config.PubSub) (message.Subscriber, func(), error) {
	wlogger := watermill.NopLogger{}
	switch cfg.Mode {
	case "nats":
		natsCfg := cfg.Nats
		options, marshaler, jsConfig := natsConfig(natsCfg)

		subscriber, err := wnats.NewSubscriber(
			wnats.SubscriberConfig{
				URL:              natsCfg.Url,
				CloseTimeout:     30 * time.Second,
				AckWaitTimeout:   30 * time.Second,
				NatsOptions:      options,
				Unmarshaler:      marshaler,
				JetStream:        jsConfig,
				QueueGroupPrefix: "eventpix_subscriber",
			},
			wlogger,
		)
		if err != nil {
			return nil, func() {}, fmt.Errorf("creating subscriber: %w", err)
		}
		return subscriber, func() { subscriber.Close() }, nil
	default:
		return nil, func() {}, fmt.Errorf("unsupported pubsub mode: %s", cfg.Mode)
	}

}

// get common watermill nats config for publisher/subscriber
func natsConfig(_ *config.Nats) ([]nats.Option, wnats.MarshalerUnmarshaler, wnats.JetStreamConfig) {
	options := []nats.Option{
		nats.RetryOnFailedConnect(true),
		nats.Timeout(30 * time.Second),
		nats.ReconnectWait(1 * time.Second),
	}
	jsConfig := wnats.JetStreamConfig{Disabled: true}
	marshaler := wnats.JSONMarshaler{}

	return options, marshaler, jsConfig
}
