package update

import (
	"crypto"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"runtime"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/inconshreveable/go-update"
	"github.com/sirupsen/logrus"
)

const releaseServer = "http://dist-test.dividat.ch.s3.amazonaws.com/releases/driver/"

const updateInterval = 60 * time.Second

var rawPublicKey = []byte(`
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE8sv+i3PuPlTcB3pPMgO87dtOq/ko
2JsEBT+baM7jI+PWkFqpxnoziWF9SL0FU8euKNpxkowztWrmAqXgLZ5NPg==
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

// Start watch for a new version, then download and swap binary
func Start(baseLog *logrus.Entry, version string, channel string) {
	log := baseLog.WithFields(logrus.Fields{
		"version":   version,
		"channel":   channel,
		"latestURL": latestJSONURL(channel),
	})
	updateTicker = time.NewTicker(updateInterval)
	loop := func() {
		if updating {
			return
		}
		updating = true
		updated, err := doUpdateLoop(log, version, channel)
		if err != nil {
			log.Error(err)
		}
		if updated {
			updateTicker.Stop()
		}
		updating = false
	}

	loop()

	for {
		select {
		case <-updateTicker.C:
			loop()
		}
	}
}

func doUpdateLoop(log *logrus.Entry, version string, channel string) (bool, error) {
	log.Info("Checking if udpate is needed...")

	latestRelease, err := GetLatestReleaseInfo(channel)
	if err != nil {
		return false, err
	}
	log = log.WithField("newVersion", latestRelease.Version)

	latestSemVersion, err := semver.NewVersion(latestRelease.Version)
	if err != nil {
		return false, err
	}

	currentSemVersion, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}

	if currentSemVersion.LessThan(*latestSemVersion) {
		log.Info("Newer version discovered.")
		err = downloadAndUpdate(log, channel, latestRelease)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	log.Info("Current version is latest.")
	return false, nil
}

// GetLatestReleaseInfo download and parse JSON info for latest version from repository
func GetLatestReleaseInfo(channel string) (*LatestRelease, error) {
	latestResp, err := http.Get(latestJSONURL(channel))
	if err != nil {
		return nil, err
	}

	latestRelease := new(LatestRelease)
	latestReleasePayload, _ := ioutil.ReadAll(latestResp.Body)
	if err = json.Unmarshal(latestReleasePayload, &latestRelease); err != nil {
		return nil, err
	}
	return latestRelease, nil
}

func downloadAndUpdate(log *logrus.Entry, channel string, latestRelease *LatestRelease) error {
	log.Info("Downloading update.")

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

	log.Debug("Building update options")
	opts := update.Options{
		Signature: signature,
		Hash:      crypto.SHA256,
		Verifier:  update.NewECDSAVerifier(),
	}
	log.Debug("Setting public key")
	err = opts.SetPublicKeyPEM(rawPublicKey)
	if err != nil {
		return err
	}
	log.Debug("Applying update")
	err = update.Apply(binResp.Body, opts)
	if err != nil {
		return err
	}
	log.Info("Update done.")
	return nil
}

func latestJSONURL(channel string) string {
	return releaseServer + channel + "/latest.json"
}
