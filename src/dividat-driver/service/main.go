package service

// This module contains functions to discover Sensos via mDNS.
import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/zeroconf/v2"
)

// Represents a service that has been discovered.
// Relevant information about the service is lifted
// out of the zeroconf record for ease of use.
type Service struct {
	Text         Text
	Address      string
	ServiceEntry zeroconf.ServiceEntry
}

// Information parsed from services' txt records.
type Text struct {
	Serial string
	Mode   string
}

func (s Service) String() string {
	return fmt.Sprintf("{ Serial: %s, Address: %s}", s.Text.Serial, s.Address)
}

// Service type, indicating what mode a Senso is in.
// SensoUpdate means the Senso is in bootloader mode,
// while SensoControl means the Senso is in "normal" mode.
type ServiceType string

const (
	SensoUpdate  ServiceType = "_sensoUpdate._udp"
	SensoControl ServiceType = "_sensoControl._tcp"
)

// Up to firmware 2.0.0.0 the bootloader advertised itself
// with the same service identifier as the application level
// firmware. Because of this we also have to check the `mode`
// field on a service's txt records to determine what mode a
// Senso is in. The enum below represents the possible values
// of this field.
const (
	ApplicationMode = "Application"
	BootloaderMode  = "Bootloader"
)

// Scan for services of a specific type, ie `SensoUpdate` or `SensoControl`.
func scanForType(ctx context.Context, t ServiceType, results chan<- Service, wg *sync.WaitGroup) {
	wg.Add(2)
	// Zeroconf closes the channel on context cancellation,
	// so we cannot share channels between multiple browse calls.
	// Doing so would lead to panic as one instance would try to close
	// a channel that was already closed by another instance.
	// To prevent this, we create an intermediate channel for each instance,
	// then forward the discovered service entries to the main results channel in
	// a separate goroutine.
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

// Scan for both types of services concurrently.
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

// Like `Scan`, but blocking.
// Returns a slice of services found within the specified timeout.
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

// Type alias to a filter function that can be used with `Find`.
// Helpers to construct commonly used filters are defined below.
type Filter = func(service Service) bool

// Looks for a service specified by the filter function.
// As soon as a match is found, it cancels scanning
// and returns the match.
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

// Commonly used filters to look for services.

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

// Helper to parse relevant information from the
// txt record of a service entry.
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

// Senso firmware up to 3.8.0 adds garbage at end of serial in mDNS
// entries due to improper string sizing. Because bootloader firmware
// will not be updated via Ethernet, the problem will stay around for a
// while and we clean up the serial here to produce readable output for
// older devices.
func cleanSerial(serialStr string) string {
	return strings.Split(serialStr, "\\000")[0]
}
