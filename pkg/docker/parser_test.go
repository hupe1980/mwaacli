package docker

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDockerCompose(t *testing.T) {
	// Create a temporary docker-compose.yml file for testing
	tempFile, err := os.CreateTemp("", "docker-compose-*.yml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	content := `
services:
  web:
    image: nginx:latest
    environment:
      - ENV=production
      - DEBUG=false
  db:
    image: postgres:13
    environment:
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=secret
`
	_, err = tempFile.WriteString(content)
	assert.NoError(t, err)
	assert.NoError(t, tempFile.Close())

	// Test ParseDockerCompose
	compose, err := ParseDockerCompose(tempFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, compose)
	assert.Contains(t, compose.Services, "web")
	assert.Contains(t, compose.Services, "db")

	// Validate the web service
	webService := compose.Services["web"]
	assert.Equal(t, "nginx:latest", webService.Image)
	assert.ElementsMatch(t, []string{"ENV=production", "DEBUG=false"}, webService.Environment)

	// Validate the db service
	dbService := compose.Services["db"]
	assert.Equal(t, "postgres:13", dbService.Image)
	assert.ElementsMatch(t, []string{"POSTGRES_USER=admin", "POSTGRES_PASSWORD=secret"}, dbService.Environment)
}

func TestParseDockerComposeFromReader(t *testing.T) {
	// Define the content of the docker-compose.yml as a string
	content := `
services:
  web:
    image: nginx:latest
    environment:
      - ENV=production
      - DEBUG=false
  db:
    image: postgres:13
    environment:
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=secret
`

	// Use a strings.Reader to simulate an io.Reader
	reader := strings.NewReader(content)

	// Test ParseDockerComposeFromReader
	compose, err := ParseDockerComposeFromReader(reader)
	assert.NoError(t, err)
	assert.NotNil(t, compose)
	assert.Contains(t, compose.Services, "web")
	assert.Contains(t, compose.Services, "db")

	// Validate the web service
	webService := compose.Services["web"]
	assert.Equal(t, "nginx:latest", webService.Image)
	assert.ElementsMatch(t, []string{"ENV=production", "DEBUG=false"}, webService.Environment)

	// Validate the db service
	dbService := compose.Services["db"]
	assert.Equal(t, "postgres:13", dbService.Image)
	assert.ElementsMatch(t, []string{"POSTGRES_USER=admin", "POSTGRES_PASSWORD=secret"}, dbService.Environment)
}

func TestGetServiceImage(t *testing.T) {
	compose := &Compose{
		Services: map[string]ServiceConfig{
			"web": {Image: "nginx:latest"},
			"db":  {Image: "postgres:13"},
		},
	}

	// Test valid service
	image, err := compose.GetServiceImage("web")
	assert.NoError(t, err)
	assert.Equal(t, "nginx:latest", image)

	// Test invalid service
	image, err = compose.GetServiceImage("unknown")
	assert.Error(t, err)
	assert.Equal(t, "", image)
	assert.EqualError(t, err, "service unknown not found")
}

func TestGetServiceEnvironment(t *testing.T) {
	compose := &Compose{
		Services: map[string]ServiceConfig{
			"web": {Environment: []string{"ENV=production", "DEBUG=false"}},
			"db":  {Environment: []string{"POSTGRES_USER=admin", "POSTGRES_PASSWORD=secret"}},
		},
	}

	// Test valid service
	env, err := compose.GetServiceEnvironment("db")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"POSTGRES_USER=admin", "POSTGRES_PASSWORD=secret"}, env)

	// Test invalid service
	env, err = compose.GetServiceEnvironment("unknown")
	assert.Error(t, err)
	assert.Nil(t, env)
	assert.EqualError(t, err, "service unknown not found")
}
