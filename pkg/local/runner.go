package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/moby/term"
)

const RepoURL = "https://github.com/aws/aws-mwaa-local-runner.git"

type Runner struct {
	RepoURL                  string
	ClonePath                string
	AirflowVersion           string
	DockerComposeProjectName string
}

// NewRunner creates a new MWAA installer
func NewRunner(version string) *Runner {
	return &Runner{
		RepoURL:                  RepoURL,
		ClonePath:                "./.aws-mwaa-local-runner",
		AirflowVersion:           version,
		DockerComposeProjectName: fmt.Sprintf("aws-mwaa-local-runner-%s", convertVersion(version)),
	}
}

func (r *Runner) Init() error {
	// Check if directory exists and is not empty
	if err := ensurePathIsEmptyOrNonExistent(r.ClonePath); err != nil {
		return err
	}

	// Clone repository
	memStore := memory.NewStorage()
	fs := memfs.New()

	repo, err := git.Clone(memStore, fs, &git.CloneOptions{
		URL:           r.RepoURL,
		ReferenceName: plumbing.ReferenceName(r.AirflowVersion),
		Progress:      os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get repository head: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return fmt.Errorf("failed to get commit object: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree from commit: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	err = tree.Files().ForEach(func(f *object.File) error {
		if matched, _ := regexp.MatchString(`^(mwaa-local-env|.github)`, f.Name); matched {
			// Skip files and directories
			return nil
		} else if matched, _ := regexp.MatchString(`^(dags|plugins|requirements|startup_script)`, f.Name); matched {
			return createFile(cwd, f)
		}

		return createFile(r.ClonePath, f)
	})
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	return nil
}

// ValidatePrereqs checks if Docker and Docker Compose are installed
func (r *Runner) ValidatePrereqs() error {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Check Docker version
	_, err = cli.ServerVersion(context.Background())
	if err != nil {
		return fmt.Errorf("docker is not installed or not running: %w", err)
	}

	// Check Docker Compose (via Docker CLI plugin)
	var composePath string
	if runtime.GOOS == "windows" {
		composePath = filepath.Join(os.Getenv("USERPROFILE"), ".docker", "cli-plugins", "docker-compose.exe")
	} else {
		composePath = filepath.Join(os.Getenv("HOME"), ".docker", "cli-plugins", "docker-compose")
	}

	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("docker Compose is not installed")
	}

	return nil
}

func (r *Runner) BuildImage() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	buildContextDir := filepath.Join(r.ClonePath, "docker")
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{fmt.Sprintf("amazon/mwaa-local:%s", convertVersion(r.AirflowVersion))},
		Dockerfile: "Dockerfile",
	}

	buildCtx, err := archive.TarWithOptions(buildContextDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer buildCtx.Close()

	resp, err := cli.ImageBuild(context.Background(), buildCtx, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}
	defer resp.Body.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)

	return jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
}

func (r *Runner) Start() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Check if the Docker Compose project is already running
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.project="+r.DockerComposeProjectName)),
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) > 0 {
		return fmt.Errorf("airflow local environment is already running")
	}

	// Define the paths for the Docker Compose files
	composeFilePath := filepath.Join(r.ClonePath, "docker", "docker-compose-local.yml")
	if _, err := os.Stat(composeFilePath); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose-local.yml not found at %s", composeFilePath)
	}

	err = r.runDockerCompose(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to start Airflow local environment: %w", err)
	}

	return nil
}

func (r *Runner) Stop() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Check if the Docker Compose project is running
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.project="+r.DockerComposeProjectName)),
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No running containers found for the Docker Compose project.")
		return nil
	}

	// Stop all containers in the Docker Compose project
	for _, c := range containers {
		fmt.Printf("Stopping container: %s\n", c.Names[0])

		if err := cli.ContainerStop(context.Background(), c.ID, container.StopOptions{}); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", c.Names[0], err)
		}
	}

	fmt.Println("All containers in the Docker Compose project have been stopped.")

	return nil
}

func (r *Runner) runDockerCompose(composeFilePath string) error {
	// Validate the composeFilePath
	absPath, err := filepath.Abs(composeFilePath)
	if err != nil {
		return fmt.Errorf("invalid compose file path: %w", err)
	}

	// Validate that the file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("compose file does not exist: %s", absPath)
	}

	// Allow only alphanumeric characters and hyphens for the project name to prevent command injection
	if !isValidProjectName(r.DockerComposeProjectName) {
		return fmt.Errorf("invalid project name: %s", r.DockerComposeProjectName)
	}

	// Run the Docker Compose command
	cmd := exec.Command("docker", "compose", "-p", r.DockerComposeProjectName, "-f", composeFilePath, "up", "-d") //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run Docker Compose: %w", err)
	}

	return nil
}

func (r *Runner) WaitForAppReady(appURL string) error {
	const (
		timeout  = 2 * time.Minute // Maximum wait time
		interval = 5 * time.Second // Polling interval
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

// convertVersion converts a version string like "v2.20.2" to "2_20_2".
func convertVersion(version string) string {
	// Remove the leading "v" if it exists
	version = strings.TrimPrefix(version, "v")

	// Replace dots with underscores
	return strings.ReplaceAll(version, ".", "_")
}

// isValidProjectName ensures the project name contains only safe characters
func isValidProjectName(name string) bool {
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validName.MatchString(name)
}

func createFile(path string, f *object.File) error {
	filePath := filepath.Join(path, f.Name)
	dirPath := filepath.Dir(filePath)

	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	reader, err := f.Blob.Reader()
	if err != nil {
		return fmt.Errorf("failed to get blob reader: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func ensurePathIsEmptyOrNonExistent(path string) error {
	if entries, err := os.ReadDir(path); err == nil && len(entries) > 0 {
		return fmt.Errorf("path %s already exists and is not empty", path)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	return nil
}
