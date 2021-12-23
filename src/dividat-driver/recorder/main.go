package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := parseUrl()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Could not record from '%s': %s", u.String(), err)
	}
	defer c.Close()

	prev := time.Now()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			encoded := base64.StdEncoding.EncodeToString(message)
			now := time.Now()
			d := now.Sub(prev)
			prev = now
			fmt.Println(fmt.Sprintf("%d, ", d.Nanoseconds()/1000000) + encoded)
			if err != nil {
				panic(err)
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			select {
			case <-done:
			}
			return
		}
	}
}

func parseUrl() url.URL {
	if (len(os.Args) < 2) {
		log.Fatal("Expected the WebSocket URL to record from as a parameter")
	}
	u, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatalf("Malformed WebSocket URL: %s", err)
	}
	return *u
}
