package server

//go:generate go-bindata -pkg $GOPACKAGE -o ssl.go ssl/...

import (
	"context"
	"encoding/json"
	"github.com/dividat/driver-go/senso"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var version = "2.0.0"

// Start the driver server
func Start() {

	log.Printf("Dividat Driver (%s)", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sensoHandle := senso.New(ctx)

	sensoHandle.Connect("127.0.0.1")

	httpServer(sensoHandle)

}

func httpServer(sensoHandle *senso.Handle) {

	// Load SSL keys
	tempDir, tempDirErr := ioutil.TempDir("", "dividat-driver")
	if tempDirErr != nil {
		log.Panic("Could not create temp directory.")
	}
	defer os.RemoveAll(tempDir) // clean up

	restoreAssetsErr := RestoreAssets(tempDir, "")
	if restoreAssetsErr != nil {
		log.Panic("Could not restore assets.")
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

	log.Panic(http.ListenAndServeTLS(":8380", filepath.Join(sslDir, "cert.pem"), filepath.Join(sslDir, "key.pem"), nil))
}
