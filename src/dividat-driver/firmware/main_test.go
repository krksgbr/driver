package firmware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

type TestCase struct {
	description string
	setup       *TestSetup
	expectedLog []string
	skip        bool
}

func Test(t *testing.T) {

	testCases := []TestCase{
		{
			description: "Discovers Senso in normal mode, then rediscovers it in dfu mode.",
			skip:        false,
			setup: &TestSetup{
				Image:              bytes.NewBufferString("mock image data"),
				Address:            "",
				Serial:             "",
				DiscoverFunc:       succeedDiscoveryFor("any"),
				SendDfuCommandFunc: succeedDfuSend,
				PutTFTPFunc:        succeedPutTFTP,
			},
			// No address is provided, so it should call discover.
			// The discovery is successful, so it should send DFU command.
			// The DFU command is successful, so it find the Senso in DFU mode.
			// The discovery is successful again, so it should transfer the image.
			expectedLog: []string{
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", normalService),
				fmt.Sprintf("SendDfuCommand | host: 127.0.0.1 (via: %s), port: 55567", normalService),
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", dfuService),
				"Sleep | duration: 5s",
				fmt.Sprintf("PutTFTP | host: 127.0.0.1 (via: %s), port: 69, image: mock image data", dfuService),
			},
		},
		{
			description: "Does not discover Senso in normal mode, but discovers it in DFU mode",
			skip:        false,
			setup: &TestSetup{
				Image:              bytes.NewBufferString("mock image data"),
				Address:            "",
				Serial:             "",
				DiscoverFunc:       succeedDiscoveryFor(dfuService),
				SendDfuCommandFunc: succeedDfuSend,
				PutTFTPFunc:        succeedPutTFTP,
			},
			// It should not discover the Senso in normal mode.
			// It should discover the Senso in dfu mode and transfer the image.
			expectedLog: []string{
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", normalService),
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", dfuService),
				"Sleep | duration: 5s",
				fmt.Sprintf("PutTFTP | host: 127.0.0.1 (via: %s), port: 69, image: mock image data", dfuService),
			},
		},
		{
			description: "Does not discover Senso in either normal of DFU mode",
			skip:        false,
			setup: &TestSetup{
				Image:              bytes.NewBufferString("mock image data"),
				Address:            "",
				Serial:             "",
				DiscoverFunc:       succeedDiscoveryFor("¯\\_(ツ)_/¯"),
				SendDfuCommandFunc: failDfuSend,
				PutTFTPFunc:        succeedPutTFTP,
			},

			// Should not do anything if both discovery attempts failed.
			expectedLog: []string{
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", normalService),
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", dfuService),
			},
		},

		// With configured address
		{
			description: "Sending DFU command successful to configured address",
			skip:        false,
			setup: &TestSetup{
				Image:   bytes.NewBufferString("mock image data"),
				Address: "127.0.0.1",
				Serial:  "",
				// What discover returns should be irrelevant in this case,
				// because it should not be called.
				DiscoverFunc:       succeedDiscoveryFor(dfuService),
				SendDfuCommandFunc: succeedDfuSend,
				PutTFTPFunc:        succeedPutTFTP,
			},

			// Should send DFU command to the configured address,
			// then transfer the image to the same address.
			expectedLog: []string{
				"SendDfuCommand | host: 127.0.0.1, port: 55567",
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", dfuService),
				"Sleep | duration: 5s",
				fmt.Sprintf("PutTFTP | host: 127.0.0.1 (via: %s), port: 69, image: mock image data", dfuService),
			},
		},
		{
			description: "Sending DFU command unsuccessful to configured address",
			skip:        false,
			setup: &TestSetup{
				Image:   bytes.NewBufferString("mock image data"),
				Address: "some-address",
				Serial:  "",
				DiscoverFunc:       succeedDiscoveryFor("none"),
				SendDfuCommandFunc: failDfuSend,
				PutTFTPFunc:        succeedPutTFTP,
			},

			// Should try to find the senso in DFU mode.
			expectedLog: []string{
				"SendDfuCommand | host: some-address, port: 55567",
				fmt.Sprintf("Discover | service: %s, deviceSerial: none", dfuService),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			setup := testCase.setup
			// Reset or initialize mocks and log collectors here if necessary
			mockDeps := &MockDeps{
				Log:   make([]string, 0),
				setup: setup,
			}

			if testCase.skip {
				t.Skip()
			} else {
				ctx := context.Background()
				Update(ctx, setup.Image, &setup.Address, &setup.Serial, mockDeps)
				assertLogsEqual(testCase.expectedLog, mockDeps.Log, t)
			}
		})
	}
}

