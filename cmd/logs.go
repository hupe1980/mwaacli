package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	"github.com/hupe1980/mwaacli/pkg/cloudwatch"
	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

// newLogsCommand creates the parent logs command.
func newLogsCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Manage MWAA logs",
	}

	cmd.AddCommand(newLogsAllCommand(globalOpts))
	cmd.AddCommand(newLogsDagProcessingCommand(globalOpts))
	cmd.AddCommand(newLogsSchedulerCommand(globalOpts))
	cmd.AddCommand(newLogsTaskCommand(globalOpts))
	cmd.AddCommand(newLogsWebserverCommand(globalOpts))
	cmd.AddCommand(newLogsWorkerCommand(globalOpts))

	return cmd
}

// fetchLogs is a helper function to fetch logs for a specific log type or all logs.
func fetchLogs(globalOpts *globalOptions, cmd *cobra.Command, ignoredLogs map[string]bool, startTime, endTime, filterPattern, mwaaEnvName string) error {
	cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS config: %w", err)
	}

	client := mwaa.NewClient(cfg)
	ctx := context.Background()

	// Get environment name if not provided
	if mwaaEnvName == "" {
		mwaaEnvName, err = getEnvironment(ctx, client)
		if err != nil {
			return err
		}
	}

	// Fetch MWAA environment details
	environment, err := client.GetEnvironment(ctx, mwaaEnvName)
	if err != nil {
		return fmt.Errorf("failed to get environment: %w", err)
	}

	// Extract log group ARNs
	logGroupARNs := extractLogGroupARNs(environment.LoggingConfiguration, ignoredLogs)

	// Parse start and end times safely
	start, err := parseTimeOrDefault(startTime, time.Now().Add(-1*time.Hour)) // Default: 1 hour ago
	if err != nil {
		return fmt.Errorf("invalid start time format: %w", err)
	}

	end, err := parseTimeOrDefault(endTime, time.Now()) // Default: now
	if err != nil {
		return fmt.Errorf("invalid end time format: %w", err)
	}

	// Ensure start is before end
	if start.After(end) {
		return fmt.Errorf("start time must be before end time")
	}

	// Initialize CloudWatch Logs client
	cloudwatchClient := cloudwatch.NewClient(cfg)

	// Fetch logs
	logs, err := cloudwatchClient.FetchLogs(ctx, logGroupARNs, &cloudwatch.LogFilter{
		StartTime:     aws.Int64(start.UnixMilli()),
		EndTime:       aws.Int64(end.UnixMilli()),
		FilterPattern: aws.String(filterPattern),
	})
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	// Print logs with timestamp and log group name
	for _, log := range logs {
		cmd.Printf("[%s] %s\n", log.LogGroup, log.Message)
	}

	return nil
}

