package senso

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// Handle for managing Senso
type Handle struct {
	Data    chan []byte
	Control chan []byte
	ctx     context.Context
}

// New returns an initialized Senso handler
func New(ctx context.Context) *Handle {
	handle := Handle{}

	handle.ctx = ctx

	// Make channels
	handle.Data = make(chan []byte)
	handle.Control = make(chan []byte)

	return &handle
}

// Connect to a Senso, will create TCP connections to control and data ports
func (handle *Handle) Connect(address string) context.CancelFunc {
	// Create a child context for a new connection. This allows an individual connection (attempt) to be cancelled without restarting the whole Senso handler
	ctx, cancel := context.WithCancel(handle.ctx)

	go connectTCP(ctx, address+":55568", handle.Data)
	go connectTCP(ctx, address+":55567", handle.Control)

	return cancel
}

func connectTCP(ctx context.Context, address string, data chan []byte) {
	var dialer net.Dialer

	// loop to retry connection
	for {
		// attempt to open a new connection
		dialer.Deadline = time.Now().Add(1 * time.Second)
		log.Println("dialing", address, "...")
		conn, connErr := dialer.DialContext(ctx, "tcp", address)

		if connErr != nil {
			log.Println(connErr.Error())
		} else {

			log.Println("connected to", address)

			// Close connection if we break or return
			defer conn.Close()

			// create channel for reading data and go read
			readChannel := make(chan []byte)
			go tcpReader(conn, readChannel)

			// create channel for writing data
			writeChannel := make(chan []byte)
			writeErrors := make(chan error)
			defer close(writeChannel)
			go tcpWriter(conn, writeChannel, writeErrors)

			// Inner loop for handling data
		DataLoop:
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
					} else {
						close(writeChannel)
						break DataLoop
					}

				case dataToWrite := <-data:
					writeChannel <- dataToWrite

				case writeError := <-writeErrors:
					if err, ok := writeError.(net.Error); ok && err.Timeout() {
						onError(err)
					} else {
						close(writeChannel)
						break DataLoop
					}
				}
			}

		}
		// Check if connection has been cancelled
		select {
		case <-ctx.Done():
			return
		default:
			// Sleep 5s before reattempting to reconnect
			time.Sleep(5 * time.Second)
		}

	}
}

// Helper to read from TCP connection
func tcpReader(conn net.Conn, channel chan<- []byte) {

	defer close(channel)

	buffer := make([]byte, 1024)

	// Loop and read from connection.
	for {
		// conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		readN, readErr := conn.Read(buffer)

		if readErr != nil {
			if readErr == io.EOF {
				// connection is closed
				return
			} else if err, ok := readErr.(net.Error); ok && err.Timeout() {
				// Read timeout, just continue Nothing
			} else {
				log.Println(readErr.Error())
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

// How to deal with errors
func onError(err error) {
	log.Println(err)
}

// Implement net/http Handler interface
func (handle *Handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		onError(err)
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		return
	}

	// send data
	go func() {
		for data := range handle.Data {
			// fmt.Println(data)
			conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
			err := conn.WriteMessage(websocket.BinaryMessage, data)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					onError(err)
				}
				break
			}
		}
	}()

	// receive messages
	go func() {
		for {
			messageType, b, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					onError(err)
				}
				break
			} else {
				if messageType == websocket.BinaryMessage {
					handle.Control <- b
				}
			}

		}

	}()

}
