package local

import (
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hupe1980/mwaacli/pkg/util"
)

// AWSCredentials represents AWS credentials and the associated region.
type AWSCredentials struct {
	aws.Credentials
	Region string // AWS region
}

// Envs represents environment variables required for the MWAA local runner.
type Envs struct {
	Credentials        *AWSCredentials // AWS credentials
	S3DagsPath         string          // Path to the S3 bucket for DAGs
	S3RequirementsPath string          // Path to the S3 bucket for requirements
	S3PluginsPath      string          // Path to the S3 bucket for plugins
}

// ToSlice converts the AWS credentials and other environment variables into a slice of strings.
// Each string is formatted as "KEY=VALUE".
func (e *Envs) ToSlice() []string {
	var envVars []string

	// Add AWS credentials to the environment variables
	if e.Credentials != nil {
		if e.Credentials.AccessKeyID != "" {
			envVars = append(envVars, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", e.Credentials.AccessKeyID))
		}

		if e.Credentials.SecretAccessKey != "" {
			envVars = append(envVars, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", e.Credentials.SecretAccessKey))
		}

		if e.Credentials.SessionToken != "" {
			envVars = append(envVars, fmt.Sprintf("AWS_SESSION_TOKEN=%s", e.Credentials.SessionToken))
		}

		if e.Credentials.Region != "" {
			envVars = append(envVars, fmt.Sprintf("AWS_REGION=%s", e.Credentials.Region), fmt.Sprintf("AWS_DEFAULT_REGION=%s", e.Credentials.Region))
		}
	}

	// Add other environment variables if needed
	if e.S3DagsPath != "" {
		envVars = append(envVars, fmt.Sprintf("S3_DAGS_PATH=%s", e.S3DagsPath))
	}

	if e.S3RequirementsPath != "" {
		envVars = append(envVars, fmt.Sprintf("S3_REQUIREMENTS_PATH=%s", e.S3RequirementsPath))
	}

	if e.S3PluginsPath != "" {
		envVars = append(envVars, fmt.Sprintf("S3_PLUGINS_PATH=%s", e.S3PluginsPath))
	}

	return envVars
}

// buildEnvironmentVariables constructs the environment variables required for the MWAA local runner.
func (r *Runner) buildEnvironmentVariables(envs *Envs) ([]string, error) {
	// Parse the .env file
	envFilePath := filepath.Join(r.opts.ClonePath, "docker", "config", ".env.localrunner")

	mwaaEnv, err := util.ParseEnvFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env file: %w", err)
	}

	// Append default environment variables
	mwaaEnv = append(mwaaEnv, "LOAD_EX=n", "EXECUTOR=Local")

	// Add extra environment variables if provided
	mwaaEnv = append(mwaaEnv, envs.ToSlice()...)

	// Merge environment variables and remove duplicates
	return util.MergeEnvVars(mwaaEnv, true), nil
}
