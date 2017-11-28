package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/coreos/go-semver/semver"

	"update"
)

// NPMPackage represent `package.json`
type NPMPackage struct {
	Version string
}

func main() {
	channel := flag.String("channel", "", "Release channel")
	version := flag.String("version", "", "New version to be released")

	flag.Parse()

	semVersion, err := semver.NewVersion(*version)
	if err != nil {
		exit("Not a valid semantic version: " + err.Error())
	}

	rawGitTag, err := exec.Command("bash", "-c", "git describe --exact-match HEAD").Output()
	if err != nil {
		exit("Unable to read HEAD Git tag: " + err.Error())
	}
	gitTag := string(rawGitTag)

	if *version != gitTag {
		exit("Version (" + *version + ") and annotated Git tag of HEAD (" + gitTag + ") must match")
	}

	npmPackage := new(NPMPackage)
	rawNpmPackage, _ := ioutil.ReadFile("package.json")
	if err = json.Unmarshal(rawNpmPackage, npmPackage); err != nil {
		exit("Unable to read and parse `package.json`")
	}
	if npmPackage.Version != *version {
		exit("Version (" + *version + ") and `package.json` version (" + npmPackage.Version + ") must match")
	}

	latestRelease, err := update.GetLatestReleaseInfo(*channel)
	if err != nil {
		exit("Unable to fetch latest version: " + err.Error())
	}

	latestSemVersion, err := semver.NewVersion(latestRelease.Version)
	if err != nil {
		exit("Unable to parse latest version (" + latestRelease.Version + ") as semantic: " + err.Error())
	}

	if !latestSemVersion.LessThan(*semVersion) {
		exit("New version must be higher than latest version")
	}

	os.Exit(0)
}

func exit(message string) {
	os.Stderr.WriteString(message)
	os.Exit(1)
}
