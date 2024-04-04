package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/zeroconf/v2"
)

type Service struct {
	Text         Text
	Address      string
	ServiceEntry zeroconf.ServiceEntry
}

func (s Service) String() string {
	return fmt.Sprintf("{ Serial: %s, Address: %s}", s.Text.Serial, s.Address)
}

type ServiceType string

const (
	SensoUpdate  ServiceType = "_sensoUpdate._udp"
	SensoControl ServiceType = "_sensoControl._tcp"
)

const (
	ApplicationMode = "Application"
	BootloaderMode  = "Bootloader"
)

func scanForType(ctx context.Context, t ServiceType, results chan<- Service, wg *sync.WaitGroup) {
	wg.Add(2)
	// Zeroconf closes the channel on context cancellation,
	// so we cannot share channels between multiple browse calls.
	// Doing so would lead to panic as one instance would try to close
	// a channel that was already closed by another instance.
	localEntries := make(chan *zeroconf.ServiceEntry)
	go func() {
		defer wg.Done()
		err := zeroconf.Browse(ctx, string(t), "local.", localEntries)
		if err != nil {
			fmt.Println("Discovery error:", err)
		}
	}()

	// Forward entries from localEntries to the main results channel
	go func() {
		defer wg.Done()
		entriesWithoutSerial := 0
		for entry := range localEntries {
			if entry != nil {
				text := getText(*entry)
				if text.Serial == "" {
					text.Serial = fmt.Sprintf("UNKNOWN-%d", entriesWithoutSerial)
					entriesWithoutSerial++
				}
				var address string
				if entry.AddrIPv4[0] != nil {
					address = entry.AddrIPv4[0].String()
				} else {
					continue
				}
				results <- Service{
					Address:      address,
					Text:         text,
					ServiceEntry: *entry,
				}
			}
		}
	}()
}

func Scan(ctx context.Context) chan Service {
	var wg sync.WaitGroup
	services := make(chan Service)
	scanForType(ctx, SensoUpdate, services, &wg)
	scanForType(ctx, SensoControl, services, &wg)
	go func() {
		wg.Wait()
		close(services)
	}()
	return services
}

func List(ctx context.Context, timeout time.Duration) []Service {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var result []Service
	services := Scan(ctx)

	for service := range services {
		result = append(result, service)
	}

	return result
}

type Filter = func(service Service) bool

func SerialNumberFilter(wantedSerial string) Filter {
	return func(service Service) bool {
		return service.Text.Serial == wantedSerial
	}
}

func AddressFilter(wantedAddress string) Filter {
	return func(service Service) bool {
		return service.Address == wantedAddress
	}
}

func IsDfuService(service Service) bool {
	return service.ServiceEntry.Service == string(SensoUpdate) || service.Text.Mode == BootloaderMode
}

func Find(ctx context.Context, timeout time.Duration, filter Filter) *Service {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	services := Scan(ctx)

	for service := range services {
		if filter(service) {
			cancel()
			return &service
		}
	}

	return nil
}

type Text struct {
	Serial string
	Mode   string
}

func getText(entry zeroconf.ServiceEntry) Text {
	text := Text{
		Serial: "",
		Mode:   "",
	}
	for _, txtField := range entry.Text {
		if strings.HasPrefix(txtField, "ser_no=") {
			text.Serial = cleanSerial(strings.TrimPrefix(txtField, "ser_no="))
		} else if strings.HasPrefix(txtField, "mode=") {
			text.Mode = strings.TrimPrefix(txtField, "mode=")
		}
	}
	return text
}

func cleanSerial(serialStr string) string {
	// Senso firmware up to 3.8.0 adds garbage at end of serial in mDNS
	// entries due to improper string sizing.  Because bootloader firmware
	// will not be updated via Ethernet, the problem will stay around for a
	// while and we clean up the serial here to produce readable output for
	// older devices.
	return strings.Split(serialStr, "\\000")[0]
}
