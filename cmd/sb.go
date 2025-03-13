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

	cmd.AddCommand(newListConnectionsCommand(globalOpts))
	cmd.AddCommand(newListVariablesCommand(globalOpts))

	cmd.AddCommand(newGetConnectionCommand(globalOpts))
	cmd.AddCommand(newGetVariableCommand(globalOpts))

	return cmd
}

func newListConnectionsCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:   "list-connections",
		Short: "List connections in the secrets backend",
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			env, err := client.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return fmt.Errorf("failed to get environment: %w", err)
			}

			secretsClient, err := secretsbackend.NewClient(cfg, env)
			if err != nil {
				return fmt.Errorf("failed to create secrets backend client: %w", err)
			}

			connections, err := secretsClient.ListConnections(ctx)
			if err != nil {
				return fmt.Errorf("failed to list connections: %w", err)
			}

			return printJSON(cmd, connections)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newListVariablesCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:   "list-variables",
		Short: "List variables in the secrets backend",
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			env, err := client.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return fmt.Errorf("failed to get environment: %w", err)
			}

			secretsClient, err := secretsbackend.NewClient(cfg, env)
			if err != nil {
				return fmt.Errorf("failed to create secrets backend client: %w", err)
			}

			variables, err := secretsClient.ListVariables(ctx)
			if err != nil {
				return fmt.Errorf("failed to list variables: %w", err)
			}

			return printJSON(cmd, variables)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newGetConnectionCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:   "get-connection [conn-id]",
		Short: "Get a connection from the secrets backend",
		Args:  cobra.ExactArgs(1),
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

			env, err := client.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return fmt.Errorf("failed to get environment: %w", err)
			}

			secretsClient, err := secretsbackend.NewClient(cfg, env)
			if err != nil {
				return fmt.Errorf("failed to create secrets backend client: %w", err)
			}

			connection, err := secretsClient.GetConnection(ctx, args[0])
			if err != nil {
				return fmt.Errorf("failed to get connection: %w", err)
			}

			var data map[string]any
			if err := json.Unmarshal([]byte(connection), &data); err != nil {
				return printJSON(cmd, connection)
			}

			return printJSON(cmd, data)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newGetVariableCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:   "get-variable [var-name]",
		Short: "Get a variable from the secrets backend",
		Args:  cobra.ExactArgs(1),
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

			env, err := client.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return fmt.Errorf("failed to get environment: %w", err)
			}

			secretsClient, err := secretsbackend.NewClient(cfg, env)
			if err != nil {
				return fmt.Errorf("failed to create secrets backend client: %w", err)
			}

			variable, err := secretsClient.GetVariable(ctx, args[0])
			if err != nil {
				return fmt.Errorf("failed to get variable: %w", err)
			}

			var data map[string]any
			if err := json.Unmarshal([]byte(variable), &data); err != nil {
				return printJSON(cmd, variable)
			}

			return printJSON(cmd, data)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}
