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

// TODO: implement a backoff strategy
const dialTimeout = 1 * time.Second
const retryTimeout = 5 * time.Second

var exp = backoff.NewExponentialBackOff()

// connectTCP creates a persistent tcp connection to address
func connectTCP(ctx context.Context, baseLogger *logrus.Entry, address string, data chan []byte) {
	var dialer net.Dialer

	var log = baseLogger.WithField("address", address)

	// attempt to open a new connection
	dialer.Deadline = time.Now().Add(dialTimeout)

	var conn net.Conn
	dialTCP := func() error {
		var connErr error
		log.Info("dialing")
		conn, connErr = dialer.DialContext(ctx, "tcp", address)
		return connErr
	}

	connErr := backoff.Retry(dialTCP, backoff.NewExponentialBackOff())

	if connErr != nil {
		log.WithError(connErr).Info("dial failed")
	} else {

		log.Info("connected")

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
		for {
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
				}

			case dataToWrite := <-data:
				writeChannel <- dataToWrite

			case writeError := <-writeErrors:
				if err, ok := writeError.(net.Error); ok && err.Timeout() {
					log.Debug("timeout on write")
				} else {
					log.WithError(writeError).Error("write error")
				}
			}
		}

	}
	// Check if connection has been cancelled
	select {
	case <-ctx.Done():
		return

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
				log.Info("connection closed")
				return
			} else if err, ok := readErr.(net.Error); ok && err.Timeout() {
				// Read timeout, just continue Nothing
			} else {
				log.WithError(readErr).Error("read error")
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
				errorChannel <- errors.New("not connected, can not write to TCP connection")
			}

		} else {
			return
		}

	}
}
