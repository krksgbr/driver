package flex

/* Connects to Senso Flex devices through a serial connection and combines serial data into measurement sets.

This helps establish an indirect WebSocket connection to receive a stream of samples from the device.

The functionality of this module is as follows:

- While connected, scan for serial devices that look like a potential Flex device
- Connect to suitable serial devices and start polling for measurements
- Minimally parse incoming data to determine start and end of a measurement
- Send each complete measurement set to client as a binary package


NOTE At the moment this functionality is limited to Linux and macOS systems.

*/

import (
	"context"
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
	subscriberCount int

	log *logrus.Entry
}

// New returns an initialized handler
func New(ctx context.Context, log *logrus.Entry) *Handle {
	handle := Handle{
		broker: pubsub.New(32),
		ctx: ctx,
		log: log,
	}

	// Clean up
	go func() {
		<-ctx.Done()
		handle.broker.Shutdown()
	}()

	return &handle
}

// Connect to device
func (handle *Handle) Connect() {
	handle.subscriberCount++

	// If there is no existing connection, create it
	if handle.cancelCurrentConnection == nil {
		ctx, cancel := context.WithCancel(handle.ctx)

		onReceive := func(data []byte) {
			handle.broker.TryPub(data, "flex-rx")
		}

		go listeningLoop(ctx, handle.log, onReceive)

		handle.cancelCurrentConnection = cancel
	}
}

// Deregister subscribers and disconnect when none left
func (handle *Handle) DeregisterSubscriber() {
	handle.subscriberCount--

	if handle.subscriberCount == 0 && handle.cancelCurrentConnection != nil {
		handle.cancelCurrentConnection()
		handle.cancelCurrentConnection = nil
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
func scanAndConnectSerial(ctx context.Context, logger *logrus.Entry, onReceive func([]byte)) {
	deviceFileFolder := "/dev"

	files, err := ioutil.ReadDir(deviceFileFolder)
	if err != nil {
		logger.WithField("error", err).Info("Could not list serial devices.")
		return
	}

	for _, f := range files {
		// Terminate if we have been cancelled
		if ctx.Err() != nil {
			return
		}

		if f.IsDir() {
			continue
		}

		if !hasExpectedName(f.Name()) {
			continue
		}

		connectSerial(ctx, logger, path.Join(deviceFileFolder, f.Name()), onReceive)
	}
}

func hasExpectedName(filename string) bool {
	// Device shows up as a USB modem on Linux and macOS
	return strings.HasPrefix(filename, "ttyACM") || strings.HasPrefix(filename, "cu.usbmodem")
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
		Baud:        115200, // ignored by device
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

		// Finite State Machine for parsing byte stream
		switch {
		case state == WAITING_FOR_HEADER && input == HEADER_START_MARKER:
			state = HEADER_START
		case state == HEADER_START && input == '\n':
			state = HEADER_READ_LENGTH_MSB
                case state == HEADER_READ_LENGTH_MSB:
			// The number of measurements in each set may vary and is
			// given as two consecutive bytes (big-endian).
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

					// Get ready for next set and request it
					buff = []byte{}
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
