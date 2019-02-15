package firmware

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pin/tftp"
	"github.com/grandcat/zeroconf"
)

// Flags

const tftpPort = "69"
const controllerPort = "55567"

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
		abort(fmt.Sprintf("Could not open image file: %v", err))
	}

	Update(context.Background(), file, deviceSerial, configuredAddr)
}

func Update(ctx context.Context, image io.Reader, deviceSerial *string, configuredAddr *string) {
	// Discover Senso IP
	var controllerHost string
	if *configuredAddr == "" {
		ctx, _ := context.WithTimeout(ctx, 5 * time.Second)
		discoveredAddr, err := discover("_sensoControl._tcp", deviceSerial, ctx)
		if err != nil {
			abort(err.Error())
		}

		controllerHost = discoveredAddr
	} else {
		controllerHost = *configuredAddr
	}

	// Request reboot into boot controller
	err := sendDfuCommand(controllerHost, controllerPort)
	if err != nil {
		abort(err.Error())
	}

	// Re-discover Senso IP in case it changes on reboot
	var dfuHost string
	if *configuredAddr == "" {
		ctx, _ := context.WithTimeout(ctx, 60 * time.Second)
		discoveredAddr, err := discover("_sensoUpdate._udp", deviceSerial, ctx)
		if err != nil {
			// Try to discover boot controller via legacy identifier
			ctx, _ := context.WithTimeout(ctx, 60 * time.Second)
			legacyDiscoveredAddr, err := discover("_sensoControl._tcp", deviceSerial, ctx)
			if err != nil {
				abort(err.Error())
			}
			dfuHost = legacyDiscoveredAddr
		} else {
			dfuHost = discoveredAddr
		}
	} else {
		dfuHost = *configuredAddr
	}

	// Wait briefly after discovery to ensure proper TFTP startup
	time.Sleep(5 * time.Second)

	// Transmit firmware via TFTP
	err = putTFTP(dfuHost, tftpPort, image)
	if err != nil {
		abort(err.Error())
	}
	fmt.Println("Firmware transmitted to Senso.")
}

func sendDfuCommand(host string, port string) error {
	// Header
	const PROTOCOL_VERSION = 0x00
	const NUMOFBLOCKS = 0x01
	reserve := bytes.Repeat([]byte{ 0x00 }, 6)
	header := append([]byte{ PROTOCOL_VERSION, NUMOFBLOCKS }, reserve...)

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

	fmt.Printf("Sent DFU command to %s:%s.\n", host, port)

	return nil
}

func putTFTP(host string, port string, image io.Reader) error {
	client, err := tftp.NewClient(fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return fmt.Errorf("Could not create tftp client: %v", err)
	}
	rf, err := client.Send("controller-app.bin", "octet")
	if err != nil {
		return fmt.Errorf("Could not create send connection: %v", err)
	}
	n, err := rf.ReadFrom(image)
	if err != nil {
		return fmt.Errorf("Could not read from file: %v", err)
	}
	fmt.Printf("%d bytes sent\n", n)
	return nil
}

func discover(service string, deviceSerial *string, ctx context.Context) (addr string, err error) {

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		err = fmt.Errorf("Initializing discovery failed: %v", err)
		return
	}

	fmt.Printf("Starting discovery: %s\n", service)

	entries := make(chan *zeroconf.ServiceEntry)

	err = resolver.Browse(ctx, service, "local.", entries)
	if err != nil {
		err = fmt.Errorf("Browsing failed: %v", err)
		return
	}

	devices := make(map[string][]string)
	entriesWithoutSerial := 0
	select {
	case entry := <-entries:
		if entry == nil {
			break
		}

		var serial string
		for ix, txt := range entry.Text {
			if strings.HasPrefix(txt, "ser_no=") {
				serial = strings.TrimPrefix(txt, "ser_no=")
				break
			} else if ix == len(entry.Text) - 1 {
				entriesWithoutSerial++
				serial = fmt.Sprintf("UNKNOWN-%d", entriesWithoutSerial)
			}
		}
		if deviceSerial != nil && serial != *deviceSerial {
			break
		}
		for _, addrCandidate := range entry.AddrIPv4 {
			if addrCandidate.String() == "0.0.0.0" {
				fmt.Printf("Skipping discovered address 0.0.0.0 for %s.\n", serial)
			} else {
				devices[serial] = append(devices[serial], addrCandidate.String())
			}
		}
	}



	if len(devices) == 0 && deviceSerial == nil {
		err = errors.New("No Sensos discovered.")
	} else if len(devices) == 0 && deviceSerial != nil {
		err = fmt.Errorf("Could not find Senso %s.", *deviceSerial)
	} else if len(devices) == 1 {
		for serial, addrs := range devices {
			addr = addrs[0]
			fmt.Printf("Discovered %s at %v, using %s.\n", serial, addrs, addr)
			return
		}
	} else {
		err = fmt.Errorf("Discovered multiple Sensos: %v. Please specify a serial or IP.", devices)
		return
	}
	return
}

func abort(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
