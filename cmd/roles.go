package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

// newRolesCommand creates the parent "roles" command.
func newRolesCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roles",
		Short: "Manage Airflow roles",
		Long:  "Manage Airflow roles, including listing, retrieving, and creating roles.",
	}

	// Add subcommands
	cmd.AddCommand(newListRolesCommand(globalOpts))
	cmd.AddCommand(newGetRoleCommand(globalOpts))
	cmd.AddCommand(newCreateRoleCommand(globalOpts))

	return cmd
}

// newListRolesCommand creates the "list" subcommand for Airflow roles.
func newListRolesCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List Airflow roles",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client := mwaa.NewClient(cfg)

			ctx := context.Background()
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			var response struct {
				Roles []map[string]any `json:"roles"`
			}
			if err := client.RestAPIGet(ctx, mwaaEnvName, "/roles", nil, &response); err != nil {
				return err
			}

			return printJSON(cmd, response.Roles)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// newGetRoleCommand creates the "get" subcommand for retrieving a specific Airflow role.
func newGetRoleCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:           "get [role-name]",
		Short:         "Get details of an Airflow role",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1), // Ensure exactly one argument is provided
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]

			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client := mwaa.NewClient(cfg)

			ctx := context.Background()
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			var response map[string]any
			endpoint := fmt.Sprintf("/roles/%s", roleName)
			if err := client.RestAPIGet(ctx, mwaaEnvName, endpoint, nil, &response); err != nil {
				return err
			}

			return printJSON(cmd, response)
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// newCreateRoleCommand creates the "create" subcommand for creating a new Airflow role.
func newCreateRoleCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName string
		actions     []string
	)

	cmd := &cobra.Command{
		Use:           "create [role-name]",
		Short:         "Create a new Airflow role",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1), // Ensure exactly one argument is provided
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]

			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client := mwaa.NewClient(cfg)

			ctx := context.Background()
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			// Parse the actions flag into the required structure
			parsedActions := []map[string]map[string]string{}
			for _, action := range actions {
				parts := strings.Split(action, ".")
				if len(parts) != 2 {
					return fmt.Errorf("invalid action format: %s, expected format is action:resource", action)
				}
				parsedActions = append(parsedActions, map[string]map[string]string{
					"resource": {"name": parts[0]},
					"action":   {"name": parts[1]},
				})
			}

			// Prepare the payload for creating the role
			payload := map[string]any{
				"name":    roleName,
				"actions": parsedActions,
			}

			// Call the MWAA REST API to create the role
			var response map[string]any
			if err := client.RestAPIPost(ctx, mwaaEnvName, "/roles", nil, payload, &response); err != nil {
				return err
			}

			return printJSON(cmd, response)
		},
	}

	cmd.Flags().StringSliceVar(&actions, "actions", nil, "Comma-separated list of actions in the format resource.action (e.g. DAGs.can_read or DAG:example_dag_id.can_read)")
	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}
