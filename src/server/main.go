package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"logging"
	"senso"
	"update"
)

// build var (-ldflags)
var version string
var channel string

const serverPort = "8382"

// Start the driver server
func Start() {

	// Set up logging
	logrus.SetLevel(logrus.DebugLevel)
	logServer := logging.NewLogServer()
	logrus.AddHook(logServer)
	logrus.AddHook(logging.NewAMQPHook())
	http.Handle("/log", logServer)

	baseLog := logrus.WithFields(logrus.Fields{
		"version":        version,
		"releaseChannel": channel,
	})
	baseLog.Info("Dividat Driver starting")

	// Setup a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Senso
	sensoHandle := senso.New(ctx, baseLog.WithField("package", "senso"))
	http.Handle("/senso", sensoHandle)

	// Create a logger for server
	log := baseLog.WithField("package", "server")

	// Start the monitor
	go startMonitor(baseLog.WithField("package", "monitor"))

	// Setup driver update loop
	go update.Start(baseLog.WithField("package", "update"), version, channel)

	// Server root
	rootMsg, _ := json.Marshal(map[string]string{
		"message": "Dividat Driver",
		"channel": channel,
		"version": version,
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(rootMsg)
	})

	// Start the server
	log.WithField("port", serverPort).Info("Starting HTTP server.")
	log.Panic(http.ListenAndServe(":"+serverPort, nil))
}
