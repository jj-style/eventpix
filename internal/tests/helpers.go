package test_utils

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	natsclient "github.com/nats-io/nats.go"
)

// https://github.com/nats-io/nats.go/issues/467#issuecomment-1771424369
func NewInProcessNATSServer(t *testing.T) (conn *natsclient.Conn, cleanup func(), err error) {
	t.Helper()

	tmp, err := os.MkdirTemp("", "nats_test")
	if err != nil {
		err = fmt.Errorf("failed to create temp directory for NATS storage: %w", err)
		return
	}
	server, err := natsserver.NewServer(&natsserver.Options{
		DontListen: true, // Don't make a TCP socket.
		JetStream:  false,
		StoreDir:   tmp,
	})
	if err != nil {
		err = fmt.Errorf("failed to create NATS server: %w", err)
		return
	}
	// Add logs to stdout.
	// server.ConfigureLogger()
	server.Start()
	cleanup = func() {
		server.Shutdown()
		os.RemoveAll(tmp)
	}

	if !server.ReadyForConnections(time.Second * 5) {
		err = errors.New("failed to start server after 5 seconds")
		return
	}

	// Create a connection.
	conn, err = natsclient.Connect("", natsclient.InProcessServer(server))
	if err != nil {
		err = fmt.Errorf("failed to connect to server: %w", err)
		return
	}

	waitConnected(t, conn)

	return
}

func waitConnected(t *testing.T, c *nats.Conn) {
	t.Helper()

	timeout := time.Now().Add(3 * time.Second)
	for time.Now().Before(timeout) {
		if c.IsConnected() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("client connecting timeout")
}