// Mock input and dependencies for a test case
type TestSetup struct {
	Image              io.Reader
	DiscoverFunc       func(service string, deviceSerial *string, ctx context.Context) (addr string, err error)
	SendDfuCommandFunc func(host, port string) error
	PutTFTPFunc        func(host, port string, image io.Reader) error
	Address            string
	Serial             string
}

type MockDeps struct {
	setup *TestSetup
	Log   []string
}

// Mock implementations of the update functions dependencies

func (m *MockDeps) Sleep(d time.Duration) {
	logEntry := fmt.Sprintf("Sleep | duration: %v", d)
	m.Log = append(m.Log, logEntry)
}

func (m *MockDeps) Discover(ctx context.Context, service string, wantedSerial *string) (string, error) {
	serial := "none"
	if wantedSerial != nil && *wantedSerial != "" {
		serial = *wantedSerial
	}
	logEntry := fmt.Sprintf("Discover | service: %s, deviceSerial: %s", service, serial)
	m.Log = append(m.Log, logEntry)
	return m.setup.DiscoverFunc(service, wantedSerial, ctx)
}

func (m *MockDeps) SendDfuCommand(host, port string) error {
	logEntry := fmt.Sprintf("SendDfuCommand | host: %s, port: %v", host, port)
	m.Log = append(m.Log, logEntry)
	return m.setup.SendDfuCommandFunc(host, port)
}

func (m *MockDeps) PutTFTP(host, port string, image io.Reader) error {
	imageContent := "Error"
	bytes, err := io.ReadAll(image)
	if err == nil {
		imageContent = string(bytes)
	}
	logEntry := fmt.Sprintf("PutTFTP | host: %s, port: %v, image: %s", host, port, imageContent)
	m.Log = append(m.Log, logEntry)
	return m.setup.PutTFTPFunc(host, port, image)
}

// Functions for setting up test cases

func succeedDiscoveryFor(mode string) func(service string, serial *string, ctx context.Context) (_ string, _ error) {
	return func(service string, serial *string, ctx context.Context) (_ string, _ error) {
		if mode == "any" || mode == service {
			return fmt.Sprintf("127.0.0.1 (via: %s)", service), nil
		}
		return failDiscovery(service, serial, ctx)
	}

}
func failDiscovery(_ string, _ *string, _ context.Context) (_ string, _ error) {
	return "", errors.New("failed to discover")
}

func succeedDfuSend(_, _ string) error {
	return nil
}

func failDfuSend(_, _ string) error {
	return errors.New("failed sending dfu")
}

func succeedPutTFTP(_, _ string, _ io.Reader) error {
	return nil
}

func failPutTFTP(_, _ string, _ io.Reader) error {
	return errors.New("failed TFTP")
}

// Helpers

func assertLogsEqual(expected []string, actual []string, t *testing.T) {
	ok := reflect.DeepEqual(expected, actual)
	if ok {
		return
	}

	messageParts := []string{
		"",
		"",
		"Expected:",
		"--------------------------------------------------------------------------------",
		strings.Join(expected, "\n"),
		"--------------------------------------------------------------------------------",
		"",
		"Actual:",
		"--------------------------------------------------------------------------------",
		strings.Join(actual, "\n"),
		"--------------------------------------------------------------------------------",
		"",
	}

	t.Errorf(strings.Join(messageParts, "\n"))
}
