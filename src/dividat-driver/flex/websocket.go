package flex

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WEBSOCKET PROTOCOL

// Implement net/http Handler interface
func (handle *Handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Set up logger
	var log = handle.log.WithFields(logrus.Fields{
		"clientAddress": r.RemoteAddr,
		"userAgent":     r.UserAgent(),
	})

	// Update to WebSocket
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("Could not upgrade connection to WebSocket.")
		http.Error(w, "WebSocket upgrade error", http.StatusBadRequest)
		return
	}

	log.Info("WebSocket connection opened")

	// Create a mutex for writing to WebSocket (connection supports only one concurrent reader and one concurrent writer (https://godoc.org/github.com/gorilla/websocket#hdr-Concurrency))
	writeMutex := sync.Mutex{}

	// Create a context for this WebSocket connection
	ctx, cancel := context.WithCancel(context.Background())

	// Send binary data up the WebSocket
	sendBinary := func(data []byte) error {
		writeMutex.Lock()
		conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
		err := conn.WriteMessage(websocket.BinaryMessage, data)
		writeMutex.Unlock()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.WithError(err).Error("WebSocket error")
			}
			return err
		}
		return nil
	}

	// Create channels with data received from SensingTex controller
	rx := handle.broker.Sub("rx")

	// send data from device
	go rx_data_loop(ctx, rx, sendBinary)

	// Helper function to close the connection
	close := func() {
		// Unsubscribe from broker
		handle.broker.Unsub(rx)

		// Stop serial connection
		handle.Disconnect()

		// Cancel the context
		cancel()

		// Close websocket connection
		conn.Close()

		log.Info("Websocket connection closed")
	}

	// Start connecting to devices
	handle.Connect()

	// Main loop for the WebSocket connection
	go func() {
		defer close()
		for {

			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.WithError(err).Error("WebSocket error")
				}
				return
			}

		}
	}()

}

// HELPERS

// rx_data_loop reads data from SensingTex and forwards it up the WebSocket
func rx_data_loop(ctx context.Context, rx chan interface{}, send func([]byte) error) {
	var err error
	for {
		select {
		case <-ctx.Done():
			return

		case i := <-rx:
			data, ok := i.([]byte)
			if ok {
				err = send(data)
			}
		}

		if err != nil {
			return
		}
	}
}

// Helper to upgrade http to WebSocket
var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
