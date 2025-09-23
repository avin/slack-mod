//go:build darwin

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func getSlackPath() (string, error) {
	home, _ := os.UserHomeDir()
	candidates := []string{
		"/Applications/Slack.app/Contents/MacOS/Slack",
		filepath.Join(home, "Applications", "Slack.app", "Contents", "MacOS", "Slack"),
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	if path, err := exec.LookPath("Slack"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("slack"); err == nil {
		return path, nil
	}

	return "", errors.New("unable to locate Slack installation")
}

func launchSlack(port int) error {
	slackPath, err := getSlackPath()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		slackPath,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--remote-allow-origins=*",
		"--startup",
	)

	return cmd.Start()
}
