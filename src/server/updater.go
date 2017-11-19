package server

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/inconshreveable/go-update"
	"github.com/sirupsen/logrus"
)

const releaseServer = "http://dist-test.dividat.ch.s3.amazonaws.com/releases/driver/"

const updateInterval = 60 * time.Second

// LatestRelease metadata
type LatestRelease struct {
	Version string
	Commit  string
}

// Update diver: watch for a new version, then download and swap binary
func startUpdateLoop(log *logrus.Entry, channel string, version string) {
	updateTicker := time.NewTicker(updateInterval)
	for {
		select {
		case <-updateTicker.C:
			downloadAndApply(log.WithField("package", "server"), channel, version)
		}
	}
}

func downloadAndApply(log *logrus.Entry, channel string, version string) {
	log.Info("Checking if udpate is needed...")

	var latestURL = releaseServer + channel + "/latest.json"
	log.Debug("Downloading latest info from " + latestURL)
	latestResp, err := http.Get(latestURL)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debug("Unmarshalling latest release data")
	latestRelease := new(LatestRelease)
	latestReleasePayload, _ := ioutil.ReadAll(latestResp.Body)
	if err = json.Unmarshal(latestReleasePayload, &latestRelease); err != nil {
		log.Error(err)
		return
	}

	if latestRelease.Version != version {
		log.Info("Current version (" + version + ") differs from latest version (" + latestRelease.Version + "), downloading update.")

		var filename = "dividat-driver-" + runtime.GOOS + "-" + runtime.GOARCH + "-" + latestRelease.Version
		var versionURL = releaseServer + path.Join(channel, latestRelease.Version, runtime.GOOS, filename)
		var shaURL = versionURL + ".sha256"

		log.Debug("Downloading new release from " + versionURL)
		var binResp *http.Response
		if binResp, err = http.Get(versionURL); err != nil {
			log.Error(err)
			return
		}

		log.Debug("Downloading checksum file from " + shaURL)
		var shaResp *http.Response
		if shaResp, err = http.Get(shaURL); err != nil {
			log.Error(err)
			return
		}

		log.Debug("Reading expected checksum")
		var expectedChecksum []byte
		if expectedChecksum, err = readExpectedChecksum(shaResp, filename); err != nil {
			log.Error(err)
			return
		}

		log.Debug("Applying downloaded update")
		err = update.Apply(binResp.Body, update.Options{
			Checksum: expectedChecksum,
		})
		if err != nil {
			log.Error(err)
			return
		}

		log.Info("Update done.")
	} else {
		log.Info("Current version (" + version + ") is latest.")
	}
}

func readExpectedChecksum(shaResp *http.Response, filename string) ([]byte, error) {
	scanner := bufio.NewScanner(shaResp.Body)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "  ")
		if len(parts) == 2 && parts[1] == filename {
			return hex.DecodeString(parts[0])
		}
	}
	return nil, errors.New("Checksum not found for file " + filename)
}
