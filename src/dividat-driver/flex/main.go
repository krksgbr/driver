package flex

import (
	"context"
	"sync"

	"fmt"
	"bufio"
	"io"
	"time"

	"github.com/cskr/pubsub"
	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

// Handle for managing SensingTex connection
type Handle struct {
	broker *pubsub.PubSub

	Address *string

	ctx context.Context

	cancelCurrentConnection context.CancelFunc
	connectionChangeMutex   *sync.Mutex

	log *logrus.Entry
}

// New returns an initialized handler
func New(ctx context.Context, log *logrus.Entry) *Handle {
	handle := Handle{}

	handle.ctx = ctx

	handle.log = log

	handle.connectionChangeMutex = &sync.Mutex{}

	// PubSub broker
	handle.broker = pubsub.New(32)

	// Clean up
	go func() {
		<-ctx.Done()
		handle.broker.Shutdown()
	}()

	return &handle
}

// Connect to a Senso, will create TCP connections to control and data ports
func (handle *Handle) Connect(address string) {

	// Only allow one connection change at a time
	handle.connectionChangeMutex.Lock()
	defer handle.connectionChangeMutex.Unlock()

	// disconnect current connection first
	handle.Disconnect()

	// set address in handle
	handle.Address = &address

	// Create a child context for a new connection. This allows an individual connection (attempt) to be cancelled without restarting the whole handler
	ctx, cancel := context.WithCancel(handle.ctx)

	handle.log.WithField("address", address).Info("Attempting to connect with serial port.")

	onReceive := func(data []byte) {
		handle.broker.TryPub(data, "rx")
	}

	go connectSerial(ctx, handle.log, address, onReceive)

	handle.cancelCurrentConnection = cancel
}

// Disconnect from current connection
func (handle *Handle) Disconnect() {
	if handle.cancelCurrentConnection != nil {
		handle.log.Info("Disconnecting from serial port.")
		handle.cancelCurrentConnection()
		handle.Address = nil
	}
}

type ReaderState int

const (
	WaitingForFirstHeader ReaderState = iota
	HeaderStarted
	ExpectingHeaderEnd
	ReadingRowData
	ReachedRowEnd
	UnexpectedByte
)

func connectSerial(ctx context.Context, baseLogger *logrus.Entry, address string, onReceive func([]byte)) {
	config := &serial.Config{
		Name: "/dev/ttyAMA0",
		Baud: 115200,
		ReadTimeout: 100 * time.Millisecond,
		Size: 8,
		Parity: serial.ParityNone,
		StopBits: serial.Stop1,
	}
	fmt.Println(config)

	serialHandle, err := serial.OpenPort(config)
        if err != nil {
                // TODO
                panic(err)
        }

	_, err = serialHandle.Write([]byte{'S', '\n'})
	if err != nil {
		panic(err)
	}

        reader := bufio.NewReader(serialHandle)
	state := WaitingForFirstHeader

        var buff []byte
        for {
                input, err := reader.ReadByte()
                // TODO Handle other errors
                if err != nil && err == io.EOF {
                        break
                }

                if state == WaitingForFirstHeader && input == 0x48 {
			state = HeaderStarted
                } else if state == ReachedRowEnd && input == 0x48 {
			state = HeaderStarted
			fmt.Println()
			fmt.Printf("%x\n", buff)
			onReceive(buff)
                        buff = []byte{}
		} else if state == HeaderStarted && input == 0x00 {
			state = ExpectingHeaderEnd
		} else if state == ExpectingHeaderEnd && input == 0x0A {
			state = ReachedRowEnd
		} else if state == ReadingRowData && input == 0x0A {
			state = ReachedRowEnd
		} else if state == ReachedRowEnd && input == 0x4D {
			state = ReadingRowData
			buff = append(buff, input)
		} else if state == ReadingRowData {
			buff = append(buff, input)
		}
        }
}
