package cmd

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	"github.com/stretchr/testify/assert"
)

func TestExtractLogGroupARNs(t *testing.T) {
	tests := []struct {
		name          string
		loggingConfig *types.LoggingConfiguration
		ignoredLogs   map[string]bool
		expectedARNs  []string
	}{
		{
			name: "Include all log types",
			loggingConfig: &types.LoggingConfiguration{
				DagProcessingLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:dag-processing"),
				},
				SchedulerLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:scheduler"),
				},
				TaskLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:task"),
				},
				WebserverLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:webserver"),
				},
				WorkerLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:worker"),
				},
			},
			ignoredLogs: map[string]bool{
				"dag-processing": false,
				"scheduler":      false,
				"task":           false,
				"webserver":      false,
				"worker":         false,
			},
			expectedARNs: []string{
				"arn:aws:logs:region:account-id:log-group:dag-processing",
				"arn:aws:logs:region:account-id:log-group:scheduler",
				"arn:aws:logs:region:account-id:log-group:task",
				"arn:aws:logs:region:account-id:log-group:webserver",
				"arn:aws:logs:region:account-id:log-group:worker",
			},
		},
		{
			name: "Ignore some log types",
			loggingConfig: &types.LoggingConfiguration{
				DagProcessingLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:dag-processing"),
				},
				SchedulerLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:scheduler"),
				},
				TaskLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:task"),
				},
				WebserverLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:webserver"),
				},
				WorkerLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(true),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:worker"),
				},
			},
			ignoredLogs: map[string]bool{
				"dag-processing": true,
				"scheduler":      false,
				"task":           true,
				"webserver":      false,
				"worker":         true,
			},
			expectedARNs: []string{
				"arn:aws:logs:region:account-id:log-group:scheduler",
				"arn:aws:logs:region:account-id:log-group:webserver",
			},
		},
		{
			name: "No enabled log types",
			loggingConfig: &types.LoggingConfiguration{
				DagProcessingLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(false),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:dag-processing"),
				},
				SchedulerLogs: &types.ModuleLoggingConfiguration{
					Enabled:               aws.Bool(false),
					CloudWatchLogGroupArn: aws.String("arn:aws:logs:region:account-id:log-group:scheduler"),
				},
			},
			ignoredLogs: map[string]bool{
				"dag-processing": false,
				"scheduler":      false,
			},
			expectedARNs: []string{},
		},
		{
			name:          "Nil logging configuration",
			loggingConfig: nil,
			ignoredLogs: map[string]bool{
				"dag-processing": false,
				"scheduler":      false,
			},
			expectedARNs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLogGroupARNs(tt.loggingConfig, tt.ignoredLogs)
			assert.ElementsMatch(t, tt.expectedARNs, result)
		})
	}
}

func TestParseTimeOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		timeStr      string
		defaultTime  time.Time
		expectedTime time.Time
		expectError  bool
	}{
		{
			name:         "Valid RFC3339 time string",
			timeStr:      "2023-03-27T15:04:05Z",
			defaultTime:  time.Now(),
			expectedTime: time.Date(2023, 3, 27, 15, 4, 5, 0, time.UTC),
			expectError:  false,
		},
		{
			name:         "Empty time string, use default",
			timeStr:      "",
			defaultTime:  time.Date(2023, 3, 27, 12, 0, 0, 0, time.UTC),
			expectedTime: time.Date(2023, 3, 27, 12, 0, 0, 0, time.UTC),
			expectError:  false,
		},
		{
			name:         "Invalid time string",
			timeStr:      "invalid-time",
			defaultTime:  time.Now(),
			expectedTime: time.Time{},
			expectError:  true,
		},
		{
			name:         "Valid RFC3339 time string with offset",
			timeStr:      "2023-03-27T15:04:05+02:00",
			defaultTime:  time.Now(),
			expectedTime: time.Date(2023, 3, 27, 15, 4, 5, 0, time.FixedZone("", 2*60*60)),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeOrDefault(tt.timeStr, tt.defaultTime)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTime, result)
			}
		})
	}
}
