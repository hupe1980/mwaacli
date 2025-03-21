package local

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

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

type StartOptions struct {
	Port    string
	ResetDB bool
	Envs    *Envs
}

func (r *Runner) Start(ctx context.Context, optFns ...func(o *StartOptions)) error {
	opts := StartOptions{
		Port: "8080",
		Envs: nil,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if !util.IsPortFree(opts.Port) {
		return fmt.Errorf("port %s is already in use", opts.Port)
	}

	containers, err := r.client.ListContainersByLabel(ctx, fmt.Sprintf("%s=%s", LabelKey, r.opts.ContainerLabel), false)
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
		LabelKey: r.opts.ContainerLabel,
	}

	if opts.ResetDB {
		dbDataPath := filepath.Join(r.cwd, r.opts.ClonePath, "db-data")
		if err := os.RemoveAll(dbDataPath); err != nil {
			return fmt.Errorf("failed to clear database files: %w", err)
		}

		if err := os.MkdirAll(dbDataPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to recreate database directory: %w", err)
		}
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
		RestartPolicy: container.RestartPolicy{Name: "always"},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(r.cwd, r.opts.ClonePath, "db-data"),
				Target: "/var/lib/postgresql/data",
			},
		},
		LogConfig: logConfig,
	}

	postgresID, err := r.client.RunContainer(ctx, postgresConfig, postgresHostConfig, networkConfig, "postgres")
	if err != nil {
		return fmt.Errorf("failed to create and start Postgres container: %w", err)
	}

	if err := r.client.WaitForContainerReady(ctx, postgresID, 5*60); err != nil {
		return fmt.Errorf("failed to wait for Postgres container: %w", err)
	}

	mwaaEnv, err := r.buildEnvironmentVariables(opts.Envs)
	if err != nil {
		return fmt.Errorf("failed to build environment variables: %w", err)
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
		PortBindings:  nat.PortMap{"8080/tcp": {nat.PortBinding{HostPort: opts.Port}}},
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
	return r.client.StopContainersByLabel(ctx, fmt.Sprintf("%s=%s", LabelKey, r.opts.ContainerLabel))
}

func (r *Runner) WaitForWebserverReady(ctx context.Context, webserverURL string) error {
	const (
		timeout  = 2 * time.Minute // Maximum wait time
		interval = 3 * time.Second // Polling interval
	)

	parsedURL, err := url.ParseRequestURI(webserverURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("unsupported URL scheme, must be http or https")
	}

	// Create an HTTP client with a timeout for individual requests
	client := &http.Client{
		Timeout: 10 * time.Second, // Prevent hanging requests
	}

	// Create a context with timeout for the entire operation
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for application to be ready: %w", ctx.Err())
		case <-ticker.C:
			// Create a new HTTP request with the context
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
			if err != nil {
				return fmt.Errorf("failed to create HTTP request: %w", err)
			}

			// Perform the HTTP request
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close() // Ensure response body is always closed

				if resp.StatusCode == http.StatusOK {
					return nil // Application is ready
				}
			}
		}
	}
}

func (r *Runner) Close() error {
	return r.client.Close()
}
