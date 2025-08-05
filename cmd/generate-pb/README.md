# Generate-PB Tool

This tool converts FIX message definitions to Protocol Buffer (protobuf) definitions and generates bidirectional conversion functions.

## Purpose

The `generate-pb` tool is designed to:
1. Convert FIX message definitions from data dictionaries to protobuf format
2. Generate enum definitions for FIX fields that support string values
3. Create Go extension functions to convert between protobuf enum values (numbers) and FIX string values
4. **Generate bidirectional conversion functions between protobuf and QuickFIX structs**
5. **Generate all message definitions in a single protobuf file for easier management**

## Usage

```bash
generate-pb [required-flags] <path to data dictionary> [additional dictionaries...]
```

### Required Flags (All Must Be Specified)

- `-pb_go_pkg <package>`: Go package for generated protobuf files (used in go_package option)
- `-pb_root <path>`: Directory for generated proto files
- `-go_root <path>`: Directory for generated Go files
- `-fix_pkg <package>`: Root import path for QuickFIX packages

### Examples

```bash
# Basic usage with all required flags
generate-pb -pb_go_pkg github.com/mycompany/proto \
            -pb_root ./proto \
            -go_root ./internal/proto \
            -fix_pkg github.com/mycompany/quickfix \
            spec/FIX44.xml

# Using original quickfix package
generate-pb -pb_go_pkg github.com/quickfixgo/quickfix/proto \
            -pb_root ./output/proto \
            -go_root ./output/go \
            -fix_pkg github.com/quickfixgo/quickfix \
            spec/FIX44.xml

# Multiple dictionary files
generate-pb -pb_go_pkg github.com/trading/proto \
            -pb_root ./proto \
            -go_root ./internal/proto \
            -fix_pkg github.com/trading/quickfix \
            spec/FIX44.xml spec/FIXT11.xml
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

### Component and Group Support
- **Components**: XML中的component元素被转换为独立的protobuf消息，避免信息缺失
- **Groups (重复组)**: FIX协议中的重复组转换为repeated字段和独立的group消息定义
- **去重处理**: 同一个component或group在多个消息中被引用时，只生成一次定义

### Conversion Functions (Always Generated!)
The tool automatically generates:
- **Bidirectional conversion functions** between protobuf structs and QuickFIX Go structs
- **Enum conversions**: Automatic handling of enum value mapping between string (QuickFIX) and numeric (protobuf) representations
- **Group conversions**: Support for converting repeating groups between both formats
- **Component conversions**: Full support for component conversion with proper field mapping

#### Conversion Features:
- `MessageToProto(quickfix.Messagable) *ProtoMessage` - Convert QuickFIX message to protobuf
- `MessageFromProto(*ProtoMessage) quickfix.Message` - Convert protobuf message to QuickFIX
- **Type-safe conversions** with proper error handling
- **Zero-value handling** to avoid unnecessary field assignments
- **Enum string mapping** using existing enum extension functions

### Configurable Packages and Paths
- **Protobuf Go Package**: Specified via `-pb_go_pkg` flag
- **Separate Output Directories**: Proto files (`-pb_root`) and Go files (`-go_root`) can be in different locations
- **Custom QuickFIX Package**: Specified via `-fix_pkg` flag for flexible project structures

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

### With Conversion Functions Example

```bash
generate-pb -conversions -directory ./proto -go_directory ./internal/proto spec/FIX44.xml
```

This will create:
```
proto/
├── enums.proto           # All enum definitions
└── messages.proto        # All message definitions

internal/proto/
├── enum_extensions.go    # Enum string conversion functions  
└── conversions.go        # Bidirectional conversion functions (NEW!)
```

### Sample conversions.go

```go
package proto

import (
	"strconv"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/field"
	"github.com/quickfixgo/quickfix/enum"
)

// NewOrderSingleToProto converts QuickFIX NewOrderSingle message to protobuf
func NewOrderSingleToProto(msg quickfix.Messagable) *NewOrderSingle {
	proto := &NewOrderSingle{}
	
	// Convert field: ClOrdID
	if fieldValue := msg.GetString(11); fieldValue != "" {
		proto.ClOrdId = fieldValue
	}
	
	// Convert field: Side
	if fieldValue := msg.GetString(54); fieldValue != "" {
		if enumValue := StringToSide(fieldValue); enumValue != SIDE_ENUM_UNSPECIFIED {
			proto.Side = enumValue
		}
	}
	
	// Convert field: OrderQty
	if fieldValue := msg.GetString(38); fieldValue != "" {
		proto.OrderQty = convertQuickFixToDouble(fieldValue)
	}
	
	return proto
}

// NewOrderSingleFromProto converts protobuf NewOrderSingle to QuickFIX message
func NewOrderSingleFromProto(proto *NewOrderSingle) quickfix.Message {
	msg := quickfix.NewMessage()
	
	// Convert field from proto: ClOrdID
	if proto.ClOrdId != "" {
		msg.SetString(11, proto.ClOrdId)
	}
	
	// Convert field from proto: Side
	if proto.Side != SIDE_ENUM_UNSPECIFIED {
		msg.SetString(54, SideToString(proto.Side))
	}
	
	// Convert field from proto: OrderQty
	if proto.OrderQty != 0.0 {
		msg.SetString(38, convertProtoToDouble(proto.OrderQty))
	}
	
	return msg
}
```

## Usage Examples

### Basic Protobuf Generation
```go
// Generate only protobuf definitions
go run cmd/generate-pb/generate-pb.go spec/FIX44.xml
```

### With Conversion Functions
```go
// Generate protobuf definitions + conversion functions
go run cmd/generate-pb/generate-pb.go -conversions spec/FIX44.xml
```

### Using Generated Conversion Functions
```go
package main

import (
	"github.com/quickfixgo/quickfix"
	"your-project/proto" // Your generated proto package
)

func main() {
	// Create a QuickFIX message
	quickfixMsg := quickfix.NewMessage()
	quickfixMsg.SetString(11, "ORDER123")  // ClOrdID
	quickfixMsg.SetString(54, "1")         // Side = Buy
	quickfixMsg.SetString(38, "100.0")     // OrderQty
	
	// Convert QuickFIX to Protobuf
	protoMsg := proto.NewOrderSingleToProto(quickfixMsg)
	
	// Use protobuf message...
	// (serialize, send over network, etc.)
	
	// Convert back to QuickFIX
	convertedMsg := proto.NewOrderSingleFromProto(protoMsg)
}
```
