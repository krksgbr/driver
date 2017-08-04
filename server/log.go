package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// LogServer implements logrus.Hook and http.Handler interfaces
type LogServer struct {
	channel chan *log.Entry
}

// NewLogServer returns a new LogServer
func NewLogServer() *LogServer {
	logServer := LogServer{}

	logServer.channel = make(chan *log.Entry, 20)

	return &logServer
}

// Levels implements the logrus.Hook interface
func (logServer *LogServer) Levels() []log.Level {
	return log.AllLevels
}

// Fire implements the logrus.Hook interface
func (logServer *LogServer) Fire(entry *log.Entry) error {
	// TODO: handle multiple receivers
	select {
	case logServer.channel <- entry:
	default:
		fmt.Println("Could not send logentry")
	}
	return nil
}

// Implement net/http Handler interface
var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var jsonFormatter = log.JSONFormatter{}

func (logServer *LogServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Update to WebSocket
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("websocket upgrade error")
		http.Error(w, "WebSocket upgrade error", http.StatusBadRequest)
		return
	}

	for entry := range logServer.channel {

		encoded, encodeErr := jsonFormatter.Format(entry)
		if encodeErr != nil {
			continue
		}

		writeErr := conn.WriteMessage(websocket.TextMessage, encoded)
		if writeErr != nil {
			return
		}
	}
}
