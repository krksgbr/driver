package firmware

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dividat/driver/src/dividat-driver/service"
)

// Command-line interface to Update
func Command(flags []string) {
	updateFlags := flag.NewFlagSet("update", flag.ExitOnError)
	imagePath := updateFlags.String("i", "", "Firmware image path")
	configuredAddr := updateFlags.String("a", "", "Senso address (optional)")
	sensoSerial := updateFlags.String("s", "", "Senso serial (optional)")
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

	onProgress := func(progressMsg string) {
		fmt.Println(progressMsg)
	}

	if *sensoSerial != "" {
		err = UpdateBySerial(context.Background(), *sensoSerial, file, onProgress)
	} else if *configuredAddr != "" {
		err = updateByAddress(context.Background(), *configuredAddr, file, onProgress)
	} else {
		err = updateByDiscovery(context.Background(), file, onProgress)
	}

	if err != nil {
		fmt.Println()
		fmt.Printf("Update failed: %v \n", err)
		os.Exit(1)
	}
}

func updateByAddress(ctx context.Context, address string, image io.Reader, onProgress OnProgress) error {
	onProgress(fmt.Sprintf("Using specified address %s", address))
	match := service.Find(ctx, 15*time.Second, service.AddressFilter(address))
	if match == nil {
		return fmt.Errorf("Failed to find Senso with address %s.\n%s", address, tryPowerCycling)
	}

	return update(ctx, *match, image, onProgress)
}

func updateByDiscovery(ctx context.Context, image io.Reader, onProgress OnProgress) error {
	onProgress("Discovering sensos")
	services := service.List(ctx, 15*time.Second)
	if len(services) == 1 {
		target := services[0]
		onProgress(fmt.Sprintf("Discovered Senso: %s (%s)", target.Text.Serial, target.Address))
		return update(ctx, target, image, onProgress)
	} else if len(services) == 0 {
		return fmt.Errorf("Could not find any Sensos.\n%s", tryPowerCycling)
	} else {
		return fmt.Errorf("discovered multiple Sensos: %v, please specify a serial or IP", services)
	}
}
