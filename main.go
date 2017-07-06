package main

import (
	"context"
	"encoding/json"
	"github.com/dividat/driver-go/senso"
	"log"
	"net/http"
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

	rootMsg, _ := json.Marshal(map[string]string{
		"message": "Dividat Driver",
		"version": version,
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		w.Write(rootMsg)
	})

	http.HandleFunc("/senso", sensoHandle.HTTPHandler)

	log.Panic(http.ListenAndServe(":8380", nil))
}
