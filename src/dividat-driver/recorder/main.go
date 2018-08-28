package main

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "localhost:8382", Path: "/senso"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		panic(err)
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
