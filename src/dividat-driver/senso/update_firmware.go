package senso

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/dividat/driver/src/dividat-driver/firmware"
)

type OnUpdate func(msg FirmwareUpdateMessage)

func (handle *Handle) isUpdatingFirmware() bool {
	handle.firmwareUpdateMutex.Lock()
	state := handle.firmwareUpdateInProgress
	handle.firmwareUpdateMutex.Unlock()
	return state
}

func (handle *Handle) setUpdatingFirmware(state bool) {
	handle.firmwareUpdateMutex.Lock()
	handle.firmwareUpdateInProgress = state
	handle.firmwareUpdateMutex.Unlock()
}

// Disconnect from current connection
func (handle *Handle) ProcessFirmwareUpdateRequest(command UpdateFirmware, onUpdate OnUpdate) {
	handle.log.Info("Processing firmware update request.")
	handle.setUpdatingFirmware(true)

	onProgress := func(progressMsg string) {
		onUpdate(FirmwareUpdateMessage{FirmwareUpdateProgress: &progressMsg})
	}

	if handle.cancelCurrentConnection != nil {
		onProgress("Disconnecting from the Senso")
		handle.cancelCurrentConnection()
	}

	image, err := decodeImage(command.Image)
	if err != nil {
		msg := fmt.Sprintf("Error decoding base64 string: %v", err)
		onUpdate(FirmwareUpdateMessage{FirmwareUpdateFailure: &msg})
		handle.log.Error(msg)
	}

	onProgress("Waiting 10 seconds for connection teardown")
	time.Sleep(10 * time.Second)
	err = firmware.UpdateBySerial(context.Background(), command.SerialNumber, image, onProgress)
	if err != nil {
		failureMsg := fmt.Sprintf("Failed to update firmware: %v", err)
		onUpdate(FirmwareUpdateMessage{FirmwareUpdateFailure: &failureMsg})
		handle.log.Error(failureMsg)
	} else {
		successMsg := "Firmware successfully transmitted."
		onUpdate(FirmwareUpdateMessage{FirmwareUpdateSuccess: &successMsg})
	}
	handle.setUpdatingFirmware(false)
}

func decodeImage(base64Str string) (io.Reader, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}
