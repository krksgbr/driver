package server

//go:generate go-bindata -pkg $GOPACKAGE -o ssl.go ssl/...

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

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
	sensoHandle.Connect("127.0.0.1")
	http.Handle("/senso", sensoHandle)

	// Create a logger for server
	log := logrus.WithField("package", "server")

	// Unpack ssl keys
	tempDir, tempDirErr := ioutil.TempDir("", "dividat-driver")
	if tempDirErr != nil {
		log.Panic("could not create temp directory")
	}
	defer os.RemoveAll(tempDir) // clean up
	restoreAssetsErr := RestoreAssets(tempDir, "")
	if restoreAssetsErr != nil {
		log.Panic("could not restore ssl keys")
	}
	sslDir := filepath.Join(tempDir, "ssl")

	// Server root
	rootMsg, _ := json.Marshal(map[string]string{
		"message": "Dividat Driver",
		"version": version,
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		w.Write(rootMsg)
	})

	// Start the server
	log.WithField("port", serverPort).Info("starting http server")
	log.Panic(http.ListenAndServeTLS(":"+serverPort, filepath.Join(sslDir, "cert.pem"), filepath.Join(sslDir, "key.pem"), nil))
}
