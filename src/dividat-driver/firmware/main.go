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
	configuredSerial := updateFlags.String("s", "", "Senso serial (optional)")
	updateFlags.Parse(flags)

	if *imagePath == "" {
		flag.PrintDefaults()
		return
	}
	file, err := os.Open(*imagePath)
	if err != nil {
		fmt.Printf("Could not open image file: %v\n", err)
		os.Exit(1)
	}

	err = Update(context.Background(), file, configuredAddr, configuredSerial, UpdateDepsImpl{})
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println()
		fmt.Println("Update failed. Try turning the Senso off and on, waiting for 30 seconds and then running this update tool again.")
		os.Exit(1)
	}

	os.Exit(0)

}

// Update function dependencies
type UpdateDeps interface {
	Discover(ctx context.Context, service string, wantedSerial *string) (string, error)
	SendDfuCommand(host, port string) error
	PutTFTP(host, port string, image io.Reader) error
	Sleep(duration time.Duration)
}

func Update(ctx context.Context, image io.Reader, configuredAddr *string, wantedSerial *string, impl UpdateDeps) (fail error) {
	if *configuredAddr != "" {
		host := Host{
			addr:    *configuredAddr,
			service: normalService, // Assume it's in normal mode
		}
		return UpdateByAddress(ctx, image, host, impl)
	}
	return UpdateByDiscovery(ctx, image, wantedSerial, impl)
}

func UpdateByDiscovery(parentCtx context.Context, image io.Reader, wantedSerial *string, impl UpdateDeps) (fail error) {
	controllerHost, err := DiscoverAny(parentCtx, wantedSerial, impl)
	if err != nil {
		fail = fmt.Errorf("Discovery failed.")
		return
	}
	return UpdateByAddress(parentCtx, image, controllerHost, impl)
}

func DiscoverAny(parentCtx context.Context, wantedSerial *string, impl UpdateDeps) (host Host, fail error) {
	host, fail = DiscoverWithTimeout(parentCtx, 15*time.Second, normalService, wantedSerial, impl)
	if fail != nil {
		host, fail = DiscoverWithTimeout(parentCtx, 15*time.Second, dfuService, wantedSerial, impl)
	}
	return host, fail
}

// Represents a host that has already been discovered
type Host struct {
	addr    string
	service string
}

func UpdateByAddress(parentCtx context.Context, image io.Reader, host Host, impl UpdateDeps) (fail error) {
	var dfuHost Host
	// 1: Try to send DFU command to Senso, unless it's already known to be in bootloader mode
	if host.service == normalService {
		err := impl.SendDfuCommand(host.addr, controllerPort)

		// If sending the DFU command failed, the Senso could already be in bootloader mode.
		// Keep going.
		if err != nil {
			fmt.Printf("Could not send DFU command to Senso at %s: %s\n", host.addr, err)
		}

		// 2: (Re-)discover Senso in DFU mode
		discoveredHost, discoveryError := DiscoverWithTimeout(parentCtx, 60*time.Second, dfuService, nil, impl)
		if discoveryError != nil {
			fail = discoveryError
			return
		}
		dfuHost = discoveredHost
	} else {
		dfuHost = host
	}

	// 2. Try transferring the firmware via TFTP
	// Wait to ensure proper TFTP startup
	impl.Sleep(5 * time.Second)
	err := impl.PutTFTP(dfuHost.addr, tftpPort, image)
	if err != nil {
		fail = err
		return
	}

	fmt.Println("Success! Firmware transmitted to Senso.")
	return
}

func DiscoverWithTimeout(
	parentCtx context.Context,
	duration time.Duration,
	service string,
	wantedSerial *string,
	impl UpdateDeps,
) (Host, error) {
	ctx, cancel := context.WithTimeout(parentCtx, duration)
	addr, err := impl.Discover(ctx, service, wantedSerial)
	cancel()
	return Host{addr: addr, service: service}, err
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

func (u UpdateDepsImpl) Discover(ctx context.Context, service string, wantedSerial *string) (addr string, err error) {
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
	return resolveEntries(entries, wantedSerial)
}

func resolveEntries(entries chan *zeroconf.ServiceEntry, wantedSerial *string) (addr string, err error) {
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
		if wantedSerial != nil && serial != *wantedSerial {
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

	if len(devices) == 0 && wantedSerial == nil {
		err = fmt.Errorf("Serial not found.")
	} else if len(devices) == 0 && wantedSerial != nil {
		err = fmt.Errorf("Could not find Senso %s.", *wantedSerial)
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
