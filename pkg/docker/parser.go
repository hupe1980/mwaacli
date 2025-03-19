package docker

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Compose represents the structure of a docker-compose.yml file
type Compose struct {
	Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig represents a service inside docker-compose.yml
type ServiceConfig struct {
	Image       string   `yaml:"image"`
	Environment []string `yaml:"environment"`
}

// ParseDockerCompose reads and parses the docker-compose.yml file
func ParseDockerCompose(filePath string) (*Compose, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var compose Compose
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &compose, nil
}

func (c *Compose) GetServiceImage(serviceName string) (string, error) {
	service, ok := c.Services[serviceName]
	if !ok {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	return service.Image, nil
}

func (c *Compose) GetServiceEnvironment(serviceName string) ([]string, error) {
	service, ok := c.Services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	return service.Environment, nil
}
