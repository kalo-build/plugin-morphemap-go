package mapdef

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadMapsFromDirectory loads all .map files from a directory (recursively).
func LoadMapsFromDirectory(dirPath string) ([]MorpheMap, error) {
	var maps []MorpheMap

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !isMapFile(path) {
			return nil
		}

		m, loadErr := LoadMapFromFile(path)
		if loadErr != nil {
			return fmt.Errorf("failed to load map from %s: %w", path, loadErr)
		}
		maps = append(maps, *m)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return maps, nil
}

// LoadMapFromFile loads a single .map file.
func LoadMapFromFile(filePath string) (*MorpheMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var m MorpheMap
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", filePath, err)
	}

	if m.Name == "" {
		return nil, fmt.Errorf("map in %s is missing required 'name' field", filePath)
	}

	if len(m.Aliases) == 0 {
		return nil, fmt.Errorf("map %q in %s is missing required 'aliases' field", m.Name, filePath)
	}

	return &m, nil
}

// isMapFile checks if a file path has a .map extension.
func isMapFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".map")
}
