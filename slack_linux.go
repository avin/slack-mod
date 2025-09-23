//go:build linux

package main

import (
	"errors"
	"fmt"
	"os/exec"
)

func getSlackPath() (string, error) {
	if path, err := exec.LookPath("slack"); err == nil {
		return path, nil
	}

	candidates := []string{
		"/usr/lib/slack/slack",
		"/usr/local/lib/slack/slack",
		"/opt/slack/slack",
		"/snap/bin/slack",
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate, nil
		}
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
