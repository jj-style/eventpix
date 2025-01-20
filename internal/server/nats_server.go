package server

import natsServer "github.com/nats-io/nats-server/v2/server"

func NewNatsServer() (*natsServer.Server, error) {
	opts := &natsServer.Options{}
	return natsServer.NewServer(opts)
}
