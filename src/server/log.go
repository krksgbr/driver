package server

import (
	"container/list"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// LogServer implements logrus.Hook and http.Handler interfaces
type LogServer struct {
	channel chan *logrus.Entry
	entries *list.List

	listeners *list.List
}

// NewLogServer returns a new LogServer
func NewLogServer() *LogServer {
	logServer := LogServer{}

	logServer.channel = make(chan *logrus.Entry, 5)
	logServer.entries = list.New()

	logServer.listeners = list.New()

	// start a goroutine handling incoming log entries
	go func() {
		for entry := range logServer.channel {
			// add to log
			logServer.entries.PushFront(entry)

			for listenerElement := logServer.listeners.Front(); listenerElement != nil; listenerElement = listenerElement.Next() {
				listener, ok := listenerElement.Value.(chan *logrus.Entry)

				if !ok {
					continue
				}

				listener <- entry
			}
		}
	}()

	return &logServer
}

// Levels implements the logrus.Hook interface
func (logServer *LogServer) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire implements the logrus.Hook interface
func (logServer *LogServer) Fire(entry *logrus.Entry) error {
	// TODO: handle multiple receivers
	select {
	case logServer.channel <- entry:
	default:
		fmt.Println("ERROR[LogServer]: Could not handle log entry.")
		fmt.Println(entry.String())
	}
	return nil
}

var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var jsonFormatter = logrus.JSONFormatter{}

// Implement net/http Handler interface
func (logServer *LogServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Update to WebSocket
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).Error("websocket upgrade error")
		http.Error(w, "WebSocket upgrade error", http.StatusBadRequest)
		return
	}

	// Set up as listener so live log entries are received here and forwarded via WebSocket
	entries := make(chan *logrus.Entry)
	listenerElement := logServer.listeners.PushFront(entries)
	defer logServer.listeners.Remove(listenerElement)

	for entry := range entries {

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
