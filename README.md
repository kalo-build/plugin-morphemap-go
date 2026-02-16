# plugin-morphemap-go

Kalo plugin that generates type-safe Go converter functions from MorpheMap (`.map`) definitions.

## Overview

Given MorpheMap files and Morphe type registries, this plugin generates Go functions that convert data between structurally different source and target types. Each `.map` file produces a corresponding `.go` file with:

- **Field maps**: A converter function that takes the source type and returns the target type, handling direct renames, path traversal, type coercion, enum translation, conditionals, and constants
- **Enum maps**: Helper functions for enum entry translation with exhaustive switch statements

This plugin is designed for **cross-domain structural mapping** -- converting between external API types and local domain models where field-level mapping decisions are captured in `.map` files.

## Input

- **MorpheMap files** (`KA:MM1:YAML1`): Transformation definitions
- **Local Morphe Registry** (`KA:MO1:YAML1`): Local project Morphe schema files
- **External Morphe Registry** (`KA:MO1:YAML1`, optional): Third-party API type definitions

## Output

- **Go files** (`KA:MM1:GO1`): Type-safe converter functions

## Configuration

```yaml
config:
  "@kalo-build/plugin-morphemap-go":
    packagePath: "github.com/myproject/internal/generated/converters"
    sourcePackagePath: "github.com/myproject/internal/types/external"
    targetPackagePath: "github.com/myproject/internal/types/models"
    errorHandling: "return"
```

### Config Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `packagePath` | string | Yes | - | Go package path for generated files |
| `sourcePackagePath` | string | Yes | - | Package path for source types (imports) |
| `targetPackagePath` | string | Yes | - | Package path for target types (imports) |
| `errorHandling` | string | No | `"return"` | Error strategy: `return`, `collect`, or `panic` |

## Build

```bash
cd scripts && bash build.sh
```

## License

MIT
