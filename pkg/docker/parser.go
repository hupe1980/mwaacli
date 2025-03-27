package docker

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Compose represents the structure of a docker-compose.yml file.
type Compose struct {
	Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig represents a service inside docker-compose.yml.
type ServiceConfig struct {
	Image       string   `yaml:"image"`
	Environment []string `yaml:"environment"`
}

// ParseDockerCompose reads and parses a docker-compose.yml file from the given file path.
// It internally uses ParseDockerComposeFromReader to parse the file content.
func ParseDockerCompose(filePath string) (*Compose, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return ParseDockerComposeFromReader(file)
}

// ParseDockerComposeFromReader reads and parses a docker-compose.yml file from an io.Reader.
// This function is useful for testing or when the file content is already in memory.
func ParseDockerComposeFromReader(reader io.Reader) (*Compose, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var compose Compose
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &compose, nil
}

// GetServiceImage retrieves the image name of a specific service from the docker-compose.yml structure.
// Returns an error if the service is not found.
func (c *Compose) GetServiceImage(serviceName string) (string, error) {
	service, ok := c.Services[serviceName]
	if !ok {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	return service.Image, nil
}

// GetServiceEnvironment retrieves the environment variables of a specific service from the docker-compose.yml structure.
// Returns an error if the service is not found.
func (c *Compose) GetServiceEnvironment(serviceName string) ([]string, error) {
	service, ok := c.Services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	return service.Environment, nil
}
