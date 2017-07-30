package main

//go:generate go-bindata -pkg $GOPACKAGE -o assets.go assets/...

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

func main() {

	log.Printf("Dividat Driver (%s), starting up...", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sensoHandle := senso.New(ctx)

	sensoHandle.Connect("localhost")

	startHTTPServer(sensoHandle)

}

func startHTTPServer(sensoHandle *senso.Handle) {

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
	sslDir := filepath.Join(tempDir, "assets", "ssl")

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
