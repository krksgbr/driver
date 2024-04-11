package senso

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/dividat/driver/src/dividat-driver/firmware"
)

type SendMsg struct {
	progress func(string)
	failure  func(string)
	success  func(string)
}


// Disconnect from current connection
func (handle *Handle) ProcessFirmwareUpdateRequest(command UpdateFirmware, send SendMsg) {
	handle.log.Info("Processing firmware update request.")
	handle.firmwareUpdate.SetUpdating(true)

	if handle.cancelCurrentConnection != nil {
		send.progress("Disconnecting from the Senso")
		handle.cancelCurrentConnection()
	}

	image, err := decodeImage(command.Image)
	if err != nil {
		msg := fmt.Sprintf("Error decoding base64 string: %v", err)
		send.failure(msg)
		handle.log.Error(msg)
	}

	err = firmware.UpdateBySerial(context.Background(), command.SerialNumber, image, send.progress)
	if err != nil {
		failureMsg := fmt.Sprintf("Failed to update firmware: %v", err)
		send.failure(failureMsg)
		handle.log.Error(failureMsg)
	} else {
		send.success("Firmware successfully transmitted.")
	}
	handle.firmwareUpdate.SetUpdating(false)
}

func decodeImage(base64Str string) (io.Reader, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}
