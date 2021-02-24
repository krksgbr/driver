package flex

import (
	"context"
	"sync"

	"encoding/binary"
	"bufio"
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
		handle.cancelCurrentConnection()
	}
}

// Keep looking for serial devices and connect to them when found, sending signals into the
// callback.
func listeningLoop(ctx context.Context, logger *logrus.Entry, onReceive func([]byte)) {
	for {
		scanAndConnectSerial(ctx, logger, onReceive)

		// Terminate if we were cancelled
		if ctx.Err() != nil {
			return
		}

		time.Sleep(2 * time.Second)
	}
}

// One pass of browsing for serial devices and trying to connect to them turn by turn, first
// successful connection wins.
//
// NOTE Portability of serial device detection has not been tested. This is a prototype
// implementation intended for Linux systems.
func scanAndConnectSerial(ctx context.Context, logger *logrus.Entry, onReceive func([]byte)) {
	deviceFileFolder := "/dev"

	files, err := ioutil.ReadDir(deviceFileFolder)
	if err != nil {
		logger.WithField("error", err).Info("Could not list serial devices.")
		return
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasPrefix(f.Name(), "ttyACM") {
			continue
		}

		connectSerial(ctx, logger, path.Join(deviceFileFolder, f.Name()), onReceive)

		// Terminate if we were cancelled
		if ctx.Err() != nil {
			return
		}
	}
}


// Serial communication

type ReaderState int

const (
	WAITING_FOR_HEADER ReaderState = iota
	HEADER_START
	HEADER_READ_LENGTH_MSB
	WAITING_FOR_BODY
	BODY_START
	BODY_POINT
	UNEXPECTED_BYTE
)

const (
	HEADER_START_MARKER = 'N'
	BODY_START_MARKER = 'P'
)

// Actually attempt to connect to an individual serial port and pipe its signal into the callback, summarizing
// package units into a buffer.
func connectSerial(ctx context.Context, logger *logrus.Entry, serialName string, onReceive func([]byte)) {
	config := &serial.Config{
		Name:        serialName,
		Baud:        921600,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
	}

	START_MEASUREMENT_CMD := []byte{'S', '\n'}

	logger.WithField("address", serialName).Info("Attempting to connect with serial port.")
	port, err := serial.OpenPort(config)
	if err != nil {
		logger.WithField("config", config).WithField("error", err).Info("Failed to open connection to serial port.")
		return
	}
	defer port.Close()

	_, err = port.Write(START_MEASUREMENT_CMD)
	if err != nil {
		logger.WithField("error", err).Info("Failed to write start message to serial port.")
		port.Close()
		return
	}

	reader := bufio.NewReader(port)
	state := WAITING_FOR_HEADER
	pointsLeftInSet := 0
	bytesLeftInPoint := 0

	var buff []byte
	for {
		// Terminate if we were cancelled
		if ctx.Err() != nil {
			logger.WithField("address", serialName).Info("Disconnecting from serial port.")
			return
		}

		input, err := reader.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}

		switch {
		case state == WAITING_FOR_HEADER && input == HEADER_START_MARKER:
			state = HEADER_START
		case state == HEADER_START && input == '\n':
			state = HEADER_READ_LENGTH_MSB
                case state == HEADER_READ_LENGTH_MSB:
			msb := input
			lsb, err := reader.ReadByte()
			if err == io.EOF {
				break
			} else if err != nil {
				continue
			}
			pointsLeftInSet = int(binary.BigEndian.Uint16([]byte{msb,lsb}))
			state = WAITING_FOR_BODY
		case state == WAITING_FOR_BODY && input == BODY_START_MARKER:
			state = BODY_START
		case state == BODY_START && input == '\n':
			state = BODY_POINT
			bytesLeftInPoint = 4
                case state == BODY_POINT:
			buff = append(buff, input)
			bytesLeftInPoint = bytesLeftInPoint - 1

			if bytesLeftInPoint <= 0 {
				pointsLeftInSet = pointsLeftInSet - 1

				if pointsLeftInSet <= 0 {
					// Finish and send set
					onReceive(buff)
					buff = []byte{}

					// Get ready for next set and request it
					state = WAITING_FOR_HEADER
					_, err = port.Write(START_MEASUREMENT_CMD)
					if err != nil {
						logger.WithField("error", err).Info("Failed to write poll message to serial port.")
						port.Close()
						return
					}
				} else {
					// Start next point
					bytesLeftInPoint = 4
				}
			}
		case state == UNEXPECTED_BYTE && input == HEADER_START_MARKER:
			// Recover from error state when a new header is seen
			buff = []byte{}
			bytesLeftInPoint = 0
			pointsLeftInSet = 0
			state = HEADER_START
		default:
			state = UNEXPECTED_BYTE
		}

	}

}
