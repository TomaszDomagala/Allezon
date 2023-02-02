// Package tests is a collection of integration tests for Allezon services.
// It uses docker containers to set up the environment for the tests.
package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	idgetterImageName = "tomaszdomagala/allezon-idgetter"
	idgetterImageTag  = "latest"
)

func findProjectPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %w", err)
	}

	projectName := "Allezon"
	projectNameIndex := strings.LastIndex(wd, projectName)
	if projectNameIndex == -1 {
		return "", fmt.Errorf("could not find project name in working directory: %s", wd)
	}
	return wd[:projectNameIndex+len(projectName)], nil
}

func srcPath() (string, error) {
	projectPath, err := findProjectPath()
	if err != nil {
		return "", fmt.Errorf("could not get project path: %w", err)
	}
	return path.Join(projectPath, "src"), nil
}

// buildIDGetterImage builds the id getter image by running the make target.
// It returns stdout, stderr and an error.
func buildIDGetterImage() (string, string, error) {
	var stdout, stderr strings.Builder
	var err error

	cmd := exec.Command("make", "docker-build-idgetter")
	cmd.Dir, err = srcPath()
	if err != nil {
		return "", "", fmt.Errorf("could not get src path: %w", err)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("could not build id getter image: %w", err)
	}
	return stdout.String(), stderr.String(), nil
}
