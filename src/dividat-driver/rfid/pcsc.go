package rfid

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/ebfe/scard"
	"github.com/sirupsen/logrus"
)

var READER_POLLING_INTERVAL = 1 * time.Second
var CARD_POLLING_TIMEOUT = 1 * time.Second

// Special MSFT name to bolt plug&play onto PC/SC
// Supported by winscard and libpcsc
const MAGIC_PNP_NAME = "\\\\?PnP?\\Notification"

// APDU to retrieve a card's UID
var UID_APDU = []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}
var NO_BUZZ_APDU = []byte{0xFF, 0x00, 0x52, 0x00, 0x00}

func pollSmartCard(ctx context.Context, log *logrus.Entry, onToken func(string), onReadersChange func([]string)) {

	scardContextBackoff := backoff.NewExponentialBackOff()
	scardContextBackoff.MaxElapsedTime = 0
	scardContextBackoff.MaxInterval = 2 * time.Minute

	for {
		// Establish a PC/SC context
		scard_ctx, err := scard.EstablishContext()
		if err != nil {
			log.WithError(err).Error("Could not create smart card context.")

			select {
			case <-time.After(scardContextBackoff.NextBackOff()):
				continue
			case <-ctx.Done():
				return
			}
		}
		defer scard_ctx.Release()

		// Now we have a context
		// Detect PnP support
		pnpReaderStates := []scard.ReaderState{
			makeReaderState(MAGIC_PNP_NAME),
		}
		scard_ctx.GetStatusChange(pnpReaderStates, 0)
		hasPnP := !is(pnpReaderStates[0].EventState, scard.StateUnknown)

		log.WithField("pnp", hasPnP).Info("Starting RFID scanner.")

		go waitForCardActivity(log, scard_ctx, hasPnP, onToken, onReadersChange)

		<-ctx.Done()
		// Cancel `GetStatusChange`
		scard_ctx.Cancel()

		log.Info("Stopping RFID scanner.")
		return
	}
}

func waitForCardActivity(log *logrus.Entry, scard_ctx *scard.Context, hasPnP bool, onToken func(string), onReadersChange func([]string)) {
	knownReaders := map[string]ReaderProfile{}

	updateKnownReaders := func(log *logrus.Entry, onReadersChange func([]string), current []string) {
		hasListChanged := false
		// Detect reader removal
		for name := range knownReaders {
			if !contains(current, name) {
				delete(knownReaders, name)
				log.Info(fmt.Sprintf("Reader became unavailable: '%s'", name))
				hasListChanged = true
			}
		}
		// Detect reader appearance
		for _, name := range current {
			if _, present := knownReaders[name]; !present {
				knownReaders[name] = ReaderProfile{
					lastKnownState: scard.StateUnknown,
					lastKnownToken: nil,
				}
				log.Info(fmt.Sprintf("Reader became available: '%s'", name))
				hasListChanged = true
			}
		}

		if hasListChanged {
			onReadersChange(normalizeReaderList(current))
		}
	}

	for {
		// Retrieve available readers
		newReaders, err := scard_ctx.ListReaders()
		if err != nil {
			// TODO With pcsclite this fails if there are no smart card readers. Too noisy.
			log.WithError(err).Debug("Error listing readers.")
		}
		updateKnownReaders(log, onReadersChange, newReaders)

		// Wait for readers to appear
		if len(knownReaders) == 0 {
			if hasPnP {
				// `GetStatusChange` acts as a smarter sleep that finishes early
				code := scard_ctx.GetStatusChange(
					[]scard.ReaderState{makeReaderState(MAGIC_PNP_NAME)},
					READER_POLLING_INTERVAL,
				)
				if code == scard.ErrCancelled {
					return
				}
			} else {
				time.Sleep(READER_POLLING_INTERVAL)
			}

			// Restart loop to list readers
			continue
		}

		// Now there are available readers
		// Wait for card presence
		readerStates := []scard.ReaderState{}
		for readerName, readerProfile := range knownReaders {
			// Restore last known state
			readerStates = append(readerStates, makeReaderState(readerName, readerProfile.lastKnownState))
		}
		// We need to timeout perodically to check for new readers
		code := scard_ctx.GetStatusChange(readerStates, CARD_POLLING_TIMEOUT)
		if code == scard.ErrCancelled {
			return
		} else if code != nil {
			continue
		}

		// One or more readers changed their status
		for _, readerState := range readerStates {

			if readerState.CurrentState == readerState.EventState {
				// This reader has not changed.
				continue
			}

			profile := knownReaders[readerState.Reader]

			if is(readerState.EventState, scard.StateChanged) {
				// Event state becomes current state
				readerState.CurrentState = readerState.EventState
				// Keep track of last known state for next refresh cycle
				knownReaders[readerState.Reader] = profile.withState(readerState.CurrentState)
			} else {
				continue
			}

			if !is(readerState.CurrentState, scard.StatePresent) {
				// This reader has no card.
				knownReaders[readerState.Reader] = makeEmptyProfile()
				continue
			}

			// Connect to the card
			card, err := scard_ctx.Connect(readerState.Reader, scard.ShareShared, scard.ProtocolAny)
			if err != nil {
				log.WithError(err).Error("Error connecting to card.")
				knownReaders[readerState.Reader] = makeEmptyProfile()
				continue
			}

			// Turn off buzzer for the lifetime of the connection to the reader. Most
			// drivers don't allow transmission of commands without a card present, so
			// this will silence all but the first buzz during the connection's
			// lifetime.
			_, err = card.Transmit(NO_BUZZ_APDU)
			if err != nil {
				log.WithError(err).Debug("Failed while transmitting silencer APDU.")
				continue
			}

			// Request UID
			response, err := card.Transmit(UID_APDU)
			if err != nil {
				log.WithError(err).Debug("Failed while transmitting UID APDU.")
				continue
			}
			uid := ""
			for i := 0; i < len(response)-2; i++ {
				uid += fmt.Sprintf("%X", response[i])
			}
			if len(uid) > 0 && (profile.lastKnownToken == nil || *profile.lastKnownToken != uid) {
				log.Info("Detected RFID token.")
				knownReaders[readerState.Reader] = profile.withToken(&uid)
				onToken(uid)
			}

			card.Disconnect(scard.UnpowerCard)
		}
	}
}

type ReaderProfile struct {
	// Reuse last known state when querying for state changes.
	lastKnownState scard.StateFlag
	// PC/SC implementation on Windows can emit multiple distinct states for
	// a single touch-on. We store detected card IDs to deduplicate token stream
	// for subscribers.
	lastKnownToken *string
}

func makeEmptyProfile() ReaderProfile {
	return ReaderProfile{lastKnownState: scard.StateUnknown, lastKnownToken: nil}
}

func (profile ReaderProfile) withState(flag scard.StateFlag) ReaderProfile {
	return ReaderProfile{lastKnownState: flag, lastKnownToken: profile.lastKnownToken}
}

func (profile ReaderProfile) withToken(token *string) ReaderProfile {
	return ReaderProfile{lastKnownState: profile.lastKnownState, lastKnownToken: token}
}

// Helpers

func is(mask scard.StateFlag, flag scard.StateFlag) bool {
	return mask&flag != 0
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

func normalizeReaderList(readers []string) []string {
	if readers == nil {
		return []string{}
	} else {
		return readers
	}
}
