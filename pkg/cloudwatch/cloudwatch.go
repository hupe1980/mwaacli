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

// LogEvent represents a CloudWatch log event
type LogEvent struct {
	Timestamp int64
	Message   string
	LogGroup  string
}

type Client struct {
	client *cloudwatchlogs.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		client: cloudwatchlogs.NewFromConfig(cfg.AWSConfig),
	}
}

type LogFilter struct {
	StartTime     *int64
	EndTime       *int64
	FilterPattern *string
}

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

// Extracts log group name from ARN
func extractLogGroupName(arn string) (string, error) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return "", fmt.Errorf("invalid Log Group ARN format: %s", arn)
	}

	return parts[5][10:], nil // Remove "log-group:" prefix
}
