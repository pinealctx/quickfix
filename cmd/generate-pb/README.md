# Generate-PB Tool

This tool converts FIX message definitions to Protocol Buffer (protobuf) definitions.

## Purpose

The `generate-pb` tool is designed to:
1. Convert FIX message definitions from data dictionaries to protobuf format
2. Generate enum definitions for FIX fields that support string values
3. Create Go extension functions to convert between protobuf enum values (numbers) and FIX string values
4. **Generate all message definitions in a single protobuf file for easier management**

## Usage

```bash
generate-pb [flags] <path to data dictionary> [additional dictionaries...]
```

### Flags

- `-go_package <prefix>`: Specify the Go package prefix for generated protobuf files (default: "github.com/quickfixgo/quickfix/proto")
- `-directory <path>`: Directory to write generated proto files to (default: ".")
- `-go_directory <path>`: Directory to write generated Go files to (default: same as -directory)

### Examples

```bash
# Generate to current directory (proto and Go files together)
generate-pb spec/FIX44.xml

# Generate proto files to ./proto, Go files to same directory
generate-pb -directory ./proto spec/FIX44.xml

# Separate proto and Go files to different directories
generate-pb -directory ./proto -go_directory ./go spec/FIX44.xml

# Custom go_package with separate directories
generate-pb -go_package "github.com/mycompany/trading/proto" \
            -directory ./proto \
            -go_directory ./internal/enums \
            spec/FIX44.xml
```

## Features

### Unified Message File
- **All message definitions are generated in a single `messages.proto` file**
- Easier to manage and deploy than multiple separate files
- Cleaner import structure

### Protobuf Generation
- Converts FIX messages to protobuf message definitions
- Maps FIX field types to appropriate protobuf types
- Handles enum fields specially since protobuf enums only support numeric values

### Configurable Go Package
- Supports custom Go package paths via `-go_package` flag
- Automatically generates proper package imports
- Maintains consistent package structure across generated files

### Enum Handling
Since FIX enums can have string values but protobuf enums only support numeric values, the tool:
1. Generates numeric enum definitions in protobuf files
2. Creates Go extension functions for string-to-enum and enum-to-string conversion
3. Maintains mapping between FIX string values and protobuf numeric values
4. Handles enum name conflicts (case-insensitive) automatically

### Output Files
- `enums.proto` - Contains all enum definitions (written to `-directory`)
- `messages.proto` - **Single file containing all message definitions** (written to `-directory`)
- `enum_extensions.go` - Go functions for enum string conversion (written to `-go_directory`)

### Separate Output Directories
The tool supports separating protobuf files and Go files into different directories:
- **Proto files** (`.proto`): Use `-directory` to specify where to place `enums.proto` and `messages.proto`
- **Go files** (`.go`): Use `-go_directory` to specify where to place `enum_extensions.go`
- If `-go_directory` is not specified, Go files will be placed in the same directory as proto files

This separation is useful when:
- Proto files need to be in a `proto/` directory for protoc compilation
- Go files need to be in a specific Go package directory
- You want to keep generated files organized by type

## Example Output

### Generated File Structure

```bash
generate-pb -directory ./output spec/FIX44.xml
```

Output directory will contain:
```
output/
├── enums.proto           # All enum definitions
├── messages.proto        # All message definitions in one file
└── enum_extensions.go    # Enum string conversion functions
```

### Separate Directories Example

```bash
generate-pb -directory ./proto -go_directory ./internal/enums spec/FIX44.xml
```

This will create:
```
proto/
├── enums.proto           # All enum definitions
└── messages.proto        # All message definitions

internal/enums/
└── enum_extensions.go    # Enum string conversion functions
```

### Sample messages.proto

```protobuf
syntax = "proto3";

package fixmessages;

option go_package = "github.com/quickfixgo/quickfix/proto/fixmessages";

import "enums.proto";

// NewOrderSingle message definition (from fix44 specification)
message NewOrderSingle {
  string cl_ord_id = 1; // Required
  fixenums.SIDE_ENUM side = 2; // Required
  double order_qty = 3; // Required
  string symbol = 4; // Optional
}

// ExecutionReport message definition (from fix44 specification)
message ExecutionReport {
  string order_id = 1; // Required
  string exec_id = 2; // Required
  fixenums.EXEC_TYPE_ENUM exec_type = 3; // Required
  fixenums.ORD_STATUS_ENUM ord_status = 4; // Required
}
```
