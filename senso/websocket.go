package senso

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WEBSOCKET PROTOCOL

// Command sent by Play
type Command struct {
	*GetSensoConnection
	*SensoConnect
}

// GetSensoConnection command
type GetSensoConnection struct{}

// SensoConnect command
type SensoConnect struct {
	SensoConnection SensoConnection `json:"connection"`
}

// UnmarshalJSON implements encoding/json Unmarshaler interface
func (command *Command) UnmarshalJSON(data []byte) error {
	temp := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if temp.Type == "GetSensoConnection" {
		command.GetSensoConnection = &GetSensoConnection{}
	} else if temp.Type == "SensoConnect" {
		err := json.Unmarshal(data, &command.SensoConnect)
		if err != nil {
			return err
		}
	} else {
		return errors.New("can not decode unknown command")
	}
	return nil
}

// Message that can be sent to Play
type Message struct {
	*SensoConnection
}

// SensoConnection describes where we are connected to
type SensoConnection struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// MarshalJSON implementes encoding/json Marshaler interface for Message
func (message *Message) MarshalJSON() ([]byte, error) {
	if message.SensoConnection != nil {
		return json.Marshal(&struct {
			Type            string           `json:"type"`
			SensoConnection *SensoConnection `json:"connection"`
		}{
			Type:            "SensoConnection",
			SensoConnection: message.SensoConnection,
		})

	}

	return nil, errors.New("could not marshal message")

}

// Implement net/http Handler interface
func (handle *Handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var log = handle.log.WithFields(logrus.Fields{
		"clientAddress": r.RemoteAddr,
		"userAgent":     r.UserAgent(),
	})

	// Update to WebSocket
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("websocket upgrade error")
		http.Error(w, "WebSocket upgrade error", http.StatusBadRequest)
		return
	}

	log.Info("WebSocket connection opened")

	// send data
	go func() {
		for data := range handle.Data {
			// fmt.Println(data)
			conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
			defer conn.Close()
			err := conn.WriteMessage(websocket.BinaryMessage, data)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.WithError(err).Error("WebSocket error")
				}
				return
			}
		}
	}()

	// receive messages
	go func() {
		defer conn.Close()
		for {

			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.WithError(err).Error("WebSocket error")
				}
				return
			}

			if messageType == websocket.BinaryMessage {
				log.WithField("data", msg).Debug("forwarding data to control port")
				handle.Control <- msg
			} else if messageType == websocket.TextMessage {

				var command Command
				decodeErr := json.Unmarshal(msg, &command)
				if decodeErr != nil {
					log.WithField("rawCommand", msg).WithError(decodeErr).Error("can not decode command")
					continue
				}

				// TODO: log the entire command nicer
				// log.WithField("command", string(msg[:])).Debug("received command")
				log.WithField("command", command).Debug("received command")

				if command.GetSensoConnection != nil {
					var message Message

					message.SensoConnection = &SensoConnection{Type: "IP", Address: handle.RemoteAddress}

					// encoded, encodeErr := message.MarshallJSON()
					encoded, encodeErr := json.Marshal(&message)
					if encodeErr != nil {
						log.WithError(encodeErr).Error("could not encode message")
						continue
					}

					writeErr := conn.WriteMessage(websocket.TextMessage, encoded)
					// writeErr := conn.WriteJSON(&message)
					if writeErr != nil {
						log.WithError(writeErr).Error("could not send message to websocket client")
						continue
					}

				} else if command.SensoConnect != nil {
					connectionType := command.SensoConnect.SensoConnection.Type
					if connectionType == "IP" {
						handle.Connect(command.SensoConnect.SensoConnection.Address)
					} else {
						// TODO: think about if this should cause a warning
						log.Warn("do not know how to handle connection type " + connectionType)
					}
				}

			}

		}
	}()

}

// HELPERS

// Helper to upgrade http to WebSocket
var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
