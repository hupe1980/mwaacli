package local

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hupe1980/mwaacli/pkg/docker"
	"github.com/hupe1980/mwaacli/pkg/util"
)

type RunnerOptions struct {
	ClonePath      string
	NetworkName    string
	DagsPath       string
	ContainerLabel string
	Credentials    *aws.Credentials
}

type Runner struct {
	airflowVersion string
	client         *docker.Client
	cwd            string
	opts           RunnerOptions
}

// NewRunner creates a new MWAA installer
func NewRunner(version string, optFns ...func(o *RunnerOptions)) (*Runner, error) {
	opts := RunnerOptions{
		ClonePath:      DefaultClonePath,
		NetworkName:    fmt.Sprintf("aws-mwaa-local-runner-%s_default", convertVersion(version)),
		DagsPath:       ".",
		ContainerLabel: fmt.Sprintf("aws-mwaa-local-runner-%s", convertVersion(version)),
		Credentials:    nil,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	return &Runner{
		airflowVersion: version,
		client:         client,
		cwd:            cwd,
		opts:           opts,
	}, nil
}

func (r *Runner) BuildImage(ctx context.Context) error {
	buildContextDir := filepath.Join(r.opts.ClonePath, "docker")

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{fmt.Sprintf("amazon/mwaa-local:%s", convertVersion(r.airflowVersion))},
		Dockerfile: "Dockerfile",
	}

	return r.client.BuildImage(ctx, buildContextDir, buildOptions)
}

func (r *Runner) Start(ctx context.Context) error {
	containers, err := r.client.ListContainersByLabel(ctx, fmt.Sprintf("github.com.hupe1980.mwaacli=%s", r.opts.ContainerLabel), false)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) > 0 {
		return fmt.Errorf("airflow local environment is already running")
	}

	dockerComposeLocal, err := docker.ParseDockerCompose(filepath.Join(r.opts.ClonePath, "docker", "docker-compose-local.yml"))
	if err != nil {
		return fmt.Errorf("failed to parse docker-compose-local.yml: %w", err)
	}

	networkID, err := r.client.CreateNetwork(ctx, r.opts.NetworkName)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	logConfig := container.LogConfig{
		Type: "json-file",
		Config: map[string]string{
			"max-size": "10m",
			"max-file": "3",
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			r.opts.NetworkName: {NetworkID: networkID},
		},
	}

	containerLabels := map[string]string{
		"github.com.hupe1980.mwaacli": r.opts.ContainerLabel,
	}

	postgresImage, err := dockerComposeLocal.GetServiceImage("postgres")
	if err != nil {
		return fmt.Errorf("failed to get service image for postgres: %w", err)
	}

	postgresEnv, err := dockerComposeLocal.GetServiceEnvironment("postgres")
	if err != nil {
		return fmt.Errorf("failed to get service environment for postgres: %w", err)
	}

	// Create Postgres container
	postgresConfig := &container.Config{
		Image:  postgresImage,
		Env:    postgresEnv,
		Labels: containerLabels,
	}

	postgresHostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(r.cwd, r.opts.ClonePath, "db-data"),
				Target: "/var/lib/postgresql/data",
			},
		},
		RestartPolicy: container.RestartPolicy{Name: "always"},
		LogConfig:     logConfig,
	}

	postgresID, err := r.client.RunContainer(ctx, postgresConfig, postgresHostConfig, networkConfig, "postgres")
	if err != nil {
		return fmt.Errorf("failed to create and start Postgres container: %w", err)
	}

	if err := r.client.WaitForContainerReady(ctx, postgresID, 5*60); err != nil {
		return fmt.Errorf("failed to wait for Postgres container: %w", err)
	}

	envFilePath := filepath.Join(r.opts.ClonePath, "docker", "config", ".env.localrunner")

	mwaaEnv, err := util.ParseEnvFile(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse env file for Postgres: %w", err)
	}

	mwaaEnv = append(mwaaEnv, "LOAD_EX=n", "EXECUTOR=Local")

	if r.opts.Credentials != nil {
		if r.opts.Credentials.AccessKeyID != "" {
			mwaaEnv = append(mwaaEnv, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", r.opts.Credentials.AccessKeyID))
		}

		if r.opts.Credentials.SecretAccessKey != "" {
			mwaaEnv = append(mwaaEnv, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", r.opts.Credentials.SecretAccessKey))
		}

		if r.opts.Credentials.SessionToken != "" {
			mwaaEnv = append(mwaaEnv, fmt.Sprintf("AWS_SESSION_TOKEN=%s", r.opts.Credentials.SessionToken))
		}
	}

	// Create MWAA Local Runner container
	localRunnerConfig := &container.Config{
		Image:  fmt.Sprintf("amazon/mwaa-local:%s", convertVersion(r.airflowVersion)),
		Env:    mwaaEnv,
		Cmd:    []string{"local-runner"},
		Labels: containerLabels,
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "[ -f /usr/local/airflow/airflow-webserver.pid ]"},
			Interval: 30 * time.Second,
			Timeout:  30 * time.Second,
			Retries:  3,
		},
	}

	localRunnerHostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
		PortBindings:  nat.PortMap{"8080/tcp": {nat.PortBinding{HostPort: "8080"}}},
		Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.DagsPath, "dags"), Target: "/usr/local/airflow/dags"},
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "plugins"), Target: "/usr/local/airflow/plugins"},
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "requirements"), Target: "/usr/local/airflow/requirements"},
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "startup_script"), Target: "/usr/local/airflow/startup"},
		},
		LogConfig: logConfig,
	}

	if _, err := r.client.RunContainer(ctx, localRunnerConfig, localRunnerHostConfig, networkConfig, "local-runner"); err != nil {
		return fmt.Errorf("failed to create and start MWAA Local Runner container: %w", err)
	}

	return nil
}

func (r *Runner) Stop(ctx context.Context) error {
	return r.client.StopContainersByLabel(ctx, fmt.Sprintf("github.com.hupe1980.mwaacli=%s", r.opts.ContainerLabel))
}

func (r *Runner) WaitForAppReady(appURL string) error {
	const (
		timeout  = 2 * time.Minute // Maximum wait time
		interval = 3 * time.Second // Polling interval
	)

	// Validate URL to prevent SSRF attacks
	parsedURL, err := url.ParseRequestURI(appURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("unsupported URL scheme, must be http or https")
	}

	// Create an HTTP client with timeouts
	client := &http.Client{
		Timeout: 10 * time.Second, // Prevent hanging requests
	}

	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return errors.New("timeout waiting for application to be ready")
		}

		resp, err := client.Get(parsedURL.String())
		if err == nil {
			defer resp.Body.Close() // Ensure response body is always closed

			if resp.StatusCode == http.StatusOK {
				return nil // Application is ready
			}
		}

		time.Sleep(interval) // Wait before retrying
	}
}

func (r *Runner) Close() error {
	return r.client.Close()
}

// convertVersion converts a version string like "v2.20.2" to "2_20_2".
func convertVersion(version string) string {
	// Remove the leading "v" if it exists
	version = strings.TrimPrefix(version, "v")

	// Replace dots with underscores
	return strings.ReplaceAll(version, ".", "_")
}
