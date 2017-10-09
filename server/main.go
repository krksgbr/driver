package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dividat/driver-go/senso"

	"github.com/sirupsen/logrus"
)

var version = "2.0.0"

const serverPort = "8382"

// Start the driver server
func Start() {

	// Set up logging
	logrus.SetLevel(logrus.DebugLevel)
	logServer := NewLogServer()
	logrus.AddHook(logServer)
	http.Handle("/log", logServer)

	logrus.WithField("version", version).Info("Dividat Driver starting")

	// Setup a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Senso
	sensoHandle := senso.New(ctx, logrus.WithField("package", "senso"))
	http.Handle("/senso", sensoHandle)

	// Create a logger for server
	log := logrus.WithField("package", "server")

	// Start the monitor
	go startMonitor(logrus.WithField("package", "monitor"))

	// Server root
	rootMsg, _ := json.Marshal(map[string]string{
		"message": "Dividat Driver",
		"version": version,
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(rootMsg)
	})

	// Start the server
	log.WithField("port", serverPort).Info("starting http server")
	log.Panic(http.ListenAndServe(":"+serverPort, nil))
}
