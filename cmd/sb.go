package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/hupe1980/mwaacli/pkg/secretsbackend"
	"github.com/spf13/cobra"
)

func newSBCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sb",
		Short: "Manage secrets backend",
		Long:  `Manage secrets backend in Amazon Managed Workflows for Apache Airflow (MWAA).`,
	}

	cmd.AddCommand(newListSBConnectionsCommand(globalOpts))
	cmd.AddCommand(newListSBVariablesCommand(globalOpts))

	cmd.AddCommand(newGetSBConnectionCommand(globalOpts))
	cmd.AddCommand(newGetSBVariableCommand(globalOpts))

	return cmd
}

func newListSBConnectionsCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:           "list-connections",
		Short:         "List connections in the secrets backend",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()

			secretsBackendClient, err := initSecretsBackendClient(ctx, globalOpts, &mwaaEnvName)
			if err != nil {
				return err
			}

			connections, err := secretsBackendClient.ListConnections(ctx)
			if err != nil {
				return fmt.Errorf("failed to list connections: %w", err)
			}

			return printJSON(cmd, connections)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newListSBVariablesCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:           "list-variables",
		Short:         "List variables in the secrets backend",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()

			secretsBackendClient, err := initSecretsBackendClient(ctx, globalOpts, &mwaaEnvName)
			if err != nil {
				return err
			}

			variables, err := secretsBackendClient.ListVariables(ctx)
			if err != nil {
				return fmt.Errorf("failed to list variables: %w", err)
			}

			return printJSON(cmd, variables)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newGetSBConnectionCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:           "get-connection [conn-id]",
		Short:         "Get a connection from the secrets backend",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			secretsBackendClient, err := initSecretsBackendClient(ctx, globalOpts, &mwaaEnvName)
			if err != nil {
				return err
			}

			connection, err := secretsBackendClient.GetConnection(ctx, args[0])
			if err != nil {
				return fmt.Errorf("failed to get connection: %w", err)
			}

			var data map[string]any
			if err := json.Unmarshal([]byte(connection), &data); err != nil {
				cmd.Println(connection)
				return nil
			}

			return printJSON(cmd, data)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newGetSBVariableCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:           "get-variable [var-name]",
		Short:         "Get a variable from the secrets backend",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			secretsBackendClient, err := initSecretsBackendClient(ctx, globalOpts, &mwaaEnvName)
			if err != nil {
				return err
			}

			variable, err := secretsBackendClient.GetVariable(ctx, args[0])
			if err != nil {
				return fmt.Errorf("failed to get variable: %w", err)
			}

			var data map[string]any
			if err := json.Unmarshal([]byte(variable), &data); err != nil {
				cmd.Println(variable)
				return nil
			}

			return printJSON(cmd, data)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// initSecretsBackendClient sets up a secrets backend client for the specified environment.
func initSecretsBackendClient(ctx context.Context, globalOpts *globalOptions, mwaaEnvName *string) (*secretsbackend.Client, error) {
	cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := mwaa.NewClient(cfg)

	if *mwaaEnvName == "" {
		*mwaaEnvName, err = getEnvironment(ctx, client)
		if err != nil {
			return nil, err
		}
	}

	env, err := client.GetEnvironment(ctx, *mwaaEnvName)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	return secretsbackend.NewClient(cfg, env)
}
