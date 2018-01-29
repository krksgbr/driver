package server

import (
	"runtime"

	"github.com/denisbrodbeck/machineid"
)

// Get system information

type SystemInfo struct {
	MachineId string

	Os   string
	Arch string
}

func GetSystemInfo() (*SystemInfo, error) {
	systemInfo := SystemInfo{}

	machineId, err := machineid.ProtectedID("dividat")
	if err != nil {
		return nil, err
	}

	systemInfo.Os = runtime.GOOS
	systemInfo.Arch = runtime.GOARCH

	systemInfo.MachineId = machineId

	return &systemInfo, nil
}
