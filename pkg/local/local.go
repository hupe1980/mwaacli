package local

import "strings"

const (
	MWAALocalRunnerRepoURL = "https://github.com/aws/aws-mwaa-local-runner.git"
	DefaultClonePath       = "./.aws-mwaa-local-runner"
	LabelKey               = "github.com.hupe1980.mwaacli"
)

// convertVersion converts a version string like "v2.20.2" to "2_20_2".
func convertVersion(version string) string {
	// Remove the leading "v" if it exists
	version = strings.TrimPrefix(version, "v")

	// Replace dots with underscores
	return strings.ReplaceAll(version, ".", "_")
}
