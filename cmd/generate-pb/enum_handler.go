package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/quickfixgo/quickfix/datadictionary"
)

// EnumValue represents a single enum value with its string value and assigned integer
type EnumValue struct {
	StringValue  string
	IntegerValue int32
	Description  string
	OriginalEnum string // The original enum value from FIX spec
}

// EnumDefinition represents a complete enum type definition
type EnumDefinition struct {
	Name      string
	FieldType string
	Values    []EnumValue
	ProtoName string // Sanitized name for protobuf
}

// EnumRegistry manages all enum definitions across specifications
type EnumRegistry struct {
	enums          map[string]*EnumDefinition // Key: field type name
	fieldTypeEnums map[string]bool            // Tracks which field types have enums
}

// NewEnumRegistry creates a new enum registry
func NewEnumRegistry() *EnumRegistry {
	return &EnumRegistry{
		enums:          make(map[string]*EnumDefinition),
		fieldTypeEnums: make(map[string]bool),
	}
}

// RegisterFieldEnums extracts and registers enum values from field definitions
func (r *EnumRegistry) RegisterFieldEnums(specs []*datadictionary.DataDictionary) {
	for _, spec := range specs {
		for _, field := range spec.FieldTypeByName {
			if len(field.Enums) > 0 {
				r.registerFieldEnum(field)
			}
		}
	}
}

// registerFieldEnum processes a single field type and its enum values
func (r *EnumRegistry) registerFieldEnum(field *datadictionary.FieldType) {
	enumDef := &EnumDefinition{
		Name:      field.Name(),
		FieldType: field.Type, // field.Type is a string field, not a method
		ProtoName: sanitizeEnumName(field.Name()),
		Values:    make([]EnumValue, 0, len(field.Enums)),
	}

	// Convert enum values to integers
	integerValue := int32(0) // Start from 0 for protobuf enums

	// Sort enum values for consistent ordering
	var sortedEnums []string
	for enumValue := range field.Enums {
		sortedEnums = append(sortedEnums, enumValue)
	}
	sort.Strings(sortedEnums)

	for _, enumValue := range sortedEnums {
		enumInfo := field.Enums[enumValue] // This is an Enum struct, not a string

		enumVal := EnumValue{
			StringValue:  enumValue,
			IntegerValue: integerValue,
			Description:  enumInfo.Description, // Use the Description field from Enum struct
			OriginalEnum: enumValue,
		}

		enumDef.Values = append(enumDef.Values, enumVal)
		integerValue++
	}

	r.enums[field.Name()] = enumDef
	r.fieldTypeEnums[field.Name()] = true
}

// GetEnum returns the enum definition for a field type
func (r *EnumRegistry) GetEnum(fieldTypeName string) (*EnumDefinition, bool) {
	enum, exists := r.enums[fieldTypeName]
	return enum, exists
}

// HasEnum checks if a field type has enum values
func (r *EnumRegistry) HasEnum(fieldTypeName string) bool {
	return r.fieldTypeEnums[fieldTypeName]
}

// GetAllEnums returns all registered enum definitions sorted by name
func (r *EnumRegistry) GetAllEnums() []*EnumDefinition {
	var enums []*EnumDefinition
	for _, enum := range r.enums {
		enums = append(enums, enum)
	}

	// Sort by proto name for consistent ordering
	sort.Slice(enums, func(i, j int) bool {
		return enums[i].ProtoName < enums[j].ProtoName
	})

	return enums
}

// sanitizeEnumName converts a FIX field name to a valid protobuf enum name
func sanitizeEnumName(name string) string {
	// Convert to PascalCase and ensure it's a valid protobuf enum name
	result := strings.ReplaceAll(name, " ", "")
	result = strings.ReplaceAll(result, "-", "")
	result = strings.ReplaceAll(result, "_", "")

	// Ensure first character is uppercase
	if len(result) > 0 && result[0] >= 'a' && result[0] <= 'z' {
		result = string(result[0]-('a'-'A')) + result[1:]
	}

	// 不添加"Type"后缀，直接使用原始名称
	return result
}

