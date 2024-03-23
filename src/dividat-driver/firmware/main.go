package firmware

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/pin/tftp"
)

const tftpPort = "69"
const controllerPort = "55567"

const normalService = "_sensoControl._tcp" // Service type for Sensos in normal mode
const dfuService = "_sensoUpdate._udp"     // Service type for Sensos in DFU mode

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

	updateDeps := UpdateDepsImpl{}
	err = Update(context.Background(), file, deviceSerial, configuredAddr, updateDeps)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println()
		fmt.Println("Update failed. Try turning the Senso off and on, waiting for 30 seconds and then running this update tool again.")
		os.Exit(1)
	}
}

// Update function dependencies
type UpdateDeps interface {
	Discover(service string, deviceSerial *string, ctx context.Context) (addr string, err error)
	SendDfuCommand(host, port string) error
	PutTFTP(host, port string, image io.Reader) error
	Sleep(duration time.Duration)
}

// Firmware update workhorse
func Update(parentCtx context.Context, image io.Reader, deviceSerial *string, configuredAddr *string, impl UpdateDeps) (fail error) {
	// 1: Find address of a Senso in normal mode
	var controllerHost string
	if *configuredAddr != "" {
		// Use specified controller address
		controllerHost = *configuredAddr
		fmt.Printf("Using specified controller address '%s'.\n", controllerHost)
	} else {
		// Discover controller address via mDNS
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
		discoveredAddr, err := impl.Discover(normalService, deviceSerial, ctx)
		cancel()
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		} else {
			controllerHost = discoveredAddr
		}
	}

	// 2: Switch the Senso to bootloader mode
	if controllerHost != "" {
		err := impl.SendDfuCommand(controllerHost, controllerPort)
		if err != nil {
			fmt.Printf("Could not send DFU command to Senso at %s: %s\n", controllerHost, err)
		}
	} else {
		fmt.Printf("Could not discover a Senso in regular mode, now trying to detect a Senso already in bootloader mode.\n")
	}

	// 3: Find address of Senso in bootloader mode
	var dfuHost string
	if *configuredAddr != "" {
		dfuHost = *configuredAddr
	} else {
		ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
		discoveredAddr, err := impl.Discover(dfuService, deviceSerial, ctx)
		cancel()
		if err != nil {
			// Try to discover boot controller via legacy identifier
			ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
			legacyDiscoveredAddr, err := impl.Discover(normalService, deviceSerial, ctx)
			cancel()
			if err == nil {
				dfuHost = legacyDiscoveredAddr
			} else if controllerHost != "" {
				fmt.Printf("Could not discover update service, trying to fall back to previous discovery %s.\n", controllerHost)
				dfuHost = controllerHost
			} else {
				fail = fmt.Errorf("Could not find any Senso bootloader to transmit firmware to: %s", err)
				return
			}
		} else {
			dfuHost = discoveredAddr
		}
	}

	// 4: Transmit firmware via TFTP
	impl.Sleep(5 * time.Second) // Wait to ensure proper TFTP startup
	err := impl.PutTFTP(dfuHost, tftpPort, image)
	if err != nil {
		fail = err
		return
	}

	fmt.Println("Success! Firmware transmitted to Senso.")
	return
}

type UpdateDepsImpl struct{}

func (u UpdateDepsImpl) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (u UpdateDepsImpl) SendDfuCommand(host string, port string) error {
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

	fmt.Printf("Sent DFU command to %s:%s.\n", host, port)

	return nil
}

func (u UpdateDepsImpl) PutTFTP(host string, port string, image io.Reader) error {
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

func (u UpdateDepsImpl) Discover(service string, deviceSerial *string, ctx context.Context) (addr string, err error) {
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
				serial = cleanSerial(strings.TrimPrefix(txt, "ser_no="))
				break
			} else if ix == len(entry.Text)-1 {
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
		err = fmt.Errorf("Could not find any devices for service %s.", service)
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

func cleanSerial(serialStr string) string {
	// Senso firmware up to 3.8.0 adds garbage at end of serial in mDNS
	// entries due to improper string sizing.  Because bootloader firmware
	// will not be updated via Ethernet, the problem will stay around for a
	// while and we clean up the serial here to produce readable output for
	// older devices.
	return strings.Split(serialStr, "\\000")[0]
}
