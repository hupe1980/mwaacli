package local

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func (r *Runner) PackageRequirements(ctx context.Context) error {
	requirementsConfig := &container.Config{
		Image: fmt.Sprintf("amazon/mwaa-local:%s", convertVersion(r.airflowVersion)),
		Cmd:   []string{"package-requirements"},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.DagsPath, "dags"), Target: "/usr/local/airflow/dags"},
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "plugins"), Target: "/usr/local/airflow/plugins"},
			{Type: mount.TypeBind, Source: filepath.Join(r.cwd, r.opts.ClonePath, "requirements"), Target: "/usr/local/airflow/requirements"},
		},
	}

	_, err := r.client.RunContainer(ctx, requirementsConfig, hostConfig, nil, "test-requirements")

	return err
}
