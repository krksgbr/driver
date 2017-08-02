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

	log "github.com/sirupsen/logrus"
)

var version = "2.0.0"

const serverPort = "8382"

// Start the driver server
func Start() {

	log.WithField("version", version).Info("Dividat Driver starting")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sensoHandle := senso.New(ctx, log.WithField("package", "senso"))

	sensoHandle.Connect("127.0.0.1")

	httpServer(log.WithField("package", "server"), sensoHandle)

}

func httpServer(log *log.Entry, sensoHandle *senso.Handle) {

	// Load SSL keys
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

	rootMsg, _ := json.Marshal(map[string]string{
		"message": "Dividat Driver",
		"version": version,
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		w.Write(rootMsg)
	})

	http.Handle("/senso", sensoHandle)

	log.WithField("port", serverPort).Info("starting http server")
	log.Panic(http.ListenAndServeTLS(":"+serverPort, filepath.Join(sslDir, "cert.pem"), filepath.Join(sslDir, "key.pem"), nil))
}