// newLogsAllCommand creates the "logs all" subcommand for fetching all MWAA logs.
func newLogsAllCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName   string
		startTime     string
		endTime       string
		filterPattern string

		// Flags to ignore specific log types
		ignoreDagProcessing bool
		ignoreScheduler     bool
		ignoreTask          bool
		ignoreWebserver     bool
		ignoreWorker        bool
	)

	cmd := &cobra.Command{
		Use:           "all",
		Short:         "Fetch all logs from CloudWatch for an MWAA environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ignoredLogs := map[string]bool{
				"dag-processing": ignoreDagProcessing,
				"scheduler":      ignoreScheduler,
				"task":           ignoreTask,
				"webserver":      ignoreWebserver,
				"worker":         ignoreWorker,
			}
			return fetchLogs(globalOpts, cmd, ignoredLogs, startTime, endTime, filterPattern, mwaaEnvName)
		},
	}

	// Log type ignore flags
	cmd.Flags().BoolVar(&ignoreDagProcessing, "ignore-dag-processing", false, "Ignore DAG processing logs")
	cmd.Flags().BoolVar(&ignoreScheduler, "ignore-scheduler", false, "Ignore scheduler logs")
	cmd.Flags().BoolVar(&ignoreTask, "ignore-task", false, "Ignore task logs")
	cmd.Flags().BoolVar(&ignoreWebserver, "ignore-webserver", false, "Ignore webserver logs")
	cmd.Flags().BoolVar(&ignoreWorker, "ignore-worker", false, "Ignore worker logs")

	// Other filters
	cmd.Flags().StringVar(&startTime, "start-time", "", "Start time for logs in RFC3339 format (default: 1 hour ago)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "End time for logs in RFC3339 format (default: now)")
	cmd.Flags().StringVar(&filterPattern, "filter-pattern", "", "Filter pattern for logs (optional)")
	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// newLogsDagProcessingCommand creates the "logs dag-processing" subcommand for fetching DAG processing logs.
func newLogsDagProcessingCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName   string
		startTime     string
		endTime       string
		filterPattern string
	)

	cmd := &cobra.Command{
		Use:           "dag-processing",
		Short:         "Fetch DAG processing logs from CloudWatch for an MWAA environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ignoredLogs := map[string]bool{
				"dag-processing": false, // Include only DAG processing logs
				"scheduler":      true,
				"task":           true,
				"webserver":      true,
				"worker":         true,
			}
			return fetchLogs(globalOpts, cmd, ignoredLogs, startTime, endTime, filterPattern, mwaaEnvName)
		},
	}

	// Flags for filtering logs
	cmd.Flags().StringVar(&startTime, "start-time", "", "Start time for logs in RFC3339 format (default: 1 hour ago)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "End time for logs in RFC3339 format (default: now)")
	cmd.Flags().StringVar(&filterPattern, "filter-pattern", "", "Filter pattern for logs (optional)")
	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// newLogsSchedulerCommand creates the "logs scheduler" subcommand for fetching scheduler logs.
func newLogsSchedulerCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName   string
		startTime     string
		endTime       string
		filterPattern string
	)

	cmd := &cobra.Command{
		Use:           "scheduler",
		Short:         "Fetch scheduler logs from CloudWatch for an MWAA environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ignoredLogs := map[string]bool{
				"dag-processing": true,
				"scheduler":      false, // Include only scheduler logs
				"task":           true,
				"webserver":      true,
				"worker":         true,
			}
			return fetchLogs(globalOpts, cmd, ignoredLogs, startTime, endTime, filterPattern, mwaaEnvName)
		},
	}

	// Flags for filtering logs
	cmd.Flags().StringVar(&startTime, "start-time", "", "Start time for logs in RFC3339 format (default: 1 hour ago)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "End time for logs in RFC3339 format (default: now)")
	cmd.Flags().StringVar(&filterPattern, "filter-pattern", "", "Filter pattern for logs (optional)")
	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// newLogsTaskCommand creates the "logs task" subcommand for fetching task logs.
func newLogsTaskCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName   string
		startTime     string
		endTime       string
		filterPattern string
	)

	cmd := &cobra.Command{
		Use:           "task",
		Short:         "Fetch task logs from CloudWatch for an MWAA environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ignoredLogs := map[string]bool{
				"dag-processing": true,
				"scheduler":      true,
				"task":           false, // Include only task logs
				"webserver":      true,
				"worker":         true,
			}
			return fetchLogs(globalOpts, cmd, ignoredLogs, startTime, endTime, filterPattern, mwaaEnvName)
		},
	}

	// Flags for filtering logs
	cmd.Flags().StringVar(&startTime, "start-time", "", "Start time for logs in RFC3339 format (default: 1 hour ago)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "End time for logs in RFC3339 format (default: now)")
	cmd.Flags().StringVar(&filterPattern, "filter-pattern", "", "Filter pattern for logs (optional)")
	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// newLogsWebserverCommand creates the "logs webserver" subcommand for fetching webserver logs.
func newLogsWebserverCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName   string
		startTime     string
		endTime       string
		filterPattern string
	)

	cmd := &cobra.Command{
		Use:           "webserver",
		Short:         "Fetch webserver logs from CloudWatch for an MWAA environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ignoredLogs := map[string]bool{
				"dag-processing": true,
				"scheduler":      true,
				"task":           true,
				"webserver":      false, // Include only webserver logs
				"worker":         true,
			}
			return fetchLogs(globalOpts, cmd, ignoredLogs, startTime, endTime, filterPattern, mwaaEnvName)
		},
	}

	// Flags for filtering logs
	cmd.Flags().StringVar(&startTime, "start-time", "", "Start time for logs in RFC3339 format (default: 1 hour ago)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "End time for logs in RFC3339 format (default: now)")
	cmd.Flags().StringVar(&filterPattern, "filter-pattern", "", "Filter pattern for logs (optional)")
	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// newLogsWorkerCommand creates the "logs worker" subcommand for fetching worker logs.
func newLogsWorkerCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		mwaaEnvName   string
		startTime     string
		endTime       string
		filterPattern string
	)

	cmd := &cobra.Command{
		Use:           "worker",
		Short:         "Fetch worker logs from CloudWatch for an MWAA environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ignoredLogs := map[string]bool{
				"dag-processing": true,
				"scheduler":      true,
				"task":           true,
				"webserver":      true,
				"worker":         false, // Include only worker logs
			}
			return fetchLogs(globalOpts, cmd, ignoredLogs, startTime, endTime, filterPattern, mwaaEnvName)
		},
	}

	// Flags for filtering logs
	cmd.Flags().StringVar(&startTime, "start-time", "", "Start time for logs in RFC3339 format (default: 1 hour ago)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "End time for logs in RFC3339 format (default: now)")
	cmd.Flags().StringVar(&filterPattern, "filter-pattern", "", "Filter pattern for logs (optional)")
	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

// extractLogGroupARNs extracts the CloudWatch log group ARNs from the LoggingConfiguration of an MWAA environment.
func extractLogGroupARNs(loggingConfig *types.LoggingConfiguration, ignoredLogs map[string]bool) []string {
	if loggingConfig == nil {
		return []string{} // Return an empty slice if loggingConfig is nil
	}

	var logGroupARNs []string

	// Helper function to append ARN if the log type is enabled and not ignored
	appendLogGroupARN := func(logConfig *types.ModuleLoggingConfiguration, logType string) {
		if logConfig != nil && logConfig.Enabled != nil && aws.ToBool(logConfig.Enabled) &&
			logConfig.CloudWatchLogGroupArn != nil && !ignoredLogs[logType] {
			logGroupARNs = append(logGroupARNs, aws.ToString(logConfig.CloudWatchLogGroupArn))
		}
	}

	// Check each log type
	appendLogGroupARN(loggingConfig.DagProcessingLogs, "dag-processing")
	appendLogGroupARN(loggingConfig.SchedulerLogs, "scheduler")
	appendLogGroupARN(loggingConfig.TaskLogs, "task")
	appendLogGroupARN(loggingConfig.WebserverLogs, "webserver")
	appendLogGroupARN(loggingConfig.WorkerLogs, "worker")

	return logGroupARNs
}

// parseTimeOrDefault parses time in RFC3339 format or returns a default value.
func parseTimeOrDefault(timeStr string, defaultTime time.Time) (time.Time, error) {
	if timeStr == "" {
		return defaultTime, nil
	}

	return time.Parse(time.RFC3339, timeStr)
}
