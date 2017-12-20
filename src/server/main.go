package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
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
func Start(interactive bool) context.CancelFunc {

	// Set up logging
	logger := logrus.New()
	if !interactive {
		logger.Out = ioutil.Discard
	}
	logger.SetLevel(logrus.DebugLevel)

	// Log Server
	logServer := logging.NewLogServer()
	logger.AddHook(logServer)
	http.Handle("/log", logServer)

	// AMQP Log delivery
	logger.AddHook(logging.NewAMQPHook())

	baseLog := logger.WithFields(logrus.Fields{
		"version":        version,
		"releaseChannel": channel,
	})

	// Get System information
	systemInfo, err := GetSystemInfo()
	if err != nil {
		baseLog.WithError(err).Panic("Could not get system information.")
	}

	baseLog = baseLog.WithFields(logrus.Fields{
		"machineId": systemInfo.MachineId,
		"os":        systemInfo.Os,
		"arch":      systemInfo.Arch,
	})

	baseLog.Info("Dividat Driver starting")

	// Setup a context
	ctx, cancel := context.WithCancel(context.Background())

	// Setup Senso
	sensoHandle := senso.New(ctx, baseLog.WithField("package", "senso"))
	http.Handle("/senso", sensoHandle)

	// Create a logger for server
	log := baseLog.WithField("package", "server")

	// Start the monitor
	go startMonitor(baseLog.WithField("package", "monitor"))

	// Setup driver update loop
	go update.Start(baseLog.WithField("package", "update"), version, channel)

	// Setup HTTP Server
	server := http.Server{Addr: "127.0.0.1:" + serverPort}

	// Server root
	rootMsg, _ := json.Marshal(map[string]string{
		"message":   "Dividat Driver",
		"channel":   channel,
		"version":   version,
		"machineId": systemInfo.MachineId,
		"os":        systemInfo.Os,
		"arch":      systemInfo.Arch,
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(rootMsg)
	})

	// Start the server
	log.WithField("port", serverPort).Info("Starting HTTP server.")

	go func() {
		serverErr := server.ListenAndServe()
		if serverErr != http.ErrServerClosed {
			log.Panic(serverErr)
		}
	}()

	// cleanup routine
	go func() {
		<-ctx.Done()

		log.Info("Server closing down.")
		server.Close()

	}()

	return cancel
}
