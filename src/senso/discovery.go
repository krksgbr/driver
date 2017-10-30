package senso

import (
	"context"

	"github.com/grandcat/zeroconf"
)

// Discover Sensos for a certain duration
func (handle *Handle) Discover(ctx context.Context) chan *zeroconf.ServiceEntry {

	log := handle.log

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Error("initializing discovery failed:", err.Error())
	}

	log.Info("initialized discovery")

	// create an intermediary channel for logging discoveries and handling the case when there is no reader
	entries := make(chan *zeroconf.ServiceEntry)

	err = resolver.Browse(ctx, "_sensoControl._tcp", "local.", entries)
	if err != nil {
		log.Error("browsing failed:", err.Error())
	}

	return entries

	// for entry := range entries {
	// log.Println(entry)
	// }
	// log.Println("Finished Discovery")

}
