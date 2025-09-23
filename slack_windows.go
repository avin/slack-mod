//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func getSlackPath() (string, error) {
	baseDirs := candidateWindowsSlackDirs()
	checked := make(map[string]struct{})

	for _, base := range baseDirs {
		if base == "" {
			continue
		}
		if _, ok := checked[base]; ok {
			continue
		}
		checked[base] = struct{}{}

		if path, err := findSlackInAppDirs(base); err == nil {
			return path, nil
		}

		exePath := filepath.Join(base, "slack.exe")
		if fileExists(exePath) {
			return exePath, nil
		}
	}

	if path, err := exec.LookPath("slack.exe"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("slack"); err == nil {
		return path, nil
	}

	return "", errors.New("unable to locate Slack installation")
}

func candidateWindowsSlackDirs() []string {
	dirs := []string{}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		dirs = append(dirs, filepath.Join(localAppData, "slack"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "AppData", "Local", "slack"))
	}
	if programFiles := os.Getenv("PROGRAMFILES"); programFiles != "" {
		dirs = append(dirs, filepath.Join(programFiles, "Slack"))
	}
	if programFilesX86 := os.Getenv("PROGRAMFILES(X86)"); programFilesX86 != "" {
		dirs = append(dirs, filepath.Join(programFilesX86, "Slack"))
	}
	return dirs
}

func findSlackInAppDirs(baseDir string) (string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", err
	}

	var appDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "app-") {
			appDirs = append(appDirs, entry.Name())
		}
	}

	if len(appDirs) == 0 {
		return "", fmt.Errorf("no app-* directories in %s", baseDir)
	}

	sort.Slice(appDirs, func(i, j int) bool {
		return appDirs[i] > appDirs[j]
	})

	for _, dir := range appDirs {
		candidate := filepath.Join(baseDir, dir, "slack.exe")
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("slack.exe not found in %s", baseDir)
}

func launchSlack(port int) error {
	slackPath, err := getSlackPath()
	if err != nil {
		return err
	}

	args := []string{
		"/c",
		"start",
		"",
		slackPath,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--remote-allow-origins=*",
		"--startup",
	}

	cmd := exec.Command("cmd", args...)
	return cmd.Start()
}
