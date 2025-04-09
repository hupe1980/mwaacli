package local

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func (r *Runner) TestRequirements(ctx context.Context) error {
	requirementsConfig := &container.Config{
		Image:        fmt.Sprintf("amazon/mwaa-local:%s", convertVersion(r.airflowVersion)),
		Cmd:          []string{"test-requirements"},
		Tty:          true, // Allocate a pseudo-TTY
		OpenStdin:    true, // Keep stdin open for interactive mode
		AttachStdin:  true, // Attach stdin for interaction
		AttachStdout: true, // Attach stdout for output
		AttachStderr: true, // Attach stderr for error output
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.DagsPath, "dags"), Target: "/usr/local/airflow/dags"},
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "plugins"), Target: "/usr/local/airflow/plugins"},
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "requirements"), Target: "/usr/local/airflow/requirements"},
		},
	}

	containerID, err := r.client.RunContainer(ctx, requirementsConfig, hostConfig, nil, "test-requirements")
	if err != nil {
		return fmt.Errorf("failed to run container: %w", err)
	}

	// Attach to the container for interactive mode
	if err := r.client.AttachToContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}

	return err
}

type TestStartupScriptOptions struct {
	Envs *Envs
}

func (r *Runner) TestStartupScript(ctx context.Context, optFns ...func(o *TestStartupScriptOptions)) error {
	opts := TestStartupScriptOptions{
		Envs: nil,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	mwaaEnv := opts.Envs.ToSlice()

	startupConfig := &container.Config{
		Image:        fmt.Sprintf("amazon/mwaa-local:%s", convertVersion(r.airflowVersion)),
		Env:          mwaaEnv,
		Cmd:          []string{"test-startup-script"},
		Tty:          true, // Allocate a pseudo-TTY
		OpenStdin:    true, // Keep stdin open for interactive mode
		AttachStdin:  true, // Attach stdin for interaction
		AttachStdout: true, // Attach stdout for output
		AttachStderr: true, // Attach stderr for error output
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "startup_script"), Target: "/usr/local/airflow/startup"},
		},
	}

	// Run the container
	containerID, err := r.client.RunContainer(ctx, startupConfig, hostConfig, nil, "test-startup-script")
	if err != nil {
		return fmt.Errorf("failed to run container: %w", err)
	}

	// Attach to the container for interactive mode
	if err := r.client.AttachToContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}

	return nil
}
