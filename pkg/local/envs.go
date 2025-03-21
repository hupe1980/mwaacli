package local

import (
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hupe1980/mwaacli/pkg/util"
)

type Envs struct {
	Credentials        *aws.Credentials
	S3DagsPath         string
	S3RequirementsPath string
	S3PluginsPath      string
}

// ToSlice converts the AWS credentials into a slice of environment variable strings.
func (e *Envs) ToSlice() []string {
	var envVars []string

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

func (r *Runner) buildEnvironmentVariables(envs *Envs) ([]string, error) {
	// Parse the .env file
	envFilePath := filepath.Join(r.opts.ClonePath, "docker", "config", ".env.localrunner")

	mwaaEnv, err := util.ParseEnvFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env file: %w", err)
	}

	// Append default environment variables
	mwaaEnv = append(mwaaEnv, "LOAD_EX=n", "EXECUTOR=Local")

	// Adds extra Envs if provided
	mwaaEnv = append(mwaaEnv, envs.ToSlice()...)

	return util.MergeEnvVars(mwaaEnv, true), nil
}
