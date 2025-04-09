// Package docker provides a client for interacting with the Docker daemon.
// It includes functionality for building images, running containers, and streaming logs.
package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/hupe1980/mwaacli/pkg/util"
	"github.com/moby/term"
)

type Client struct {
	client *dockerClient.Client
	logger *log.Logger
}

// NewClient initializes a new Docker client.
func NewClient() (*Client, error) {
	c, err := dockerClient.NewClientWithOpts(
		dockerClient.FromEnv,
		dockerClient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	client := &Client{
		client: c,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		if runtime.GOOS == "darwin" {
			if err := client.useColimaSocket(ctx); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("failed to ping Docker client")
		}
	}

	return client, nil
}

// useColimaSocket attempts to use the Colima Docker socket on macOS.
func (c *Client) useColimaSocket(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	colimaSocket := fmt.Sprintf("unix://%s/.colima/docker.sock", homeDir)

	c.client, err = dockerClient.NewClientWithOpts(
		dockerClient.WithHost(colimaSocket),
		dockerClient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Colima Docker client: %w", err)
	}

	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping Colima Docker client")
	}

	return nil
}

// Ping checks if the Docker daemon is reachable.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.ServerVersion(ctx)
	if err != nil {
		return fmt.Errorf("docker is not installed or not running: %w", err)
	}

	return nil
}

// BuildImage builds a Docker image from the specified context directory.
func (c *Client) BuildImage(ctx context.Context, buildContextDir string, buildOptions types.ImageBuildOptions) error {
	buildCtx, err := archive.TarWithOptions(buildContextDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer buildCtx.Close()

	resp, err := c.client.ImageBuild(ctx, buildCtx, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}
	defer resp.Body.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)

	return jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
}

// RunContainer creates and starts a container, pulling the image if necessary.
func (c *Client) RunContainer(ctx context.Context, containerConfig *container.Config, hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig, containerName string) (string, error) {
	containerID, err := c.ensureContainer(ctx, containerConfig, hostConfig, networkConfig, containerName)
	if err != nil {
		return "", err
	}

	if err := c.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container %s: %w", containerName, err)
	}

	c.logger.Printf("Started container %s with ID %s\n", containerName, ShortContainerID(containerID))

	return containerID, nil
}

// ContainerLogs streams logs from a container.
func (c *Client) ContainerLogs(ctx context.Context, containerID string) error {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	reader, err := c.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Process and pretty-print the logs
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Strip non-printable characters
		cleanLine := util.StripNonPrintable(line)
		fmt.Println(cleanLine)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read container logs: %w", err)
	}

	return nil
}

// AttachToContainer attaches to a running container's input, output, and error streams.
func (c *Client) AttachToContainer(ctx context.Context, containerID string) error {
	// Attach to the container
	resp, err := c.client.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to container %s: %w", containerID, err)
	}
	defer resp.Close()

	// Stream the container's output to the terminal
	go func() {
		if _, err := io.Copy(os.Stdout, resp.Reader); err != nil {
			c.logger.Printf("error streaming container output: %v\n", err)
		}
	}()

	// Wait for the container to finish
	statusCh, errCh := c.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case <-statusCh:
		return nil
	case err := <-errCh:
		return fmt.Errorf("error waiting for container %s: %w", containerID, err)
	}
}

// ensureContainer ensures the container exists, creating it if necessary.
func (c *Client) ensureContainer(ctx context.Context, containerConfig *container.Config, hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig, containerName string) (string, error) {
	existingContainers, err := c.ListContainersByName(ctx, containerName, true)
	if err != nil {
		return "", fmt.Errorf("failed to check existing containers: %w", err)
	}

	if len(existingContainers) > 0 {
		containerID := existingContainers[0].ID

		if err := c.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
			Force: true,
		}); err != nil {
			return "", fmt.Errorf("failed to remove existing container %s: %w", containerName, err)
		}
	}

	if err := c.ensureImage(ctx, containerConfig.Image); err != nil {
		return "", err
	}

	resp, err := c.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container %s: %w", containerName, err)
	}

	c.logger.Printf("Created new container %s with ID %s\n", containerName, ShortContainerID(resp.ID))

	return resp.ID, nil
}

