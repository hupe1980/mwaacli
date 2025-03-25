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

// newLogsCommand creates a new logs command for fetching MWAA logs.
func newLogsCommand(globalOpts *globalOptions) *cobra.Command {
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
		Use:           "logs",
		Short:         "Fetch logs from CloudWatch for an MWAA environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			// Create ignored log type map
			ignoredLogs := map[string]bool{
				"dag-processing": ignoreDagProcessing,
				"scheduler":      ignoreScheduler,
				"task":           ignoreTask,
				"webserver":      ignoreWebserver,
				"worker":         ignoreWorker,
			}

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

// extractLogGroupARNs extracts the CloudWatch log group ARNs from the LoggingConfiguration of an MWAA environment.
func extractLogGroupARNs(loggingConfig *types.LoggingConfiguration, ignoredLogs map[string]bool) []string {
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

// parseTimeOrDefault parses time in RFC3339 format or returns a default value
func parseTimeOrDefault(timeStr string, defaultTime time.Time) (time.Time, error) {
	if timeStr == "" {
		return defaultTime, nil
	}

	return time.Parse(time.RFC3339, timeStr)
}
