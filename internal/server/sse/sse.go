package sse

// source: https://github.com/romanm-perun/gin-sse-example

// v2 of the great example of SSE in go by @ismasan.
// includes fixes:
//    * infinite loop ending in panic
//    * closing a client twice
//    * potentially blocked listen() from closing a connection during multiplex step.

import (
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// the amount of time to wait when pushing a message to
// a slow client or a client that closed after `range clients` started.
const patience time.Duration = time.Second * 1

type (
	NotificationEvent struct {
		EventName string
		Payload   interface{}
	}

	NotifierChan chan NotificationEvent

	Broker struct {

		// Events are pushed to this channel by the main events-gathering routine
		Notifier NotifierChan

		// New client connections
		newClients chan NotifierChan

		// Closed client connections
		closingClients chan NotifierChan

		// Client connections registry
		clients map[NotifierChan]struct{}
	}
)

func NewBroker() (broker *Broker) {
	// Instantiate a broker
	return &Broker{
		Notifier:       make(NotifierChan, 1),
		newClients:     make(chan NotifierChan),
		closingClients: make(chan NotifierChan),
		clients:        make(map[NotifierChan]struct{}),
	}
}

func (broker *Broker) ServeHTTP(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry
	messageChan := make(NotifierChan)

	// Signal the broker that we have a new connection
	broker.newClients <- messageChan

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		broker.closingClients <- messageChan
	}()

	c.Stream(func(w io.Writer) bool {
		// Emit Server Sent Events compatible
		event := <-messageChan

		c.SSEvent(event.EventName, event.Payload)

		// Flush the data immediately instead of buffering it for later.
		c.Writer.Flush()

		return true
	})
}

// Listen for new notifications and redistribute them to clients
func (broker *Broker) Listen() {
	for {
		select {
		case s := <-broker.newClients:

			// A new client has connected.
			// Register their message channel
			broker.clients[s] = struct{}{}
			log.Printf("Client added. %d registered clients", len(broker.clients))
		case s := <-broker.closingClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))
		case event := <-broker.Notifier:

			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan := range broker.clients {
				select {
				case clientMessageChan <- event:
				case <-time.After(patience):
					log.Print("Skipping client.")
				}
			}
		}
	}
}