// ensureImage ensures the specified image exists locally, pulling it if necessary.
func (c *Client) ensureImage(ctx context.Context, imageName string) error {
	images, err := c.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				c.logger.Printf("Image %s found locally. Skipping pull.\n", imageName)
				return nil
			}
		}
	}

	c.logger.Printf("Image %s not found locally. Attempting to pull...\n", imageName)

	reader, err := c.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)

	if err := jsonmessage.DisplayJSONMessagesStream(reader, os.Stderr, termFd, isTerm, nil); err != nil {
		return fmt.Errorf("failed to read image pull output: %w", err)
	}

	c.logger.Printf("Successfully pulled image: %s\n", imageName)

	return nil
}

// WaitForContainerReady waits for a container to be ready within a timeout.
func (c *Client) WaitForContainerReady(ctx context.Context, containerID string, timeoutSeconds int) error {
	timeout := time.After(time.Duration(timeoutSeconds) * time.Second)
	ticker := time.NewTicker(1 * time.Second)

	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout reached while waiting for container %s to be ready", ShortContainerID(containerID))
		case <-ticker.C:
			containerJSON, err := c.client.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container %s: %w", ShortContainerID(containerID), err)
			}

			if containerJSON.State.Running {
				c.logger.Printf("Container %s is now running.\n", ShortContainerID(containerID))
				return nil
			}

			if containerJSON.State.Restarting {
				c.logger.Printf("Container %s is restarting, waiting...\n", ShortContainerID(containerID))
			} else if containerJSON.State.Dead || containerJSON.State.ExitCode != 0 {
				return fmt.Errorf("container %s exited unexpectedly with code %d", ShortContainerID(containerID), containerJSON.State.ExitCode)
			}
		}
	}
}

// ListContainersByName lists containers by their name.
func (c *Client) ListContainersByName(ctx context.Context, name string, all bool) ([]container.Summary, error) {
	formattedName := fmt.Sprintf("/%s", name)
	return c.listContainers(ctx, filters.NewArgs(filters.Arg("name", formattedName)), all)
}

// ListContainersByLabel lists containers by a specific label.
func (c *Client) ListContainersByLabel(ctx context.Context, label string, all bool) ([]container.Summary, error) {
	return c.listContainers(ctx, filters.NewArgs(filters.Arg("label", label)), all)
}

// listContainers is a helper to list containers with filters.
func (c *Client) listContainers(ctx context.Context, filters filters.Args, all bool) ([]container.Summary, error) {
	return c.client.ContainerList(ctx, container.ListOptions{
		Filters: filters,
		All:     all,
	})
}

// StopContainer stops a container by its ID.
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	return c.client.ContainerStop(ctx, containerID, container.StopOptions{})
}

// StopContainersByLabel stops all containers with a specific label.
func (c *Client) StopContainersByLabel(ctx context.Context, label string) error {
	containers, err := c.ListContainersByLabel(ctx, label, false)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		c.logger.Println("No running containers found for the specified label.")
		return nil
	}

	for _, container := range containers {
		c.logger.Printf("Stopping container: %s\n", container.Names[0])

		if err := c.StopContainer(ctx, container.ID); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", container.Names[0], err)
		}
	}

	c.logger.Println("All containers with the specified label have been stopped.")

	return nil
}

// CreateNetwork creates a Docker network if it does not already exist.
func (c *Client) CreateNetwork(ctx context.Context, networkName string) (string, error) {
	networks, err := c.client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			c.logger.Println("Network already exists:", networkName)
			return net.ID, nil
		}
	}

	resp, err := c.client.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create network: %w", err)
	}

	c.logger.Println("Created network:", networkName)

	return resp.ID, nil
}

// Close closes the Docker client.
func (c *Client) Close() error {
	return c.client.Close()
}

// ShortContainerID shortens a Docker container ID to the first 12 characters.
func ShortContainerID(containerID string) string {
	if len(containerID) > 12 {
		return containerID[:12]
	}

	return containerID
}
