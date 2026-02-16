package compile

import (
	"fmt"

	"github.com/kalo-build/morphe-go/pkg/registry"
	rcfg "github.com/kalo-build/morphe-go/pkg/registry/cfg"
	"github.com/kalo-build/plugin-morphemap-go/pkg/mapdef"
)

// ErrorHandling defines the strategy for handling mapping errors.
type ErrorHandling string

const (
	ErrorHandlingReturn  ErrorHandling = "return"
	ErrorHandlingCollect ErrorHandling = "collect"
	ErrorHandlingPanic   ErrorHandling = "panic"
)

// GoConverterConfig holds the configuration for Go converter generation.
type GoConverterConfig struct {
	MapsPath          string
	RegistryConfig    rcfg.MorpheLoadRegistryConfig
	ExternalConfig    *rcfg.MorpheLoadRegistryConfig
	OutputPath        string
	PackagePath       string
	SourcePackagePath string
	TargetPackagePath string
	ErrorHandling     ErrorHandling
}

// MorpheMapToGo generates Go converter functions from MorpheMap definitions.
func MorpheMapToGo(config GoConverterConfig) error {
	// Load maps
	maps, err := mapdef.LoadMapsFromDirectory(config.MapsPath)
	if err != nil {
		return fmt.Errorf("failed to load maps: %w", err)
	}

	if len(maps) == 0 {
		return fmt.Errorf("no .map files found in %s", config.MapsPath)
	}

	// Load local Morphe registry
	localRegistry, err := registry.LoadMorpheRegistry(
		registry.LoadMorpheRegistryHooks{},
		config.RegistryConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to load local Morphe registry: %w", err)
	}

	// Load external registry if configured
	var externalRegistry *registry.Registry
	if config.ExternalConfig != nil {
		extReg, err := registry.LoadMorpheRegistry(
			registry.LoadMorpheRegistryHooks{},
			*config.ExternalConfig,
		)
		if err != nil {
			return fmt.Errorf("failed to load external Morphe registry: %w", err)
		}
		externalRegistry = extReg
	}

	// Build enum map index for discovery
	enumMapIndex := buildEnumMapIndex(maps, localRegistry, externalRegistry)

	// Generate a Go file for each field map
	for _, m := range maps {
		mapType := m.InferMapType()

		switch mapType {
		case mapdef.MapTypeField:
			if err := compileFieldMap(&m, localRegistry, externalRegistry, enumMapIndex, config); err != nil {
				return fmt.Errorf("failed to compile field map %q: %w", m.Name, err)
			}

		case mapdef.MapTypeEnum:
			// Enum maps generate helper functions for entry translation
			if err := compileEnumMap(&m, config); err != nil {
				return fmt.Errorf("failed to compile enum map %q: %w", m.Name, err)
			}
		}
	}

	return nil
}

// enumMapKey identifies an enum map by its (sourceEnum, targetEnum) pair.
type enumMapKey struct {
	SourceEnum string
	TargetEnum string
}

// buildEnumMapIndex indexes all enum maps by their (source, target) enum type pair.
func buildEnumMapIndex(maps []mapdef.MorpheMap, localReg, externalReg *registry.Registry) map[enumMapKey]*mapdef.MorpheMap {
	index := make(map[enumMapKey]*mapdef.MorpheMap)

	for i := range maps {
		m := &maps[i]
		if m.InferMapType() != mapdef.MapTypeEnum {
			continue
		}

		// Resolve alias types
		var sourceEnum, targetEnum string
		for alias, path := range m.Aliases {
			if isEnumInRegistry(path, localReg) || isEnumInRegistry(path, externalReg) {
				// Heuristic: first alias alphabetically is source, second is target
				// In practice, enum maps typically use consistent alias names
				if sourceEnum == "" {
					sourceEnum = path
				} else {
					targetEnum = path
				}
				_ = alias
			}
		}

		if sourceEnum != "" && targetEnum != "" {
			index[enumMapKey{SourceEnum: sourceEnum, TargetEnum: targetEnum}] = m
			// Also index in reverse for bidirectional lookup
			index[enumMapKey{SourceEnum: targetEnum, TargetEnum: sourceEnum}] = m
		}
	}

	return index
}

func isEnumInRegistry(path string, reg *registry.Registry) bool {
	if reg == nil {
		return false
	}
	for name := range reg.GetAllEnums() {
		if name == path {
			return true
		}
	}
	return false
}