// sanitizeEnumValueName converts an enum value to a valid protobuf enum value name
func sanitizeEnumValueName(enumName, originalValue, description string) string {
	// 直接使用enum名称作为前缀，不需要移除"Type"后缀
	prefix := strings.ToUpper(enumName)

	// Use description if available and meaningful, otherwise fall back to original value
	var valueName string
	if description != "" && strings.TrimSpace(description) != "" {
		// Use the description to create meaningful enum value names
		valueName = strings.ToUpper(description)
		// Clean up the description for use as an enum value name
		valueName = strings.ReplaceAll(valueName, " ", "_")
		valueName = strings.ReplaceAll(valueName, "-", "_")
		valueName = strings.ReplaceAll(valueName, ".", "_")
		valueName = strings.ReplaceAll(valueName, "/", "_")
		valueName = strings.ReplaceAll(valueName, "(", "_")
		valueName = strings.ReplaceAll(valueName, ")", "_")
		valueName = strings.ReplaceAll(valueName, "+", "_PLUS")
		valueName = strings.ReplaceAll(valueName, "%", "_PERCENT")
		valueName = strings.ReplaceAll(valueName, "'", "")
		valueName = strings.ReplaceAll(valueName, "\"", "")
		valueName = strings.ReplaceAll(valueName, ",", "_")
		valueName = strings.ReplaceAll(valueName, ":", "_")
		valueName = strings.ReplaceAll(valueName, ";", "_")

		// Remove multiple consecutive underscores
		for strings.Contains(valueName, "__") {
			valueName = strings.ReplaceAll(valueName, "__", "_")
		}

		// Remove leading and trailing underscores
		valueName = strings.Trim(valueName, "_")

		// Ensure it doesn't start with a number
		if len(valueName) > 0 && valueName[0] >= '0' && valueName[0] <= '9' {
			valueName = "VALUE_" + valueName
		}
	} else {
		// Fall back to original value based naming
		valueName = strings.ToUpper(originalValue)
		valueName = strings.ReplaceAll(valueName, " ", "_")
		valueName = strings.ReplaceAll(valueName, "-", "_")
		valueName = strings.ReplaceAll(valueName, ".", "_")
		valueName = strings.ReplaceAll(valueName, "/", "_")
		valueName = strings.ReplaceAll(valueName, "+", "_PLUS")
		valueName = strings.ReplaceAll(valueName, "%", "_PERCENT")

		// Handle numeric values
		if len(valueName) > 0 && valueName[0] >= '0' && valueName[0] <= '9' {
			valueName = "VALUE_" + valueName
		}

		// Handle single characters
		if len(originalValue) == 1 {
			if originalValue[0] >= 'A' && originalValue[0] <= 'Z' || originalValue[0] >= 'a' && originalValue[0] <= 'z' {
				valueName = "CHAR_" + valueName
			}
		}
	}

	// Ensure the name is not empty
	if valueName == "" {
		valueName = "UNKNOWN"
	}

	return prefix + "_" + valueName
}

// GetProtoEnumValueName returns the protobuf enum value name for an enum value
func (ev *EnumValue) GetProtoEnumValueName(enumName string) string {
	return sanitizeEnumValueName(enumName, ev.OriginalEnum, ev.Description)
}

// GetDefaultValueName returns a default unspecified value name for the enum
func (ev *EnumValue) GetDefaultValueName() string {
	return "UNSPECIFIED"
}

// GenerateEnumStringMapping generates Go code for enum to string mapping
func (ed *EnumDefinition) GenerateEnumStringMapping() string {
	var builder strings.Builder

	enumTypePrefix := ed.ProtoName

	// Generate map for enum to string conversion
	builder.WriteString(fmt.Sprintf("// %sToFIX converts %s enum values to their FIX enum representation\n", ed.ProtoName, ed.ProtoName))
	builder.WriteString(fmt.Sprintf("var %sToFIX = map[%s]enum.%s{\n", ed.ProtoName, ed.ProtoName, ed.ProtoName))

	for _, value := range ed.Values {
		enumValueName := value.GetProtoEnumValueName(ed.ProtoName)
		goEnumValueName := enumTypePrefix + "_" + enumValueName
		builder.WriteString(fmt.Sprintf("\t%s: enum.%s_%s,\n", goEnumValueName, ed.ProtoName, value.Description))
	}

	builder.WriteString("}\n\n")

	// Generate map for string to enum conversion
	builder.WriteString(fmt.Sprintf("// FIXTo%s converts FIX enum values to %s enum values\n", ed.ProtoName, ed.ProtoName))
	builder.WriteString(fmt.Sprintf("var FIXTo%s = map[enum.%s]%s{\n", ed.ProtoName, ed.ProtoName, ed.ProtoName))

	for _, value := range ed.Values {
		enumValueName := value.GetProtoEnumValueName(ed.ProtoName)
		goEnumValueName := enumTypePrefix + "_" + enumValueName
		builder.WriteString(fmt.Sprintf("\tenum.%s_%s: %s,\n", ed.ProtoName, value.Description, goEnumValueName))
	}

	builder.WriteString("}\n\n")

	return builder.String()
}
