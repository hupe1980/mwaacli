package local

import (
	"fmt"

	"gopkg.in/ini.v1"
)

// ConvertAirlfowCfgToMap reads an Airflow configuration file in INI format
// and converts it into a map of key-value pairs. The keys are formatted as
// "section.key" (e.g., "core.dag_dir_list_interval").
//
// The "DEFAULT" section in the INI file is ignored.
func ConvertAirlfowCfgToMap(filename string) (map[string]string, error) {
	// Load INI file
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Convert INI to a map
	configMap := make(map[string]string)

	for _, section := range cfg.Sections() {
		sectionName := section.Name()
		if sectionName == "DEFAULT" { // Ignore DEFAULT section
			continue
		}

		for _, key := range section.Keys() {
			configMap[fmt.Sprintf("%s.%s", sectionName, key.Name())] = key.Value()
		}
	}

	return configMap, nil
}
