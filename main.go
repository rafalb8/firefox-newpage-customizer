package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// browser.newtabpage.activity-stream.topSitesCount - number of top sites to show

// CSS
// toolkit.legacyUserProfileCustomizations.stylesheets=true 
// chrome/userContent.css - custom css => @-moz-document url(about:newtab) {}

// changes in user.js are applied every time the browser is restarted
// changes in prefs.js should not be changed while the browser is running, because browser will change them

const CONFIG_PATH = ".mozilla/firefox"
const PINNED_CFG = `user_pref("browser.newtabpage.pinned", `

type Pin struct {
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
	Icon  string `json:"customScreenshotURL,omitempty"`
}

type Pinned []Pin

// check if firefox process is running
func isRunning() bool {
	pid := fmt.Sprint(os.Getpid())
	matches, _ := filepath.Glob("/proc/*/exe")
	for _, match := range matches {
		if strings.Contains(match, pid) || strings.Contains(match, "self") {
			continue
		}

		target, _ := os.Readlink(match)
		if strings.Contains(target, "firefox") {
			return true
		}
	}
	return false
}

func getDefaultProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, CONFIG_PATH)

	file, err := ini.Load(filepath.Join(path, "profiles.ini"))
	if err != nil {
		return "", err
	}

	return filepath.Join(path, file.Section("Profile0").Key("Path").String()), nil
}

func (p *Pinned) Load(profile string) error {
	// load prefs.js
	data, err := os.ReadFile(filepath.Join(profile, "prefs.js"))
	if err != nil {
		return err
	}

	// find the pinned sites
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, PINNED_CFG) {
			pinnedJSON := strings.ReplaceAll(
				strings.Trim(line[len(PINNED_CFG):], `");`),
				`\`, "",
			)

			err := json.Unmarshal([]byte(pinnedJSON), p)
			return err
		}
	}

	return errors.New("No pinned sites found")
}

func (p *Pinned) Save(profile string) error {
	file := filepath.Join(profile, "prefs.js")
	// load prefs.js
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	json, err := json.Marshal(p)
	if err != nil {
		return err
	}

	// cfg line
	cfg := fmt.Sprintf(`%s"%s");`, PINNED_CFG, strings.ReplaceAll(string(json), `"`, `\"`))

	// replace cfg line
	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, PINNED_CFG) {
			lines[i] = cfg
			replaced = true
			break
		}
	}

	if !replaced {
		lines = append(lines, cfg)
	}

	if isRunning() {
		return errors.New("Firefox is running, please close it")
	}

	return os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644)
}

func main() {
	profile, err := getDefaultProfile()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pinned := Pinned{}

	err = pinned.Load(profile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Pinned sites:", pinned)
	fmt.Println(pinned.Save(profile))
}
