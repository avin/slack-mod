package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func loadInjectionAsset(name string) ([]byte, error) {
	for _, dir := range injectionDirs() {
		candidate := filepath.Join(dir, "injection", name)
		if fileExists(candidate) {
			return ioutil.ReadFile(candidate)
		}
	}

	return nil, fmt.Errorf("injection asset %s not found", name)
}

func injectionDirs() []string {
	var dirs []string

	if execPath, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(execPath))
	}

	if wd, err := os.Getwd(); err == nil {
		if len(dirs) == 0 || dirs[0] != wd {
			dirs = append(dirs, wd)
		}
	}

	return dirs
}
