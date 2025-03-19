package util

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// OpenBrowser attempts to open the given URL in the default web browser based on the operating system.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

// EnsurePathIsEmptyOrNonExistent ensures the given path is either empty or does not exist.
func EnsurePathIsEmptyOrNonExistent(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Path does not exist, which is acceptable
		}

		return fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	if len(entries) > 0 {
		return fmt.Errorf("path %s already exists and is not empty", path)
	}

	return nil
}

func ParseEnvFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file %s: %w", filePath, err)
	}
	defer file.Close()

	var envVars []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Ensure the line is in the format KEY=VALUE
		if !strings.Contains(line, "=") {
			return nil, fmt.Errorf("invalid line in env file: %s", line)
		}

		envVars = append(envVars, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	return envVars, nil
}

// ShortContainerID shortens a Docker container ID to the first 12 characters.
func ShortContainerID(containerID string) string {
	if len(containerID) > 12 {
		return containerID[:12]
	}

	return containerID
}

// IsValidARN checks if the provided string is a valid AWS ARN.
func IsValidARN(arn string) error {
	// Regular expression to match the ARN format
	arnRegex := `^arn:(aws|aws-cn|aws-us-gov):[a-zA-Z0-9-]+:[a-z0-9-]*:[0-9]{12}:[^:]+$`

	matched, err := regexp.MatchString(arnRegex, arn)
	if err != nil {
		return err
	}

	if !matched {
		return errors.New("invalid ARN format")
	}

	return nil
}
