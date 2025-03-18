package cmd

import (
	"context"
	"fmt"

	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

func newDagsCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dags",
		Short: "Manage DAGs in MWAA",
		Long:  `Manage Directed Acyclic Graphs (DAGs) in Amazon Managed Workflows for Apache Airflow (MWAA).`,
	}

	cmd.AddCommand(newListDagsCommand(globalOpts))
	cmd.AddCommand(newGetDagCommand(globalOpts))
	cmd.AddCommand(newGetDagSourceCommand(globalOpts))

	return cmd
}

func newListDagsCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		limit        int
		offset       int
		orderBy      string
		tags         []string
		onlyActive   bool
		paused       bool
		unpaused     bool
		fields       []string
		dagIDPattern string
		mwaaEnvName  string
	)

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List DAGs in the database",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client, err := mwaa.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			queryParams := map[string]any{
				"limit":       limit,
				"offset":      offset,
				"only_active": onlyActive,
			}

			if orderBy != "" {
				queryParams["order_by"] = orderBy
			}

			if len(tags) > 0 {
				queryParams["tags"] = tags
			}

			if len(fields) > 0 {
				queryParams["fields"] = fields
			}

			if paused != unpaused {
				if paused {
					queryParams["paused"] = true
				} else {
					queryParams["paused"] = false
				}
			}

			if dagIDPattern != "" {
				queryParams["dag_id_pattern"] = dagIDPattern
			}

			var response struct {
				Dags []map[string]any `json:"dags"`
			}
			if err := client.RestAPIGet(ctx, mwaaEnvName, "/dags", queryParams, &response); err != nil {
				return err
			}

			return printJSON(cmd, response.Dags)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "The number of items to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "The number of items to skip before starting to collect the result set")
	cmd.Flags().StringVar(&orderBy, "order-by", "", "The name of the field to order the results by. Prefix a field name with - to reverse the sort order")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "List of tags to filter results")
	cmd.Flags().BoolVar(&onlyActive, "only-active", true, "Only filter active DAGs")
	cmd.Flags().BoolVar(&paused, "paused", false, "Only filter paused DAGs")
	cmd.Flags().BoolVar(&unpaused, "unpaused", false, "Only filter unpaused DAGs")
	cmd.Flags().StringSliceVar(&fields, "fields", nil, "List of fields for return")
	cmd.Flags().StringVar(&dagIDPattern, "dag-id-pattern", "", "If set, only return DAGs with dag_ids matching this pattern")

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newGetDagCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		fields      []string
		mwaaEnvName string
	)

	cmd := &cobra.Command{
		Use:           "get [dag-id]",
		Short:         "Get details of a specific DAG",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client, err := mwaa.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			dagID := args[0]

			queryParams := map[string]any{}

			if len(fields) > 0 {
				queryParams["fields"] = fields
			}

			var response map[string]any
			if err := client.RestAPIGet(ctx, mwaaEnvName, fmt.Sprintf("/dags/%s", dagID), queryParams, &response); err != nil {
				return err
			}

			return printJSON(cmd, response)
		},
	}

	cmd.Flags().StringSliceVar(&fields, "fields", nil, "List of fields for return")

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func newGetDagSourceCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName string
	)

	cmd := &cobra.Command{
		Use:           "source [dag-id]",
		Short:         "Get details of a specific DAG",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client, err := mwaa.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			dagID := args[0]

			queryParams := map[string]any{
				"fields": "file_token",
			}

			var response map[string]any
			if err := client.RestAPIGet(ctx, mwaaEnvName, fmt.Sprintf("/dags/%s", dagID), queryParams, &response); err != nil {
				return err
			}

			var source string
			if err := client.RestAPIGet(ctx, mwaaEnvName, fmt.Sprintf("/dagSources/%s", response["file_token"]), nil, &source); err != nil {
				return err
			}

			cmd.Println(source)

			return nil
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}
