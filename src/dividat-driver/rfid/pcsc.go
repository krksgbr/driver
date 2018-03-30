package rfid

import (
	"context"
	"fmt"
	"time"

	"github.com/ebfe/scard"
	"github.com/sirupsen/logrus"
)

var READER_POLLING_INTERVAL = 500 * time.Millisecond
var CARD_POLLING_INTERVAL = 50 * time.Millisecond

// Special MSFT name to bolt plug&play onto PC/SC
// Supported by winscard and libpcsc
const MAGIC_PNP_NAME = "\\\\?PnP?\\Notification"

// APDU to retrieve a card's UID
var UID_APDU = []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}

func pollSmartCard(ctx context.Context, log *logrus.Entry, callback func(string)) {
	// Establish a PC/SC context
	scard_ctx, err := scard.EstablishContext()
	if err != nil {
		log.WithError(err).Error("Error EstablishContext:")
		return
	}
	defer scard_ctx.Release()

	// Detect PnP support
	pnpReaderStates := []scard.ReaderState{
		makeReaderState(MAGIC_PNP_NAME),
	}
	scard_ctx.GetStatusChange(pnpReaderStates, 0)
	hasPnP := !is(pnpReaderStates[0].EventState, scard.StateUnknown)

	log.WithField("pnp", hasPnP).Info("Starting RFID scanner.")

	var availableReaders []string = make([]string, 0)
	var lastKnownState map[string]scard.StateFlag = map[string]scard.StateFlag{}

	for {

		select {
		case <-ctx.Done():
			log.Info("Stopping RFID scanner.")
			return

		default:
			// Retrieve available readers
			newReaders, err := scard_ctx.ListReaders()
			if err != nil {
				log.WithError(err).Error("Error listing readers.")
				return
			}
			announceReaderChanges(log, availableReaders, newReaders)
			availableReaders = newReaders

			// Wait for readers to appear
			if len(availableReaders) == 0 {
				if hasPnP {
					// `GetStatusChange` acts as a smarter sleep that finishes early
					pnpReaderStates := []scard.ReaderState{
						makeReaderState(MAGIC_PNP_NAME),
					}
					scard_ctx.GetStatusChange(pnpReaderStates, READER_POLLING_INTERVAL)
				} else {
					time.Sleep(READER_POLLING_INTERVAL)
				}

				// Restart loop to list readers
				continue
			}

			// Now there are available readers
			// Wait for card presence
			readerStates := make([]scard.ReaderState, len(availableReaders))
			for ix, readerName := range availableReaders {
				flag, ok := lastKnownState[readerName]
				if ok {
					readerStates[ix] = makeReaderState(readerName, flag)
				} else {
					readerStates[ix] = makeReaderState(readerName)
				}
			}
			// We could block until a card status changes here, but have to
			// unblock periodically to check if our context has been cancelled
			code := scard_ctx.GetStatusChange(readerStates, CARD_POLLING_INTERVAL)
			if code == scard.ErrTimeout {
				continue
			}

			// One or more readers changed their status
			for _, readerState := range readerStates {
				fmt.Println("LOOPING readers")

				if readerState.CurrentState == readerState.EventState {
					// This reader has not changed.
					continue
				}

				if is(readerState.EventState, scard.StateChanged) {
					// Event state becomes current state
					readerState.CurrentState = readerState.EventState
					// Keep track of last known state for next refresh cycle
					lastKnownState[readerState.Reader] = readerState.CurrentState
				} else {
					continue
				}

				if !is(readerState.CurrentState, scard.StatePresent) {
					// This reader has no card.
					continue
				}

				// Connect to the card
				card, err := scard_ctx.Connect(readerState.Reader, scard.ShareShared, scard.ProtocolAny)
				if err != nil {
					log.WithError(err).Error("Error connecting to card.")
					continue
				}

				// Request UID
				response, err := card.Transmit(UID_APDU)
				if err != nil {
					log.WithError(err).Debug("Failed while transmitting APDU.")
					continue
				}
				uid := ""
				for i := 0; i < len(response)-2; i++ {
					uid += fmt.Sprintf("%X", response[i])
				}
				if len(uid) > 0 {
					log.Info("Detected RFID token.")
					callback(uid)
				}

				card.Disconnect(scard.UnpowerCard)
			}
		}
	}
}

// Helpers

func is(mask scard.StateFlag, flag scard.StateFlag) bool {
	return mask&flag != 0
}

func announceReaderChanges(log *logrus.Entry, previous []string, current []string) {
	for _, name := range previous {
		if !contains(current, name) {
			log.Info(fmt.Sprintf("Reader became unavailable: '%s'", name))
		}
	}
	for _, name := range current {
		if !contains(previous, name) {
			log.Info(fmt.Sprintf("Reader became available: '%s'", name))
		}
	}
}

func contains(arr []string, name string) bool {
	for _, member := range arr {
		if member == name {
			return true
		}
	}
	return false
}

func makeReaderState(name string, state ...scard.StateFlag) scard.ReaderState {
	flag := scard.StateUnaware
	if len(state) == 1 {
		flag = state[0]
	}
	return scard.ReaderState{Reader: name, CurrentState: flag}
}
