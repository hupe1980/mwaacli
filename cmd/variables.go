package cmd

import (
	"context"

	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

func newVariablesCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variables",
		Short: "Manage variables in MWAA",
		Long:  `Manage variables in Amazon Managed Workflows for Apache Airflow (MWAA).`,
	}

	cmd.AddCommand(newListVariablesCommand(globalOpts))

	return cmd
}

func newListVariablesCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		limit       int
		offset      int
		orderBy     string
		mwaaEnvName string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List variables in the database",
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

			queryParams := map[string]any{
				"limit":  limit,
				"offset": offset,
			}

			if orderBy != "" {
				queryParams["order_by"] = orderBy
			}

			var response struct {
				Variables []map[string]any `json:"variables"`
			}
			if err := client.RestAPIGet(ctx, mwaaEnvName, "/variables", queryParams, &response); err != nil {
				return err
			}

			return printJSON(cmd, response.Variables)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "The number of items to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "The number of items to skip before starting to collect the result set")
	cmd.Flags().StringVar(&orderBy, "order-by", "", "The name of the field to order the results by. Prefix a field name with - to reverse the sort order")

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}
