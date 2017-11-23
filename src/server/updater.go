package server

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"runtime"
	"time"

	"github.com/inconshreveable/go-update"
	"github.com/sirupsen/logrus"
)

const releaseServer = "http://dist-test.dividat.ch.s3.amazonaws.com/releases/driver/"

const updateInterval = 60 * time.Second

var rawPublicKey = []byte(`
----BEGIN PUBLIC KEY-----
MFYwEAYHKoZIzj0CAQYFK4EEAAoDQgAElNVCWtYI8/Ehe0qz0Mx7YKhUNZnkp45R
6aLbwop7e3H2DNSeG523WUFxNMqd36heSswFUp5RB8evaka6eto4MA==
-----END PUBLIC KEY-----
`)

// LatestRelease metadata
type LatestRelease struct {
	Version string
	Commit  string
}

// BinMetadata for version verification
type BinMetadata struct {
	Checksum  string
	Signature string
}

var updateTicker *time.Ticker
var updating = false

// Update driver: watch for a new version, then download and swap binary
func startUpdateLoop(log *logrus.Entry, channel string, version string) {
	updateTicker = time.NewTicker(updateInterval)
	for {
		select {
		case <-updateTicker.C:
			updating = true
			updated, err := doUpdateLoop(log, channel, version)
			if err != nil {
				log.Error(err)
			}
			if updated {
				updateTicker.Stop()
			}
			updating = false
		}
	}
}

func doUpdateLoop(log *logrus.Entry, channel string, version string) (bool, error) {
	log.Info("Checking if udpate is needed...")

	latestRelease, err := getLatestReleaseInfo(log, channel)
	if err != nil {
		return false, err
	}

	if latestRelease.Version != version {
		log.Info("Current version (" + version + ") differs from latest version (" + latestRelease.Version + "), downloading update.")

		err = downloadAndUpdate(log, latestRelease)
		if err != nil {
			return false, err
		}

		log.Info("Update done, ticker stopped, waiting for the restart.")
		return true, nil
	}

	log.Info("Current version (" + version + ") is latest.")
	return false, nil
}

func getLatestReleaseInfo(log *logrus.Entry, channel string) (*LatestRelease, error) {
	var latestURL = releaseServer + channel + "/latest.json"
	log.Debug("Downloading latest info from " + latestURL)
	latestResp, err := http.Get(latestURL)
	if err != nil {
		return nil, err
	}

	log.Debug("Unmarshalling latest release data")
	latestRelease := new(LatestRelease)
	latestReleasePayload, _ := ioutil.ReadAll(latestResp.Body)
	if err = json.Unmarshal(latestReleasePayload, &latestRelease); err != nil {
		return nil, err
	}
	return latestRelease, nil
}

func downloadAndUpdate(log *logrus.Entry, latestRelease *LatestRelease) error {
	var versionPath = releaseServer + path.Join(channel, latestRelease.Version, runtime.GOOS)
	var filename = "dividat-driver-" + runtime.GOOS + "-" + runtime.GOARCH + "-" + latestRelease.Version
	var binURL = versionPath + "/" + filename
	var metadataURL = versionPath + "/" + "metadata.json"
	var err error

	log.Debug("Downloading new release from " + binURL)
	var binResp *http.Response
	if binResp, err = http.Get(binURL); err != nil {
		return err
	}

	log.Debug("Downloading metadata file from " + metadataURL)
	var metadataResp *http.Response
	if metadataResp, err = http.Get(metadataURL); err != nil {
		return err
	}

	log.Debug("Extracting metadata fields")
	metadata := new(BinMetadata)
	metadataPayload, _ := ioutil.ReadAll(metadataResp.Body)
	if err = json.Unmarshal(metadataPayload, &metadata); err != nil {
		return err
	}

	var signature []byte
	if signature, err = hex.DecodeString(metadata.Signature); err != nil {
		return err
	}

	log.Debug("Applying downloaded update")
	err = update.Apply(binResp.Body, update.Options{
		Signature: signature,
		PublicKey: rawPublicKey,
	})
	if err != nil {
		return err
	}

	return nil
}
