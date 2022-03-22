// Use with GitHub CLI installed. GitHub CLI should be configured
// and the runner/machine should have GITHUB environmental variables
// to access the desired repository. GitHub Actions passes this info
// to Docker images at run-time.
// Use alpine:git with GitHub
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// JSON struct for importing release data
type Release struct {
	Body    string `json:"body"`
	Target  string `json:"targetCommitish"`
	IsDraft bool   `json:"isDraft"`
}

func main() {

	log.Print()

	repo := os.Getenv("GITHUB_REPOSITORY")
	daysStr := os.Getenv("RELEASE_DAYS")
	excludeLabelsStr := os.Getenv("EXCLUDE_LABELS")
	allLabels := os.Getenv("ALL_LABELS")
	dryRun := os.Getenv("DRY_RUN")

	if len(repo) == 0 {
		log.Fatal("Blank repo URL provided, exiting")
	}

	// Default minimum release age: 7 days
	if len(daysStr) == 0 {
		daysStr = "7"
	}

	// Default labels that stop auto-releasing
	if len(excludeLabelsStr) == 0 {
		excludeLabelsStr = "Documentation,Features,Bug Fixes"
	}

	// build the GitHub repo URL
	repoUrl := fmt.Sprintf("https://github.com/%s.git", repo)

	days, err := strconv.Atoi(daysStr)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Repository: %s", repo)
	log.Printf("Minimum draft age to auto-release: %s", daysStr)

	var excludeLabels []string

	// If the release body contains any of these headings, auto-releasing will be disabled
	if len(allLabels) == 0 {
		excludeLabels = strings.Split(excludeLabelsStr, ",")
		log.Printf("Auto-release blocking categories: %s", strings.Join(excludeLabels, ", "))
	} else {
		log.Printf("Auto-releasing with all release categories")
	}

	log.Print()

	// Get current releases
	log.Printf("Repository: %s", repo)
	log.Print("Getting releases for repository...")
	versions, err := exec.Command(
		"gh", "release", "-R", repoUrl, "list",
	).Output()

	if err != nil {
		log.Fatal(err)
	}

	// Parse release data (\t CSV format)
	versionsCsv := csv.NewReader(strings.NewReader(string(versions)))
	versionsCsv.Comma = '\t'
	versionsList, err := versionsCsv.ReadAll()

	if err != nil {
		log.Fatal(err)
	}

	// Check that at least 1 release exists
	if len(versionsList) == 0 {
		log.Print("No releases detected, exiting")
	}

	log.Print("Found releases for repository")

	// Pull the latest version (top row)
	latestVersion := versionsList[0]

	// Define release status, tag and date
	status := latestVersion[1]
	ver := latestVersion[0]
	rawDate := latestVersion[3]

	log.Print("Most recent release:")
	log.Printf("- version: %s", ver)
	log.Printf("- date: %s", rawDate)
	log.Print()

	// Check to see the latest release is a draft
	if status != "Draft" {
		log.Printf("Latest release (%s) is not a pending draft, exiting", ver)
		os.Exit(0)
	}

	tagDate, err := time.Parse("2006-01-02T15:04:05Z", rawDate)
	if err != nil {
		log.Fatal(err)
	}

	// Check release date is over a week ago
	weekAgoDate := time.Now().AddDate(0, 0, -days)
	if tagDate.After(weekAgoDate) {
		log.Printf("Release not %s days old yet: %s. Exiting", daysStr, rawDate)
		os.Exit(0)
	}

	log.Print()

	// Pull the draft release body in JSON format and unmarshal
	log.Printf("Pulling data from draft release %s:", ver)
	tagData, err := exec.Command(
		"gh", "release", "-R", repoUrl, "view", ver, "--json", "body,targetCommitish,isDraft",
	).Output()

	if err != nil {
		log.Fatal(err)
	}

	var release Release
	json.Unmarshal(tagData, &release)

	if !release.IsDraft {
		log.Panic("Release not draft, exiting")
	}

	log.Print()
	log.Printf("Release target: %s", release.Target)
	log.Print("Release body: |")

	// Print release body to stdout
	for _, line := range strings.Split(release.Body, "\n") {
		log.Printf("\t%s", line)
	}

	// Check release contains dependency upgrades
	if !strings.Contains(release.Body, " Dependency Upgrades") {
		log.Println("Draft release does not contain formatted \"Dependency Upgrades\", exiting")
		os.Exit(0)
	}

	log.Print("Dependency upgrades found!")

	// Check for existence of exclusion labels
	log.Print("Looking for labels other than dependency updates")
	suitable := true
	for _, label := range excludeLabels {
		if strings.Contains(release.Body, label) {
			log.Printf("Found label:%s, disabling auto-release", label)
			suitable = false
		}
	}

	log.Print()

	if len(dryRun) > 0 {

		log.Print("DRY_RUN environmental variable set")

	} else if !suitable {

		log.Print("Draft release is not only dependency changes, notifications being sent instead")

	} else {

		log.Print("No other types of release found")
		log.Print("Draft release valid for auto-releasing. Releasing...")

		// gh cli does not support updating an existing release from draft to released
		// See issue https://github.com/cli/cli/issues/1997
		// GitHub API requires application etc.
		// Instead, use gh cli to pull all the draft release data and create a temp draft.
		// If that temp draft creates, delete the original then use gh cli to recreate the
		// release without the draft status.
		// Ideally use cm-cicd or github-actions token

		// First create a VER-temp draft release
		log.Print("Creating temporary draft release")
		tempDraft := fmt.Sprintf("%s-temp", ver)
		copyLog, err := exec.Command(
			"gh", "release", "create", tempDraft, "-d", "-R", repoUrl, "-t", tempDraft, "-n", release.Body, "--target", release.Target,
		).Output()

		for _, line := range strings.Split(string(copyLog), "\n") {
			log.Print(line)
		}

		if err != nil {
			log.Panic(err)
		}

		// Next delete the original draft
		log.Print("Deleting original draft release")
		deleteLog, err := exec.Command(
			"gh", "release", "delete", ver, "-R", repoUrl, "--yes",
		).Output()

		for _, line := range strings.Split(string(deleteLog), "\n") {
			log.Print(line)
		}

		if err != nil {
			log.Panic(err)
		}

		// Next recreate the release without draft status
		log.Print("Recreating release without draft status")
		releaseLog, err := exec.Command(
			"gh", "release", "create", ver, "-R", repoUrl, "-t", ver, "-n", release.Body, "--target", release.Target,
		).Output()

		for _, line := range strings.Split(string(releaseLog), "\n") {
			log.Print(line)
		}

		if err != nil {
			log.Panic(err)
		}

		// Finally delete the temp release
		log.Print("Waiting 5 seconds")
		time.Sleep(5 * time.Second)
		log.Print("Deleting temporary draft release")
		deleteTempLog, err := exec.Command(
			"gh", "release", "delete", tempDraft, "-R", repoUrl, "--yes",
		).Output()

		for _, line := range strings.Split(string(deleteTempLog), "\n") {
			log.Print(line)
		}

		if err != nil {
			log.Panic(err)
		}
	}

}
