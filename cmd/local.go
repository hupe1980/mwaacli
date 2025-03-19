package cmd

import (
	"context"
	"fmt"

	"github.com/hupe1980/mwaacli/pkg/local"
	"github.com/hupe1980/mwaacli/pkg/util"
	"github.com/spf13/cobra"
)

const (
	defaultVersion = "v2.10.3"
	webserverURL   = "http://localhost:8080"
)

func newLocalCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Setup and control the AWS MWAA local runner",
		Long:  `Manage the AWS MWAA local runner, including setup, starting, stopping, and checking the status.`,
	}

	cmd.AddCommand(newInitCommand(globalOpts))
	cmd.AddCommand(newBuildImageCommand(globalOpts))
	cmd.AddCommand(newStartCommand(globalOpts))
	cmd.AddCommand(newStopCommand(globalOpts))

	return cmd
}

func newInitCommand(_ *globalOptions) *cobra.Command {
	var (
		version string
		repoURL string
	)

	cmd := &cobra.Command{
		Use:           "init",
		Short:         "Initialize the AWS MWAA local runner",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Setting up the AWS MWAA local runner...")

			installer, err := local.NewInstaller(version, func(o *local.InstallerOptions) {
				o.RepoURL = repoURL
			})
			if err != nil {
				return fmt.Errorf("failed to create installer: %w", err)
			}

			if err := installer.Run(); err != nil {
				return err
			}

			cmd.Println("AWS MWAA local runner setup complete.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")
	cmd.Flags().StringVar(&repoURL, "repo-url", local.MWAALocalRunnerRepoURL, "Specify the repository URL for the AWS MWAA local runner")

	return cmd
}

func newBuildImageCommand(_ *globalOptions) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:           "build-image",
		Short:         "Build the Docker image for the AWS MWAA local runner",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Building the Docker image for the AWS MWAA local runner...")

			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()

			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			cmd.Println("Docker image built successfully.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")

	return cmd
}

func newStartCommand(_ *globalOptions) *cobra.Command {
	var (
		version string
		open    bool
	)

	cmd := &cobra.Command{
		Use:           "start",
		Short:         "Start the AWS MWAA local runner environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Starting the AWS MWAA local runner environment...")

			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()

			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			cmd.Println("Docker image built successfully.")

			if err := runner.Start(ctx); err != nil {
				return fmt.Errorf("failed to start AWS MWAA local runner environment: %w", err)
			}

			// Wait for the application to be ready
			cmd.Println("Waiting for the Airflow webserver to be ready...")
			if err := runner.WaitForAppReady(fmt.Sprintf("%s/health", webserverURL)); err != nil {
				return fmt.Errorf("application is not ready: %w", err)
			}

			cmd.Println("AWS MWAA local runner environment started successfully.")

			if open {
				cmd.Println("Opening the Airflow UI in the default web browser...")
				if err := util.OpenBrowser(webserverURL); err != nil {
					return fmt.Errorf("failed to open the Airflow UI: %w", err)
				}
			} else {
				cmd.Printf("You can access the Airflow UI at %s\n", webserverURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")
	cmd.Flags().BoolVar(&open, "open", false, "Open the Airflow UI in the default web browser after starting")

	return cmd
}

func newStopCommand(_ *globalOptions) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:           "stop",
		Short:         "Stop the AWS MWAA local runner environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Stopping the AWS MWAA local runner environment...")
			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}

			ctx := context.Background()

			if err := runner.Stop(ctx); err != nil {
				return fmt.Errorf("failed to stop AWS MWAA local runner environment: %w", err)
			}

			cmd.Println("AWS MWAA local runner environment stopped successfully.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")

	return cmd
}
