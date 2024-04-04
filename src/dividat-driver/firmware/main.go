package firmware

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pin/tftp"

	"github.com/dividat/driver/src/dividat-driver/service"
)

const tftpPort = "69"
const controllerPort = "55567"

// Command-line interface to Update
func Command(flags []string) {
	updateFlags := flag.NewFlagSet("update", flag.ExitOnError)
	imagePath := updateFlags.String("i", "", "Firmware image path")
	configuredAddr := updateFlags.String("a", "", "Senso address (optional)")
	sensoSerial := updateFlags.String("s", "", "Senso serial (optional)")
	updateFlags.Parse(flags)

	var deviceSerial *string = nil
	if *sensoSerial != "" {
		deviceSerial = sensoSerial
	}

	if *imagePath == "" {
		flag.PrintDefaults()
		return
	}
	file, err := os.Open(*imagePath)
	if err != nil {
		fmt.Printf("Could not open image file: %v\n", err)
		os.Exit(1)
	}

	onProgress := func(progressMsg string) {
		fmt.Println(progressMsg)
	}

	err = Update(context.Background(), file, deviceSerial, configuredAddr, onProgress)

	if err != nil {
		fmt.Println()
		fmt.Printf("Update failed: %v \n", err)
		os.Exit(1)
	}
}

type OnProgress func(msg string)

const tryPowerCycling = "Try turning the Senso off and on, waiting for 30 seconds and then running this update tool again."

func Update(parentCtx context.Context, image io.Reader, deviceSerial *string, configuredAddr *string, onProgress OnProgress) error {
	var target service.Service

	if configuredAddr != nil && *configuredAddr != "" {
		onProgress(fmt.Sprintf("Using specified address %s", *configuredAddr))
		match := service.Find(parentCtx, 15*time.Second, service.AddressFilter(*configuredAddr))
		if match == nil {
			return fmt.Errorf("Failed to find Senso with address %s.\n%s", *configuredAddr, tryPowerCycling)
		}
		target = *match
	} else if deviceSerial != nil && *deviceSerial != "" {
		onProgress(fmt.Sprintf("Using specified serial %s", *deviceSerial))
		match := service.Find(parentCtx, 15*time.Second, service.SerialNumberFilter(*deviceSerial))
		if match == nil {
			return fmt.Errorf("Failed to find Senso with serial number %s.\n%s", *configuredAddr, tryPowerCycling)
		}
		target = *match
	} else {
		// Discover controller address via mDNS
		onProgress("Discovering sensos")
		services := service.List(parentCtx, 15*time.Second)
		if len(services) == 1 {
			target = services[0]
			onProgress(fmt.Sprintf("Discovered Senso: %s (%s)", target.Text.Serial, target.Address))
		} else if len(services) == 0 {
			return fmt.Errorf("Could not find any Sensos.\n%s", tryPowerCycling)
		} else {
			return fmt.Errorf("discovered multiple Sensos: %v, please specify a serial or IP", services)
		}
	}

	if !service.IsDfuService(target) {
		trySendDfu := func() error {
			err := sendDfuCommand(target.Address, controllerPort, onProgress)
			return err
		}

		backoffStrategy := backoff.NewExponentialBackOff()
		backoffStrategy.MaxElapsedTime = 30 * time.Second
		backoffStrategy.MaxInterval = 10 * time.Second
		err := backoff.RetryNotify(trySendDfu, backoffStrategy, func(e error, d time.Duration) {
			onProgress(fmt.Sprintf("%v\nRetrying in %v", e, d))
		})

		if err != nil {
			return fmt.Errorf("could not send DFU command to Senso at %s: %s", target.Address, err)
		}

		onProgress("Looking for senso in bootloader mode")
		dfuService := service.Find(parentCtx, 30*time.Second, func(discovered service.Service) bool {
			return service.SerialNumberFilter(target.Text.Serial)(discovered) && service.IsDfuService(discovered)
		})

		if dfuService == nil {
			return fmt.Errorf("Could not rediscover Senso in bootloader mode.\n%s", tryPowerCycling)
		}

		target = *dfuService
		onProgress(fmt.Sprintf("Re-discovered Senso in bootloader mode at %s", target.Address))
		onProgress("Waiting 15 seconds to ensure proper TFTP startup")
		// Wait to ensure proper TFTP startup
		time.Sleep(15 * time.Second)
	} else {
		onProgress("Senso discovered in bootloader mode")
	}

	err := putTFTP(target.Address, tftpPort, image, onProgress)
	if err != nil {
		return err
	}

	onProgress("Success! Firmware transmitted to Senso.")
	return nil
}

func sendDfuCommand(host string, port string, onProgress OnProgress) error {
	// Header
	const PROTOCOL_VERSION = 0x00
	const NUMOFBLOCKS = 0x01
	reserve := bytes.Repeat([]byte{0x00}, 6)
	header := append([]byte{PROTOCOL_VERSION, NUMOFBLOCKS}, reserve...)

	// Message Body
	const BLOCKLENGTH = 0x0008
	const BLOCKTYPE_DFU = 0x00F0
	const MAGIC_KEY = 0xFA173CCD87664FBE
	body := make([]byte, 12)
	binary.LittleEndian.PutUint16(body[0:], BLOCKLENGTH)
	binary.LittleEndian.PutUint16(body[2:], BLOCKTYPE_DFU)
	binary.BigEndian.PutUint64(body[4:], MAGIC_KEY)

	command := append(header, body...)

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return fmt.Errorf("Could not dial connection to Senso controller at %s:%s: %v", host, port, err)
	}
	defer conn.Close()
	time.Sleep(1 * time.Second)

	_, err = io.Copy(conn, bytes.NewReader(command))
	if err != nil {
		return fmt.Errorf("Could not send DFU command: %v", err)
	}

	onProgress(fmt.Sprintf("Sent DFU command to %s:%s.", host, port))

	return nil
}

func putTFTP(host string, port string, image io.Reader, onProgress OnProgress) error {
	onProgress("Creating TFTP client")
	client, err := tftp.NewClient(fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return fmt.Errorf("Could not create tftp client: %v", err)
	}

	maxRetries := 5
	client.SetRetries(maxRetries)

	expDelay := func(attempt int) time.Duration {
		exp := math.Pow(2, float64(attempt))
		exp = math.Min(exp, 60)
		return time.Duration(int(exp)) * time.Second
	}

	client.SetBackoff(func(attempt int) time.Duration {
		a1 := attempt + 1
		msg := fmt.Sprintf("Failed on attempt %d, retrying in %v", a1, expDelay(a1))
		onProgress(msg)
		return expDelay(attempt)
	})

	onProgress("Preparing transmission")
	rf, err := client.Send("controller-app.bin", "octet")
	if err != nil {
		return fmt.Errorf("Could not create send connection: %v", err)
	}
	onProgress("Transmitting...")
	n, err := rf.ReadFrom(image)
	if err != nil {
		return fmt.Errorf("Could not read from file: %v", err)
	}
	onProgress(fmt.Sprintf("%d bytes sent", n))
	return nil
}
