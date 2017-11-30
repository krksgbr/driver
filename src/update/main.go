package update

import (
	"crypto"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"runtime"
	"strings"
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

	latestRelease, err := GetLatestReleaseInfo(log, channel, true)
	if err != nil {
		return false, err
	}
	log = log.WithField("newVersion", latestRelease)

	latestSemVersion, err := semver.NewVersion(latestRelease)
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
func GetLatestReleaseInfo(log *logrus.Entry, channel string, checkSignature bool) (string, error) {
	url := latestJSONURL(channel)

	log.Debug("Downloading latest")
	latestResp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	latestReleasePayload, err := ioutil.ReadAll(latestResp.Body)
	if err != nil {
		return "", err
	}

	if checkSignature {
		log.Debug("Downloading latest.sig")
		sigResp, err := http.Get(url + ".sig")
		if err != nil {
			return "", err
		}

		sigPayload, err := ioutil.ReadAll(sigResp.Body)
		if err != nil {
			return "", err
		}

		sig, err := base64.StdEncoding.DecodeString(string(sigPayload))
		if err != nil {
			return "", err
		}

		log.Debug("Verifying latest.sig")
		// reusing go-update sig check features
		opts := update.Options{
			Signature: sig,
			Hash:      crypto.SHA256,
			Verifier:  update.NewECDSAVerifier(),
		}
		err = opts.SetPublicKeyPEM(rawPublicKey)
		if err != nil {
			return "", err
		}

		err = verifySignature(opts, latestReleasePayload)
		if err != nil {
			return "", err
		}
	}

	return strings.TrimSpace(string(latestReleasePayload)), nil
}

func downloadAndUpdate(log *logrus.Entry, channel string, latestRelease string) error {
	log.Info("Downloading update.")

	var versionPath = releaseServer + path.Join(channel, latestRelease, runtime.GOOS)
	var filename = "dividat-driver-" + runtime.GOOS + "-" + runtime.GOARCH + "-" + latestRelease
	var binURL = versionPath + "/" + filename
	var sigURL = binURL + ".sig"
	var err error

	log.Debug("Downloading new release from " + binURL)
	binResp, err := http.Get(binURL)
	if err != nil {
		return err
	}

	log.Debug("Downloading sig file from " + sigURL)
	sigResp, err := http.Get(sigURL)
	if err != nil {
		return err
	}

	log.Debug("Extracting signature")
	sigPayload, _ := ioutil.ReadAll(sigResp.Body)
	if err != nil {
		return err
	}

	log.Debug("Decoding signature from base64")
	signature, err := base64.StdEncoding.DecodeString(string(sigPayload))
	if err != nil {
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
	return releaseServer + channel + "/latest"
}

// taken from https://github.com/inconshreveable/go-update/blob/master/apply.go#L307-L322
func verifySignature(o update.Options, updated []byte) error {
	checksum, err := checksumFor(o.Hash, updated)
	if err != nil {
		return err
	}
	return o.Verifier.VerifySignature(checksum, o.Signature, o.Hash, o.PublicKey)
}

func checksumFor(h crypto.Hash, payload []byte) ([]byte, error) {
	if !h.Available() {
		return nil, errors.New("requested hash function not available")
	}
	hash := h.New()
	hash.Write(payload) // guaranteed not to error
	return hash.Sum([]byte{}), nil
}
