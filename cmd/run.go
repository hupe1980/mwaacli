package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

// newRunCommand creates a new Cobra command for executing Airflow CLI commands
// within an Amazon MWAA environment.
func newRunCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:   "run [command]",
		Short: "Execute an Airflow CLI command in MWAA",
		Long: `Executes an Airflow CLI command within an Amazon Managed Workflows for Apache Airflow (MWAA) environment.
See https://docs.aws.amazon.com/mwaa/latest/userguide/airflow-cli-command-reference.html#airflow-cli-commands-supported 
for a list of supported commands.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load AWS configuration
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return fmt.Errorf("failed to load AWS config: %w", err)
			}

			// Create an MWAA client
			client, err := mwaa.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create MWAA client: %w", err)
			}

			ctx := context.Background()

			// If no environment name is provided, attempt to infer it
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			// Combine command arguments into a single string
			command := strings.Join(args, " ")

			// Invoke the Airflow CLI command in the specified MWAA environment
			_, stdout, stderr, err := client.InvokeCliCommand(ctx, mwaaEnvName, command)
			if err != nil {
				return fmt.Errorf("failed to execute command: %w", err)
			}

			// Filter and print standard output
			cleanOutput := filterLogs(stdout)
			if cleanOutput != "" {
				cmd.Println(cleanOutput)
			}

			// Print error output if it's meaningful
			cleanError := filterLogs(stderr)
			if cleanError != "" {
				cmd.PrintErrln(cleanError)
			}

			return nil
		},
	}

	// Add a flag for specifying the MWAA environment name
	cmd.Flags().StringVarP(&mwaaEnvName, "env", "e", "", "MWAA environment name")

	// Set output streams for the command
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	return cmd
}

// filterLogs removes known warnings and unnecessary messages from Airflow CLI output.
func filterLogs(output string) string {
	lines := strings.Split(output, "\n")
	filteredLines := []string{}

	for _, line := range lines {
		// Ignore common Airflow warnings and CloudWatch logs
		if strings.Contains(line, "RemovedInAirflow3Warning") ||
			strings.Contains(line, "FutureWarning") ||
			strings.Contains(line, "UserWarning") ||
			strings.Contains(line, "CloudWatch logging is disabled") {
			continue
		}

		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, "\n")
}
