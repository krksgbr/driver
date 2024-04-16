package firmware

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dividat/driver/src/dividat-driver/service"
)

// Command-line interface to running a firmware update
func Command(flags []string) {
	updateFlags := flag.NewFlagSet("update", flag.ExitOnError)
	imagePath := updateFlags.String("i", "", "Firmware image path")
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

	tryPowerCycling := "Try turning the Senso off and on, waiting for 30 seconds and then running this update tool again."
	suggestPowerCycling := false

	if *sensoSerial != "" {
		err = UpdateBySerial(context.Background(), *sensoSerial, file, onProgress)
		if err != nil {
			suggestPowerCycling = true
		}
	} else {
		err, suggestPowerCycling = updateByDiscovery(context.Background(), file, onProgress)
	}

	if err != nil {
		fmt.Println()
		fmt.Printf("Update failed: %v \n", err)
		if suggestPowerCycling {
			fmt.Println(tryPowerCycling)
		}
		os.Exit(1)
	}
}

func updateByDiscovery(ctx context.Context, image io.Reader, onProgress OnProgress) (err error, suggestPowerCycling bool) {
	onProgress("Discovering Sensos")
	services := service.List(ctx, discoveryTimeout)
	if len(services) == 1 {
		target := services[0]
		onProgress(fmt.Sprintf("Discovered Senso: %s (%s)", target.Text.Serial, target.Address))
		err = update(ctx, target, image, onProgress)
		if err != nil {
			suggestPowerCycling = true
		}
	} else if len(services) == 0 {
		err = fmt.Errorf("Could not find any Sensos.")
		suggestPowerCycling = true
	} else {
		err = fmt.Errorf("discovered multiple Sensos: %v, please specify a serial or IP", services)
		suggestPowerCycling = false
	}
	return
}
