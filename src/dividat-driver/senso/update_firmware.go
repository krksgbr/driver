package senso

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/dividat/driver/src/dividat-driver/firmware"
)

// Disconnect from current connection
func (handle *Handle) ProcessFirmwareUpdateRequest(command UpdateFirmware, msgChan chan<- string) {
	handle.log.Info("Processing firmware update request.")
	if handle.cancelCurrentConnection != nil {
		handle.cancelCurrentConnection()
	}
	image, err := decodeImage(command.Image)
	if err != nil {
		msg := fmt.Sprintf("Error decoding base64 string: %v", err)
		msgChan <- msg
		handle.log.Error(msg)
	}
	err = firmware.Update(context.Background(), image, nil, &command.Address, msgChan)
	if err != nil {
		msg := fmt.Sprintf("Failed to update firmware: %v", err)
		msgChan <- msg
		handle.log.Error(msg)
	}
}

func decodeImage(base64Str string) (io.Reader, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}
