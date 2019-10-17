package flex

import (
	"context"
	"sync"

	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/cskr/pubsub"
	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

// Handle for managing SensingTex connection
type Handle struct {
	broker *pubsub.PubSub

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

// Connect to device
func (handle *Handle) Connect() {

	// Only allow one connection change at a time
	handle.connectionChangeMutex.Lock()
	defer handle.connectionChangeMutex.Unlock()

	// disconnect current connection first
	handle.Disconnect()

	// Create a child context for a new connection. This allows an individual connection (attempt) to be cancelled without restarting the whole handler
	ctx, cancel := context.WithCancel(handle.ctx)

	onReceive := func(data []byte) {
		handle.broker.TryPub(data, "rx")
	}

	go listeningLoop(ctx, handle.log, onReceive)

	handle.cancelCurrentConnection = cancel
}

// Disconnect from current connection
func (handle *Handle) Disconnect() {
	if handle.cancelCurrentConnection != nil {
		handle.log.Info("Disconnecting from serial port.")
		handle.cancelCurrentConnection()
	}
}

// Keep looking for serial devices and connect to them when found, sending signals into the
// callback.
func listeningLoop(ctx context.Context, baseLogger *logrus.Entry, onReceive func([]byte)) {
	for {
		findAndConnectSerial(ctx, baseLogger, onReceive)

		// TODO Check whether to die

		time.Sleep(2 * time.Second)
	}
}

// One pass of browsing for serial devices and trying to connect to them turn by turn, first
// successful connection wins.
//
// NOTE Portability of serial device detection has not been tested. This is a prototype
// implementation intended for Linux systems.
func findAndConnectSerial(ctx context.Context, baseLogger *logrus.Entry, onReceive func([]byte)) {
	deviceFileFolder := "/dev"

	files, err := ioutil.ReadDir(deviceFileFolder)
	if err != nil {
		baseLogger.WithField("error", err).Info("Could not list serial devices.")
		return
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasPrefix(f.Name(), "ttyACM") {
			continue
		}

		connectSerial(ctx, baseLogger, path.Join(deviceFileFolder, f.Name()), onReceive)

		// TODO check whether to die
	}
}

type ReaderState int

const (
	// Init
	WaitingForFirstHeader ReaderState = iota
	// Header
	HeaderStarted
	ExpectingHeaderEnd
	// Data row
	RowStarted
	WaitingForRowIndex
	ReadingRowData
	ReachedRowEnd
	// Error state
	UnexpectedByte
)

// Actually attempt to connect to an individual serial port and pipe its signal into the callback, summarizing
// package units into a buffer.
func connectSerial(ctx context.Context, baseLogger *logrus.Entry, serialName string, onReceive func([]byte)) {
	config := &serial.Config{
		Name:        serialName,
		Baud:        460800,
		ReadTimeout: 100 * time.Millisecond,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
	}

	baseLogger.WithField("address", serialName).Info("Attempting to connect with serial port.")

	port, err := serial.OpenPort(config)
	if err != nil {
		baseLogger.WithField("config", config).WithField("error", err).Info("Failed to open connection to serial port.")
		return
	}
	defer port.Close()

	_, err = port.Write([]byte{'S', '\n'})
	if err != nil {
		baseLogger.WithField("error", err).Info("Failed to write start message to serial port.")
		port.Close()
		return
	}

	reader := bufio.NewReader(port)
	state := WaitingForFirstHeader
	bytesLeftInRow := 0

	var buff []byte
	for {
		// TODO Check whether we have been killed before reading next byte

		input, err := reader.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}

		switch {
		case state == WaitingForFirstHeader && input == 0x48:
			state = HeaderStarted
		case state == ReachedRowEnd && input == 0x48:
			state = HeaderStarted
			fmt.Println()
			fmt.Printf("%x\n", buff)
			onReceive(buff)
			buff = []byte{}
		case state == UnexpectedByte && input == 0x48:
			// Recover from error state when a new header is seen
			buff = []byte{}
			bytesLeftInRow = 0
			state = HeaderStarted
		case state == HeaderStarted && input == 0x00:
			state = ExpectingHeaderEnd
		case state == ExpectingHeaderEnd && input == 0x0A:
			state = ReachedRowEnd
		case state == ReadingRowData && bytesLeftInRow > 0:
			bytesLeftInRow = bytesLeftInRow - 1
			buff = append(buff, input)
		case state == ReadingRowData && bytesLeftInRow == 0 && input == 0x0A:
			state = ReachedRowEnd
			buff = append(buff, input)
		case state == ReachedRowEnd && input == 0x4D:
			state = RowStarted
			buff = append(buff, input)
		case state == RowStarted:
			state = WaitingForRowIndex
			// 2 bytes per sample
			bytesLeftInRow = int(input) * 2
			buff = append(buff, input)
		case state == WaitingForRowIndex:
			state = ReadingRowData
			buff = append(buff, input)
		case state == ReadingRowData:
			buff = append(buff, input)
		default:
			state = UnexpectedByte
		}
	}

}
