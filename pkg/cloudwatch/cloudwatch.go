// Package cloudwatch provides a client for interacting with Amazon CloudWatch Logs.
// It simplifies fetching and filtering log events from CloudWatch log groups, enabling
// efficient log retrieval and processing.
package cloudwatch

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/hupe1980/mwaacli/pkg/config"
)

// LogEvent represents a CloudWatch log event.
type LogEvent struct {
	Timestamp int64  // The timestamp of the log event in milliseconds since the epoch.
	Message   string // The message content of the log event.
	LogGroup  string // The name of the log group where the event was logged.
}

// Client provides methods to interact with Amazon CloudWatch Logs.
type Client struct {
	client *cloudwatchlogs.Client // The AWS CloudWatch Logs client.
}

// NewClient initializes a new CloudWatch Logs client using the provided configuration.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		client: cloudwatchlogs.NewFromConfig(cfg.AWSConfig),
	}
}

// LogFilter defines the filtering criteria for fetching log events.
type LogFilter struct {
	StartTime     *int64  // The start time for the log events in milliseconds since the epoch.
	EndTime       *int64  // The end time for the log events in milliseconds since the epoch.
	FilterPattern *string // The filter pattern to match log events.
}

// FetchLogs retrieves log events from the specified CloudWatch log groups based on the provided filter.
// This method fetches logs from multiple log groups, applies the filter, and sorts the logs by timestamp.
func (c *Client) FetchLogs(ctx context.Context, logGroupARNs []string, filter *LogFilter) ([]LogEvent, error) {
	var allLogs []LogEvent

	for _, arn := range logGroupARNs {
		logGroupName, err := extractLogGroupName(arn)
		if err != nil {
			return nil, fmt.Errorf("failed to extract log group name: %w", err)
		}

		logs, err := c.getFilteredLogs(ctx, logGroupName, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch logs for %s: %w", arn, err)
		}

		allLogs = append(allLogs, logs...)
	}

	// Sort logs by timestamp
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp < allLogs[j].Timestamp
	})

	return allLogs, nil
}

// getFilteredLogs retrieves filtered log events from a specific CloudWatch log group.
// This method uses a paginator to fetch all pages of log events that match the filter criteria.
func (c *Client) getFilteredLogs(ctx context.Context, logGroupName string, filter *LogFilter) ([]LogEvent, error) {
	var logs []LogEvent

	// Create a paginator for the FilterLogEvents API
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(c.client, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		StartTime:     filter.StartTime,
		EndTime:       filter.EndTime,
		FilterPattern: filter.FilterPattern,
	})

	// Iterate through all pages
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get log events: %w", err)
		}

		// Append log events from the current page
		for _, event := range resp.Events {
			logs = append(logs, LogEvent{
				Timestamp: *event.Timestamp,
				Message:   *event.Message,
				LogGroup:  logGroupName,
			})
		}
	}

	return logs, nil
}

// extractLogGroupName extracts the log group name from a CloudWatch log group ARN.
// The ARN must follow the standard format for CloudWatch log group ARNs.
func extractLogGroupName(arn string) (string, error) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return "", fmt.Errorf("invalid Log Group ARN format: %s", arn)
	}

	return parts[5][10:], nil // Remove "log-group:" prefix
}
