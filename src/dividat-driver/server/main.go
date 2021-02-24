package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"dividat-driver/logging"
	"dividat-driver/rfid"
	"dividat-driver/senso"
	"dividat-driver/flex"
	"dividat-driver/update"
)

// Uncomment following line for profiling. And run `go tool pprof http://localhost:8382/debug/pprof/profile` or `go tool pprof http://localhost:8382/debug/pprof/heap`
// import _ "net/http/pprof"

// build var (-ldflags)
var version string
var channel string

const serverPort = "8382"

// Start the driver server
func Start(logger *logrus.Logger) context.CancelFunc {
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

	// Setup SensingTex reader
	flexHandle := flex.New(ctx, baseLog.WithField("package", "flex"))
	http.Handle("/flex", flexHandle)
	flexHandle.Connect()

	// Setup RFID scanner
	rfidHandle := rfid.NewHandle(ctx, baseLog.WithField("package", "rfid"))
	// net/http performs a redirect from `/rfid` if only `/rfid/` is mounted
	http.Handle("/rfid", rfidHandle)
	http.Handle("/rfid/", rfidHandle)

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
