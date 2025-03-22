package util

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
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

// ParseEnvFile opens a .env file and parses its content using ParseEnv.
func ParseEnvFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file %s: %w", filePath, err)
	}
	defer file.Close()

	return ParseEnv(file)
}

// ParseEnv parses .env content from an io.Reader and returns a slice of key=value pairs.
func ParseEnv(reader io.Reader) ([]string, error) {
	var envVars []string

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Ensure the line is in the format KEY=VALUE
		if !strings.Contains(line, "=") {
			return nil, fmt.Errorf("invalid line in env content: %s", line)
		}

		// Split the line into key and value
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle inline comments for unquoted values
		if !strings.HasPrefix(value, `"`) && !strings.HasPrefix(value, `'`) {
			if idx := strings.Index(value, " #"); idx != -1 {
				value = value[:idx]
			}
		}

		// Handle quoted values
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			// Remove double quotes and handle escaped characters
			value = strings.Trim(value, `"`)
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\n`, "\n")
			value = strings.ReplaceAll(value, `\r`, "\r")
		} else if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) {
			// Remove single quotes (no escaping)
			value = strings.Trim(value, `'`)
		}

		// Reconstruct the key=value pair and add to the list
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read env content: %w", err)
	}

	return envVars, nil
}

// MergeEnvVars merges environment variables, resolving duplicate keys by keeping the last occurrence.
func MergeEnvVars(envVars []string, ignoreEmptyValues bool) []string {
	envMap := make(map[string]string)

	// Iterate over the slice and populate the map
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			if ignoreEmptyValues && value == "" {
				continue
			}

			envMap[key] = value // Overwrite the value if the key already exists
		}
	}

	// Convert the map back to a slice
	mergedEnvVars := make([]string, 0, len(envMap))
	for key, value := range envMap {
		mergedEnvVars = append(mergedEnvVars, fmt.Sprintf("%s=%s", key, value))
	}

	return mergedEnvVars
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

// IsPortFree checks if a given port is free on the machine.
func IsPortFree(port string) bool {
	// Try to listen on the port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		// Port is in use
		return false
	}
	// Close the listener to free the port
	defer listener.Close()

	return true
}
