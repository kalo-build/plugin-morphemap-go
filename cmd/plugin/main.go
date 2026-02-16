package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	rcfg "github.com/kalo-build/morphe-go/pkg/registry/cfg"
	"github.com/kalo-build/plugin-morphemap-go/pkg/compile"
)

// PluginConfig represents the configuration passed to the plugin by Kalo CLI
type PluginConfig struct {
	// Store-based paths (mounted by CLI)
	Stores map[string]StoreConfig `json:"stores,omitempty"`

	// Legacy direct paths (for backward compatibility)
	InputPath  string `json:"inputPath,omitempty"`
	OutputPath string `json:"outputPath,omitempty"`

	// Plugin-specific config
	Config  map[string]interface{} `json:"config,omitempty"`
	Verbose bool                   `json:"verbose,omitempty"`
}

// StoreConfig represents a store configuration from Kalo CLI
type StoreConfig struct {
	ID        uint32 `json:"id"`
	Type      string `json:"type"`
	MountPath string `json:"mountPath,omitempty"`
}

// Exit codes
const (
	ExitSuccess         = 0
	ExitCompileFailed   = 1
	ExitMissingConfig   = 3
	ExitInvalidConfig   = 4
	ExitInputPathError  = 12
	ExitOutputPathError = 13
	ExitPackagePath     = 14
)

// logInfo prints info messages only when verbose mode is enabled
func logInfo(verbose bool, format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: plugin-morphemap-go <config>")
		fmt.Fprintln(os.Stderr, "  config: JSON string with store configurations")
		os.Exit(ExitMissingConfig)
	}

	// Parse configuration
	rawConfig := os.Args[1]
	var config PluginConfig
	if err := json.Unmarshal([]byte(rawConfig), &config); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing config JSON:", err)
		os.Exit(ExitInvalidConfig)
	}

	// Determine paths - prefer store mounts, fall back to legacy paths
	var mapsPath, registryPath, externalPath, outputPath string

	if config.Stores != nil {
		for _, store := range config.Stores {
			switch store.MountPath {
			case "/maps":
				mapsPath = "/maps"
			case "/registry":
				registryPath = "/registry"
			case "/external":
				externalPath = "/external"
			case "/input":
				if mapsPath == "" {
					mapsPath = "/input"
				}
			case "/output":
				outputPath = "/output"
			}
		}
	}

	// Fall back to legacy paths
	if mapsPath == "" && config.InputPath != "" {
		mapsPath = config.InputPath
	}
	if outputPath == "" && config.OutputPath != "" {
		outputPath = config.OutputPath
	}

	// Validate required paths
	if mapsPath == "" {
		fmt.Fprintln(os.Stderr, "Error: maps input path is required (mount /maps store or provide inputPath)")
		os.Exit(ExitInputPathError)
	}
	if registryPath == "" {
		fmt.Fprintln(os.Stderr, "Error: registry path is required (mount /registry store)")
		os.Exit(ExitInputPathError)
	}
	if outputPath == "" {
		fmt.Fprintln(os.Stderr, "Error: output path is required (mount /output store or provide outputPath)")
		os.Exit(ExitOutputPathError)
	}

	// Extract config values
	verbose := config.Verbose

	packagePath := getStringConfig(config.Config, "packagePath")
	sourcePackagePath := getStringConfig(config.Config, "sourcePackagePath")
	targetPackagePath := getStringConfig(config.Config, "targetPackagePath")
	errorHandling := getStringConfig(config.Config, "errorHandling")

	if packagePath == "" {
		fmt.Fprintln(os.Stderr, "Error: packagePath is required in config")
		os.Exit(ExitPackagePath)
	}
	if sourcePackagePath == "" {
		fmt.Fprintln(os.Stderr, "Error: sourcePackagePath is required in config")
		os.Exit(ExitPackagePath)
	}
	if targetPackagePath == "" {
		fmt.Fprintln(os.Stderr, "Error: targetPackagePath is required in config")
		os.Exit(ExitPackagePath)
	}

	if errorHandling == "" {
		errorHandling = "return"
	}

	logInfo(verbose, "Maps path: %s", mapsPath)
	logInfo(verbose, "Registry path: %s", registryPath)
	if externalPath != "" {
		logInfo(verbose, "External path: %s", externalPath)
	}
	logInfo(verbose, "Output path: %s", outputPath)
	logInfo(verbose, "Package path: %s", packagePath)

	// Build registry configs
	registryConfig := rcfg.MorpheLoadRegistryConfig{
		RegistryModelsDirPath:     filepath.Join(registryPath, "models"),
		RegistryEntitiesDirPath:   filepath.Join(registryPath, "entities"),
		RegistryEnumsDirPath:      filepath.Join(registryPath, "enums"),
		RegistryStructuresDirPath: filepath.Join(registryPath, "structures"),
	}

	var externalConfig *rcfg.MorpheLoadRegistryConfig
	if externalPath != "" {
		externalConfig = &rcfg.MorpheLoadRegistryConfig{
			RegistryModelsDirPath:     filepath.Join(externalPath, "models"),
			RegistryEntitiesDirPath:   filepath.Join(externalPath, "entities"),
			RegistryEnumsDirPath:      filepath.Join(externalPath, "enums"),
			RegistryStructuresDirPath: filepath.Join(externalPath, "structures"),
		}
	}

	compileConfig := compile.GoConverterConfig{
		MapsPath:          mapsPath,
		RegistryConfig:    registryConfig,
		ExternalConfig:    externalConfig,
		OutputPath:        outputPath,
		PackagePath:       packagePath,
		SourcePackagePath: sourcePackagePath,
		TargetPackagePath: targetPackagePath,
		ErrorHandling:     compile.ErrorHandling(errorHandling),
	}

	logInfo(verbose, "Starting Go converter generation...")
	if err := compile.MorpheMapToGo(compileConfig); err != nil {
		fmt.Fprintln(os.Stderr, "Go converter generation failed:", err)
		os.Exit(ExitCompileFailed)
	}

	logInfo(verbose, "Go converter generation completed successfully")
	os.Exit(ExitSuccess)
}

func getStringConfig(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	if v, ok := config[key].(string); ok {
		return v
	}
	return ""
}
