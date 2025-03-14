package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

// Execute initializes and runs the root command for the CLI.
// It takes a version string as an argument and sets up the command execution.
func Execute(version string) {
	rootCmd := newRootCmd(version)
	cobra.CheckErr(rootCmd.Execute())
}

// globalOptions holds common flags for AWS interaction.
type globalOptions struct {
	profile string // AWS profile name
	region  string // AWS region name
}

// newRootCmd creates and returns the root command for the CLI.
// It initializes global flags and adds subcommands.
func newRootCmd(version string) *cobra.Command {
	var opts globalOptions

	cmd := &cobra.Command{
		Use:     "mwaacli",
		Short:   "mwaacli is a CLI for interacting with MWAA",
		Long:    `mwaacli is a command-line interface for interacting with Amazon Managed Workflows for Apache Airflow (MWAA).`,
		Version: version,
	}

	// Set output streams for the command
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	// Define persistent flags for AWS profile and region.
	cmd.PersistentFlags().StringVar(&opts.profile, "profile", "", "AWS profile")
	cmd.PersistentFlags().StringVar(&opts.region, "region", "", "AWS region")

	// Add subcommands
	cmd.AddCommand(newDagCommand(&opts))
	cmd.AddCommand(newEnvironmentCommand(&opts))
	cmd.AddCommand(newOpenCommand(&opts))
	cmd.AddCommand(newRunCommand(&opts))
	cmd.AddCommand(newSBCommand(&opts))

	return cmd
}

func getEnvironment(ctx context.Context, client *mwaa.Client) (string, error) {
	environments, err := client.ListEnvironments(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environments) != 1 {
		return "", fmt.Errorf("environment name is required")
	}

	return environments[0], nil
}

func printJSON(cmd *cobra.Command, v any) error {
	json, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	cmd.Println(string(json))

	return nil
}
