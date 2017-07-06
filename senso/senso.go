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
	Data           chan []byte
	dataConnection net.Conn

	Control           chan []byte
	ControlWrite      chan []byte
	controlConnection net.Conn

	ctx context.Context
}

// New returns an initialized Senso handler
func New(ctx context.Context) *Handle {
	handle := Handle{}

	handle.ctx = ctx

	// Make channels
	handle.Data = make(chan []byte, 2)
	handle.Control = make(chan []byte, 2)
	handle.ControlWrite = make(chan []byte, 2)

	// spawn a goroutine that listens to the ControlWrite channel and writes to the current connection
	go handle.writeControl()

	return &handle
}

func (handle *Handle) writeControl() {
	for b := range handle.ControlWrite {

		if handle.controlConnection != nil {
			handle.controlConnection.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))

			_, err := handle.controlConnection.Write(b)
			if err != nil {
				handle.onError(err)
			}
		} else {
			handle.onError(errors.New("not connected, can not write to Control"))
		}
	}

}

// How to deal with errors
func (handle *Handle) onError(err error) {
	log.Println(err)
}

// Connect with a Senso
func (handle *Handle) Connect(address string) context.CancelFunc {

	// Create a child context for a new connection. This allows an individual connection (attempt) to be cancelled without restarting the whole Senso handler
	connCtx, cancel := context.WithCancel(handle.ctx)

	dataConnectionCh := make(chan net.Conn)
	go connectAndRead(connCtx, address+":55568", dataConnectionCh, handle.Data)

	controlConnectionCh := make(chan net.Conn)
	go connectAndRead(connCtx, address+":55567", controlConnectionCh, handle.Control)

	go func() {
		for {
			select {
			case newDataConnection := <-dataConnectionCh:
				// set the new connection in the Connection struct
				if newDataConnection != nil {
					log.Println("Data connected!")
				}
				handle.dataConnection = newDataConnection
			case newControlConnection := <-controlConnectionCh:
				// set the new handleection in the Connection struct
				if newControlConnection != nil {
					log.Println("Control connected!")
				}
				handle.controlConnection = newControlConnection
			case <-connCtx.Done():
				break
			}
		}
	}()

	return cancel

}

// connectAndRead retries to connect to a given address and reads data.
// The process can be controlled via context (i.e. cancelled)
// The current connection is sent down the connection channel.
// Incoming data is sent down the data channel.
func connectAndRead(ctx context.Context, address string, connection chan net.Conn, data chan []byte) {

	var dialer net.Dialer

	// allocate a buffer to read from the TCP connection
	buffer := make([]byte, 1024)

	for {

		// attempt to open a new connection
		dialer.Deadline = time.Now().Add(1 * time.Second)
		conn, connErr := dialer.DialContext(ctx, "tcp", address)
		connection <- conn

		if connErr != nil {
			log.Println(connErr.Error())
		} else {
			// clean up connection. This also causes connection to be closed if the connection contxt is cancelled.
			defer conn.Close()

			for {
				conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				readN, readErr := conn.Read(buffer)

				if readErr != nil {
					if readErr == io.EOF {
						// connection is closed
						break
					} else if err, ok := readErr.(net.Error); ok && err.Timeout() {
						// Nothing
					} else {
						log.Println(readErr.Error())
					}
				} else {
					// attempt to send data down the channel, if not possible the data is discarded
					select {
					case data <- buffer[:readN]:
					default:
					}
				}

				// Check if reading has been cancelled
				select {
				case <-ctx.Done():
					return
				default:
				}

			}
		}

		// Check if reading has been cancelled
		select {
		case <-ctx.Done():
			return
		default:
			// Sleep 5s before attempting to reconnect
			time.Sleep(5 * time.Second)
		}
	}
}

// HTTPHandler handles a HTTP/WebSocket requests
func (handle *Handle) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		handle.onError(err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
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
					handle.onError(err)
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
					handle.onError(err)
				}
				break
			} else {
				if messageType == websocket.BinaryMessage {
					handle.ControlWrite <- b
				}
			}

		}

	}()

}
