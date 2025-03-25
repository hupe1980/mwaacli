// Package cmd provides the command-line interface (CLI) implementation for mwaacli.
// It defines the root command and its subcommands for interacting with Amazon Managed Workflows
// for Apache Airflow (MWAA). This package includes functionality for managing DAGs, environments,
// roles, variables, and other MWAA-related resources.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	cyan  = color.New(color.FgCyan).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	red   = color.New(color.FgRed, color.Bold).SprintFunc()
)

// Execute initializes and runs the root command for the CLI.
// It takes a version string as an argument and sets up the command execution.
func Execute(version string) {
	rootCmd := newRootCmd(version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, red("[ERROR]"), fmt.Sprintf("%s", err))
		os.Exit(1)
	}
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
	cmd.AddCommand(newDagsCommand(&opts))
	cmd.AddCommand(newEnvironmentsCommand(&opts))
	cmd.AddCommand(newLocalCommand(&opts))
	cmd.AddCommand(newLogsCommand(&opts))
	cmd.AddCommand(newOpenCommand(&opts))
	cmd.AddCommand(newRolesCommand(&opts))
	cmd.AddCommand(newRunCommand(&opts))
	cmd.AddCommand(newSBCommand(&opts))
	cmd.AddCommand(newVariablesCommand(&opts))

	return cmd
}

// getEnvironment retrieves the MWAA environment to use.
// If there is only one environment, it is returned automatically.
// If there are multiple environments, the user is prompted to choose one.
func getEnvironment(ctx context.Context, client *mwaa.Client) (string, error) {
	environments, err := client.ListEnvironments(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environments) == 1 {
		return environments[0], nil
	}

	if len(environments) > 1 {
		environment, err := chooseEnvironment(environments)
		if err != nil {
			return "", fmt.Errorf("failed to choose environment: %w", err)
		}

		return environment, nil
	}

	return "", fmt.Errorf("no environments found")
}

// chooseEnvironment prompts the user to select an MWAA environment from a list.
// It uses a fuzzy search to filter environments based on user input.
func chooseEnvironment(environments []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Active:   fmt.Sprintf("%s {{ . | cyan | bold }}", promptui.IconSelect),
		Inactive: "  {{ . | cyan }}",
		Selected: fmt.Sprintf("%s {{ . | cyan }}", promptui.IconGood),
	}

	searcher := func(input string, index int) bool {
		environment := environments[index]
		name := strings.Replace(strings.ToLower(environment), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     "Choose an environment",
		Items:     environments,
		Templates: templates,
		Size:      15,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return environments[i], nil
}

// printJSON prints the given value as a formatted JSON string to the command output.
// It returns an error if the value cannot be marshaled to JSON.
func printJSON(cmd *cobra.Command, v any) error {
	json, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	cmd.Println(string(json))

	return nil
}
