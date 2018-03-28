package rfid

import (
	"context"
	"fmt"
	"time"

	"github.com/ebfe/scard"
	"github.com/sirupsen/logrus"
)

const UID_APDU = []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}

func pollSmartCard(ctx context.Context, log *logrus.Entry, callback func(string)) {
	log.Info("Starting RFID scanner.")

	// Establish a PC/SC context
	context, err := scard.EstablishContext()
	if err != nil {
		log.WithError(err).Error("Error EstablishContext:")
		return
	}
	defer context.Release()

	var reader *string
	var uid *string

	for {

		// Throttle loop
		time.Sleep(50 * time.Millisecond)

		select {
		case <-ctx.Done():
			log.Info("Stopping RFID scanner.")
			return

		default:
			// TODO Use GetStatusChange to avoid polling on Linux/Win
			readers, err := context.ListReaders()
			if err != nil {
				log.WithError(err).Error("Error ListReaders:")
				return
			}

			if len(readers) == 0 {
				reader = nil
				uid = nil

				continue
			}

			// TODO This just uses the first reader. What is a proper seletion criterion?
			if reader == nil || *reader != readers[0] {
				reader = &readers[0]
				log.Info(fmt.Sprintf("Using reader: %s", *reader))
			}

			// Wait for card presence
			readerStates := []scard.ReaderState{
				{Reader: *reader},
			}
			context.GetStatusChange(readerStates, -1)
			if (readerStates[0].EventState & scard.StatePresent) == 0 {
				continue
			}

			// Connect to the card
			card, err := context.Connect(*reader, scard.ShareShared, scard.ProtocolAny)
			if err != nil {
				log.WithError(err).Error("Error Connect:")
				uid = nil
				continue
			}

			// Request UID
			response, err := card.Transmit(UID_APDU)
			if err != nil {
				log.WithError(err).Debug("Error Transmit:")
				uid = nil
				continue
			}
			scanned_uid := ""
			for i := 0; i < len(response)-2; i++ {
				scanned_uid += fmt.Sprintf("%X", response[i])
			}
			if len(scanned_uid) > 0 && (uid == nil || *uid != scanned_uid) {
				uid = &scanned_uid
				log.Info("Detected RFID token.")
				callback(*uid)
			}

			card.Disconnect(scard.LeaveCard)
		}
	}
}
