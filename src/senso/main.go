package senso

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Handle for managing Senso
type Handle struct {
	Data    chan []byte
	Control chan []byte

	Address *string

	ctx context.Context

	cancelCurrentConnection context.CancelFunc

	log *logrus.Entry
}

// New returns an initialized Senso handler
func New(ctx context.Context, log *logrus.Entry) *Handle {
	handle := Handle{}

	handle.ctx = ctx

	handle.log = log

	// Channels for data and control
	handle.Data = make(chan []byte)
	handle.Control = make(chan []byte)

	return &handle
}

// Connect to a Senso, will create TCP connections to control and data ports
func (handle *Handle) Connect(address string) {

	// disconnect current connection first
	handle.Disconnect()

	// set address in handle
	handle.Address = &address

	// Create a child context for a new connection. This allows an individual connection (attempt) to be cancelled without restarting the whole Senso handler
	ctx, cancel := context.WithCancel(handle.ctx)

	handle.log.WithField("address", address).Info("Attempting to connect with Senso.")

	go connectTCP(ctx, handle.log.WithField("channel", "data"), address+":55568", handle.Data)
	time.Sleep(20 * time.Millisecond)
	go connectTCP(ctx, handle.log.WithField("channel", "control"), address+":55567", handle.Control)

	handle.cancelCurrentConnection = cancel
}

// Disconnect from current connection
func (handle *Handle) Disconnect() {
	if handle.cancelCurrentConnection != nil {
		handle.log.Info("Disconnecting from Senso.")
		handle.cancelCurrentConnection()
		handle.Address = nil
	}
}
