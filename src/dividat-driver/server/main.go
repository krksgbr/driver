package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/dividat/driver/src/dividat-driver/flex"
	"github.com/dividat/driver/src/dividat-driver/logging"
	"github.com/dividat/driver/src/dividat-driver/rfid"
	"github.com/dividat/driver/src/dividat-driver/senso"
)

// Uncomment following line for profiling. And run `go tool pprof http://localhost:8382/debug/pprof/profile` or `go tool pprof http://localhost:8382/debug/pprof/heap`
// import _ "net/http/pprof"

// build var (-ldflags)
var version string

const serverPort = "8382"

// Start the driver server
func Start(logger *logrus.Logger, origins []string) context.CancelFunc {
	// Log Server
	logServer := logging.NewLogServer()
	logger.AddHook(logServer)
	http.Handle("/log", corsHeaders(origins, logServer))

	baseLog := logger.WithFields(logrus.Fields{
		"version": version,
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
	http.Handle("/senso", corsHeaders(origins, sensoHandle))

	// Setup SensingTex reader
	flexHandle := flex.New(ctx, baseLog.WithField("package", "flex"))
	http.Handle("/flex", corsHeaders(origins, flexHandle))

	// Setup RFID scanner
	rfidHandle := rfid.NewHandle(ctx, baseLog.WithField("package", "rfid"))
	// net/http performs a redirect from `/rfid` if only `/rfid/` is mounted
	http.Handle("/rfid", corsHeaders(origins, rfidHandle))
	http.Handle("/rfid/", corsHeaders(origins, rfidHandle))

	// Create a logger for server
	log := baseLog.WithField("package", "server")

	// Start the monitor
	go startMonitor(baseLog.WithField("package", "monitor"))

	// Setup HTTP Server
	server := http.Server{Addr: "127.0.0.1:" + serverPort}

	// Server root
	rootMsg, _ := json.Marshal(map[string]string{
		"message":   "Dividat Driver",
		"version":   version,
		"machineId": systemInfo.MachineId,
		"os":        systemInfo.Os,
		"arch":      systemInfo.Arch,
	})
	http.Handle("/", corsHeaders(origins, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(rootMsg)
	})))

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

// Middleware for CORS headers, to be applied to any route that should be accessible from browser apps.
func corsHeaders(origins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.Header["Origin"]) == 1 && contains(origins, r.Header["Origin"][0]) {
			w.Header().Set("Access-Control-Allow-Origin", r.Header["Origin"][0])
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
		}

		// Announce that `Origin` header value may affect response
		w.Header().Set("Vary", "Origin")

		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func contains(slice []string, candidate string) bool {
	for _, member := range slice {
		if member == candidate {
			return true
		}
	}
	return false
}
