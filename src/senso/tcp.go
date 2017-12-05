package senso

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
)

// How long to wait before timeing out a tcp connection attempt
const dialTimeout = 5 * time.Second

// never stop retrying to connect
const maxElapsedTime = 0

// maximal interval to wait between connection retry
const maxInterval = 30 * time.Second

// connectTCP creates a persistent tcp connection to address
func connectTCP(ctx context.Context, baseLogger *logrus.Entry, address string, data chan []byte) {
	var dialer net.Dialer

	var log = baseLogger.WithField("address", address)

	var conn net.Conn
	dialTCP := func() error {

		dialer.Deadline = time.Now().Add(dialTimeout)
		var connErr error
		if conn != nil {
			conn.Close()
		}

		log.Info("Dialing TCP connection.")
		conn, connErr = dialer.DialContext(ctx, "tcp", address)

		if connErr, ok := connErr.(net.Error); ok && !connErr.Temporary() {
			return nil
		}

		if connErr != nil {
			log.WithError(connErr).Info("Could not connect with Senso.")
		}
		return connErr
	}

	var backOffStrategy = backoff.NewExponentialBackOff()

	// Never stop retrying
	backOffStrategy.MaxElapsedTime = maxElapsedTime

	// Set maximum interval to 30s
	backOffStrategy.MaxInterval = maxInterval

	for true {

		backOffStrategy.Reset()
		backoff.Retry(dialTCP, backOffStrategy)

		if conn != nil {

			log.Info("Connected.")

			// Close connection if we break or return
			defer conn.Close()

			// create channel for reading data and go read
			readChannel := make(chan []byte)
			go tcpReader(log, conn, readChannel)

			// create channel for writing data
			writeChannel := make(chan []byte)
			// We need an additional channel for handling write errors, unlike the readChannel we don't want to close the channel as somebody might try to write to it
			writeErrors := make(chan error)
			defer close(writeChannel)
			go tcpWriter(conn, writeChannel, writeErrors)

			// Inner loop for handling data
			disconnected := false
			for !disconnected {
				select {

				case <-ctx.Done():
					return

				case receivedData, more := <-readChannel:
					if more {
						// Attempt to send data, if can not send immediately discard
						select {
						case data <- receivedData:
						default:
						}
					} else {
						disconnected = true
					}

				case dataToWrite := <-data:
					writeChannel <- dataToWrite

				case writeError := <-writeErrors:
					if err, ok := writeError.(net.Error); ok && err.Timeout() {
						log.Debug("Timeout while writing.")
					} else {
						log.WithError(writeError).Error("Write error.")
						disconnected = true
					}
				}
			}
		}

		// Check if connection has been cancelled
		select {
		case <-ctx.Done():
			return

		default:
			log.Debug("Reconnecting.")
		}

	}
}

// Helper to read from TCP connection
func tcpReader(log *logrus.Entry, conn net.Conn, channel chan<- []byte) {

	defer close(channel)

	buffer := make([]byte, 1024)

	// Loop and read from connection.
	for {
		// conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		readN, readErr := conn.Read(buffer)

		if readErr != nil {
			if readErr == io.EOF {
				// connection is closed
				log.Info("Connection closed (EOF).")
				return
			} else if err, ok := readErr.(net.Error); ok && err.Timeout() {
				// Read timeout, just continue Nothing
			} else {
				log.WithError(readErr).Error("Read error.")
				return
			}
		} else {
			channel <- buffer[:readN]
		}
	}
}

// Helper to write to TCP connection. Note that this requires an additional channel to report errors
func tcpWriter(conn net.Conn, channel <-chan []byte, errorChannel chan<- error) {
	for {

		dataToWrite, more := <-channel

		if more {

			if conn != nil {
				conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
				_, err := conn.Write(dataToWrite)

				if err != nil {
					errorChannel <- err
				}

			} else {
				errorChannel <- errors.New("Can not write to TCP connection, because not connected.")
			}

		} else {
			return
		}

	}
}
