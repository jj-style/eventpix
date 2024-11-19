package dependencies

import (
	"github.com/jj-style/eventpix/backend/internal/config"
	"github.com/nats-io/nats.go"
)

func NatsProvider(cfg *config.Nats) (*nats.Conn, func(), error) {
	nc, err := nats.Connect(cfg.Url)
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() {
		nc.Drain()
	}
	return nc, cleanup, nil
}
