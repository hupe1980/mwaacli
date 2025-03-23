package local

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// ConvertAirflowCfgToMap reads an Airflow configuration file in INI format
// and converts it into a map of key-value pairs. The keys are formatted as
// "section.key" (e.g., "core.dag_dir_list_interval").
// The "DEFAULT" section in the INI file is ignored.
func ConvertAirflowCfgToMap(filename string) (map[string]string, error) {
	// Load the INI file
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file '%s': %v", filename, err)
	}

	// Initialize the map to store config values
	configMap := make(map[string]string)

	// Iterate over each section in the INI file
	for _, section := range cfg.Sections() {
		// Skip the DEFAULT section
		if section.Name() == "DEFAULT" {
			continue
		}

		// Iterate over the keys in each section and add to the map
		for _, key := range section.Keys() {
			configMap[fmt.Sprintf("%s.%s", section.Name(), key.Name())] = key.Value()
		}
	}

	return configMap, nil
}

// Diff represents a single difference between the local and remote configurations.
type Diff struct {
	Key         string
	Type        string // "missing" or "different"
	LocalValue  string
	RemoteValue string
}

// Diffs represents a collection of differences between local and remote configurations.
type Diffs []Diff

// ToString converts the Diffs collection to a human-readable string
func (ds Diffs) ToString() string {
	var result string

	// Separate paragraphs for different types of differences
	var missingLocalDiffs, missingRemoteDiffs, differentDiffs []Diff

	for _, diff := range ds {
		switch diff.Type {
		case "missing":
			if diff.LocalValue == "" {
				missingLocalDiffs = append(missingLocalDiffs, diff)
			} else {
				missingRemoteDiffs = append(missingRemoteDiffs, diff)
			}
		case "different":
			differentDiffs = append(differentDiffs, diff)
		}
	}

	// Adding Missing Remote Section
	if len(missingRemoteDiffs) > 0 {
		result += "Missing in Remote Configuration:\n"
		for _, diff := range missingRemoteDiffs {
			result += fmt.Sprintf("  Key: %s\n", diff.Key)
			result += fmt.Sprintf("  Local Value: %s\n", diff.LocalValue)
		}

		result += "\n"
	}

	// Adding Missing Local Section
	if len(missingLocalDiffs) > 0 {
		result += "Missing in Local Configuration:\n"
		for _, diff := range missingLocalDiffs {
			result += fmt.Sprintf("  Key: %s\n", diff.Key)
			result += fmt.Sprintf("  Remote Value: %s\n", diff.RemoteValue)
		}

		result += "\n"
	}

	// Adding Differences Section
	if len(differentDiffs) > 0 {
		result += "Differences in Values:\n"
		for _, diff := range differentDiffs {
			result += fmt.Sprintf("  Key: %s\n", diff.Key)
			result += fmt.Sprintf("  Local Value: %s\n", diff.LocalValue)
			result += fmt.Sprintf("  Remote Value: %s\n", diff.RemoteValue)
		}

		result += "\n"
	}

	if result == "" {
		result = "No differences found.\n"
	}

	return result
}

// CompareAirflowConfigs compares the local Airflow configuration to a remote configuration map.
// It returns a list of Diff objects that describe the differences between the two configurations.
func CompareAirflowConfigs(remoteConfig map[string]string) (Diffs, error) {
	// Define the local config file path
	cfgFilePath := filepath.Join(DefaultClonePath, "docker", "config", "airflow.cfg")

	// Load the local configuration file into a map
	localConfig, err := ConvertAirflowCfgToMap(cfgFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load local configuration file at '%s': %v", cfgFilePath, err)
	}

	// Initialize slices to store different types of differences
	var missingLocal, missingRemote, diffs Diffs

	// Compare keys present in the local config but missing or different in the remote config
	for key, localValue := range localConfig {
		localValue = strings.TrimSpace(localValue)
		if localValue == "" {
			// Skip keys with empty local values
			continue
		}

		if remoteValue, exists := remoteConfig[key]; !exists {
			// Key is missing in remote config
			missingRemote = append(missingRemote, Diff{
				Key:        key,
				Type:       "missing",
				LocalValue: localValue,
			})
		} else if localValue != remoteValue {
			// Key value is different between local and remote config
			diffs = append(diffs, Diff{
				Key:         key,
				Type:        "different",
				LocalValue:  localValue,
				RemoteValue: remoteValue,
			})
		}
	}

	// Check for keys in remoteConfig that are missing in localConfig
	for key, remoteValue := range remoteConfig {
		remoteValue = strings.TrimSpace(remoteValue)
		if remoteValue == "" {
			// Skip keys with empty remote values
			continue
		}

		if localValue, exists := localConfig[key]; !exists || localValue == "" {
			// Key is missing in local config or local value is empty
			missingLocal = append(missingLocal, Diff{
				Key:         key,
				Type:        "missing",
				RemoteValue: remoteValue,
			})
		}
	}

	// Combine all differences (missing local, missing remote, and diffs)
	allDiffs := append(append(missingLocal, missingRemote...), diffs...)

	// Sort diffs by category order: missing local -> missing remote -> diffs
	sort.Slice(allDiffs, func(i, j int) bool {
		// First, prioritize missing local, then missing remote, and finally diffs
		if allDiffs[i].Type != allDiffs[j].Type {
			return allDiffs[i].Type < allDiffs[j].Type
		}

		// If types are the same, sort by the key
		return allDiffs[i].Key < allDiffs[j].Key
	})

	return allDiffs, nil
}
