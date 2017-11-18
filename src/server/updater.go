package server

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
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
func Update(log *logrus.Entry, channel string, version string) {

	updateTicker := time.NewTicker(updateInterval)

	downloadAndApply(log.WithField("package", "server"), channel, version)
	go func() {
		for {
			select {
			case <-updateTicker.C:
				// downloadAndApply(log.WithField("package", "server"), channel, version)
			}
		}
	}()

}

func downloadAndApply(log *logrus.Entry, channel string, version string) {
	var latestURL = releaseServer + channel + "/latest.json"
	log.Info("Downloading latest info from " + latestURL)
	latestResp, latestErr := http.Get(latestURL)

	if latestErr != nil {
		log.Error(latestErr)
		return
	}

	latestRelease := new(LatestRelease)
	latestReleasePayload, _ := ioutil.ReadAll(latestResp.Body)
	err := json.Unmarshal(latestReleasePayload, &latestRelease)
	if err != nil {
		log.Error(err)
		return
	}

	log.Info("Current version: " + version + " / Next version: " + latestRelease.Version)

	if latestRelease.Version != version {
		var filename = "dividat-driver-" + runtime.GOOS + "-" + runtime.GOARCH + "-" + latestRelease.Version
		var versionURL = releaseServer + channel + "/" + latestRelease.Version + "/" + runtime.GOOS + "/" + filename
		var shaURL = versionURL + ".sha256"

		log.Info("Downloading new release from " + versionURL)
		binResp, binErr := http.Get(versionURL)
		if binErr != nil {
			log.Error(binErr)
			return
		}
		defer binResp.Body.Close()

		shaResp, _ := http.Get(shaURL)

		log.Info("Download successful. Now applying update..")
		expectedChecksum, checksumErr := getExpectedChecksum(shaResp, filename)
		if checksumErr != nil {
			log.Error(checksumErr)
			return

		}
		err = update.Apply(binResp.Body, update.Options{
			Checksum: expectedChecksum,
		})
		if err != nil {
			log.Error(err)
			return
		}

		log.Info("Update done.")
	}
}

func getExpectedChecksum(shaResp *http.Response, filename string) ([]byte, error) {
	scanner := bufio.NewScanner(shaResp.Body)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "  ")
		if len(parts) == 2 && parts[1] == filename {
			return hex.DecodeString(parts[0])
		}
	}
	return nil, errors.New("Checksum not found for file " + filename)
}
