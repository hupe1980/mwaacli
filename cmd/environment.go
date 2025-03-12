package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

// newEnvironmentCommand creates a new cobra command for managing MWAA environments.
func newEnvironmentCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environment",
		Short: "Manage MWAA environments",
		Long:  "Manage Amazon Managed Workflows for Apache Airflow (MWAA) environments.",
	}

	cmd.AddCommand(newListEnvironmentsCommand(globalOpts))
	cmd.AddCommand(newGetEnvironmentCommand(globalOpts))

	return cmd
}

// newListEnvironmentsCommand creates a cobra command to list MWAA environments.
func newListEnvironmentsCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List MWAA environments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client, err := mwaa.NewClient(cfg)
			if err != nil {
				return err
			}

			environments, err := client.ListEnvironments(context.Background())
			if err != nil {
				return err
			}

			for _, env := range environments {
				cmd.Println(env)
			}

			return nil
		},
	}

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	return cmd
}

// newGetEnvironmentCommand creates a cobra command to get details of an MWAA environment.
func newGetEnvironmentCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [environment]",
		Short: "Get details of an MWAA environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client, err := mwaa.NewClient(cfg)
			if err != nil {
				return err
			}

			var mwaaEnvName string
			if len(args) > 0 {
				mwaaEnvName = args[0]
			}

			ctx := context.Background()

			if mwaaEnvName == "" {
				environments, err := client.ListEnvironments(ctx)
				if err != nil {
					return fmt.Errorf("failed to list environments: %w", err)
				}

				if len(environments) != 1 {
					return fmt.Errorf("environment name is required")
				}

				mwaaEnvName = environments[0]
			}

			env, err := client.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return err
			}

			envJSON, err := json.MarshalIndent(env, "", "  ")
			if err != nil {
				return err
			}

			cmd.Println(string(envJSON))

			return nil
		},
	}

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	return cmd
}
