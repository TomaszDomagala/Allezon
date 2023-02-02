package containerutils

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
)

// BuildApiImage builds api image by running make target.
func BuildApiImage() error {
	return runMakeTarget("docker-build-api")
}

// BuildWorkerImage builds worker image by running make target.
func BuildWorkerImage() error {
	return runMakeTarget("docker-build-worker")
}

// BuildIDGetterImage builds idgetter image by running make target.
func BuildIDGetterImage() error {
	return runMakeTarget("docker-build-idgetter")
}

// runMakeTarget runs make target in src directory.
func runMakeTarget(target string) error {
	var stdout, stderr strings.Builder
	var err error

	cmd := exec.Command("make", target)
	cmd.Dir, err = srcPath()
	if err != nil {
		return fmt.Errorf("could not get src path: %w", err)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("could not run make target: %w\n\nstdout: %s\n\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return nil
}

// waitForService calls the /health endpoint of the service and waits until it returns 200 or times out.
// Created for api, worker and idgetter services, as they all have /health endpoint.
func waitForService(environment *container.Environment, service *container.Service) error {
	hostport := service.ExposedHostPort()
	if hostport == "" {
		return fmt.Errorf("failed to get host port for %s", service.Name)
	}
	healthURL := fmt.Sprintf("http://%s/health", hostport)

	// Wait for the service to be ready.
	environment.Logger.Info("waiting for service to start", zap.String("service", service.Name))
	err := environment.Pool.Retry(func() error {
		environment.Logger.Debug("checking if service is ready at", zap.String("url", healthURL), zap.String("service", service.Name))
		_, err := http.Get(healthURL)
		if err != nil {
			return fmt.Errorf("failed to get health: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for service: %w", err)
	}
	environment.Logger.Info("service started", zap.String("service", service.Name))
	return nil
}

func srcPath() (string, error) {
	projectPath, err := findProjectPath()
	if err != nil {
		return "", fmt.Errorf("could not get project path: %w", err)
	}
	return path.Join(projectPath, "src"), nil
}

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
