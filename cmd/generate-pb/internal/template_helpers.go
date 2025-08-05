package internal

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/quickfixgo/quickfix/datadictionary"
)

// Global enum registry
var globalEnumRegistry *EnumRegistry

// InitializeEnumRegistry initializes the global enum registry with data from specifications
func InitializeEnumRegistry(specs []*datadictionary.DataDictionary) {
	globalEnumRegistry = NewEnumRegistry()
	globalEnumRegistry.RegisterFieldEnums(specs)
}

// Template helper functions for protobuf generation

// toProtoType converts FIX field types to protobuf types, with enum support
func toProtoType(fixType string) string {
	// Check if this field type has enum values
	if globalEnumRegistry != nil && globalEnumRegistry.HasEnum(fixType) {
		if enum, exists := globalEnumRegistry.GetEnum(fixType); exists {
			return enum.ProtoName
		}
	}

	// If we have a field type name, try to get its base type
	baseType := fixType
	if globalFieldType, ok := globalFieldTypesLookup[fixType]; ok {
		baseType = getBaseFieldType(globalFieldType)

		// Check if the base type has enum values
		if globalEnumRegistry != nil && globalEnumRegistry.HasEnum(baseType) {
			if enum, exists := globalEnumRegistry.GetEnum(baseType); exists {
				return enum.ProtoName
			}
		}
	}

	switch strings.ToUpper(baseType) {
	case "INT", "SEQNUM", "NUMINGROUP", "DAYOFMONTH":
		return "int32"
	case "LENGTH", "TAGNUM":
		return "uint32"
	case "FLOAT":
		return "double"
	case "PRICE", "PRICEOFFSET", "QTY", "PERCENTAGE", "AMT":
		return "string" // Use string for decimal types to preserve precision
	case "CHAR":
		return "string" // Single char as string in proto
	case "BOOLEAN":
		return "bool"
	case "STRING", "MULTIPLEVALUESTRING", "MULTIPLESTRINGVALUE", "MULTIPLECHARVALUE",
		"CURRENCY", "EXCHANGE", "COUNTRY", "UTCTIMEONLY", "UTCDATEONLY", "UTCTIMESTAMP",
		"LOCALMKTDATE", "TZTIMEONLY", "TZTIMESTAMP", "DATA", "XMLDATA":
		return "string"
	default:
		return "string" // Default to string for unknown types
	}
}

// isEnumType checks if a field type is an enum type
func isEnumType(fieldType *datadictionary.FieldType) bool {
	if globalEnumRegistry == nil {
		return false
	}
	return globalEnumRegistry.HasEnum(fieldType.Name())
}

// getEnumDefinition returns the enum definition for a field type
func getEnumDefinition(fieldTypeName string) (*EnumDefinition, error) {
	if globalEnumRegistry == nil {
		return nil, fmt.Errorf("enum registry not initialized")
	}
	enum, exists := globalEnumRegistry.GetEnum(fieldTypeName)
	if !exists {
		return nil, fmt.Errorf("enum definition not found for field type: %s", fieldTypeName)
	}
	return enum, nil
}

// hasEnumDefinition checks if an enum definition exists for a field type
func hasEnumDefinition(fieldTypeName string) bool {
	if globalEnumRegistry == nil {
		return false
	}
	return globalEnumRegistry.HasEnum(fieldTypeName)
}

// getAllEnumDefinitions returns all enum definitions
func getAllEnumDefinitions() []*EnumDefinition {
	if globalEnumRegistry == nil {
		return nil
	}
	return globalEnumRegistry.GetAllEnums()
}

// sanitizeProtoFieldName ensures field names are valid for protobuf (converts CamelCase to snake_case)
func sanitizeProtoFieldName(name string) string {
	// Convert CamelCase to snake_case for protobuf field names
	// Handle special cases like URLLink -> url_link, not u_r_l_link
	var result strings.Builder
	runes := []rune(name)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Check if current character is uppercase
		if r >= 'A' && r <= 'Z' {
			// Add underscore before uppercase letter if:
			// 1. It's not the first character AND
			// 2. The previous character was lowercase OR
			// 3. The next character is lowercase (end of acronym)
			if i > 0 {
				prevIsLower := runes[i-1] >= 'a' && runes[i-1] <= 'z'
				nextIsLower := i < len(runes)-1 && runes[i+1] >= 'a' && runes[i+1] <= 'z'

				if prevIsLower || nextIsLower {
					result.WriteByte('_')
				}
			}

			// Convert to lowercase
			result.WriteRune(r - ('A' - 'a'))
		} else {
			result.WriteRune(r)
		}
	}

	// Handle any remaining replacements
	finalResult := result.String()
	finalResult = strings.ReplaceAll(finalResult, " ", "_")
	finalResult = strings.ReplaceAll(finalResult, "-", "_")

	return finalResult
}

// add function for template arithmetic
func add(a, b int) int {
	return a + b
}

// getRequiredFields returns required fields from a MessageDef, sorted by field name
func getRequiredFields(msgDef *datadictionary.MessageDef) []*datadictionary.FieldDef {
	var requiredFields []*datadictionary.FieldDef
	for tag := range msgDef.RequiredTags {
		if field, ok := msgDef.Fields[tag]; ok {
			requiredFields = append(requiredFields, field)
		}
	}

	// Sort required fields by field name for consistent ordering
	sort.Slice(requiredFields, func(i, j int) bool {
		return requiredFields[i].FieldType.Name() < requiredFields[j].FieldType.Name()
	})

	return requiredFields
}

// getOptionalFields returns optional fields from a MessageDef, sorted by field name
func getOptionalFields(msgDef *datadictionary.MessageDef) []*datadictionary.FieldDef {
	var optionalFields []*datadictionary.FieldDef
	for tag, field := range msgDef.Fields {
		if _, isRequired := msgDef.RequiredTags[tag]; !isRequired {
			optionalFields = append(optionalFields, field)
		}
	}

	// Sort optional fields by field name for consistent ordering
	sort.Slice(optionalFields, func(i, j int) bool {
		return optionalFields[i].FieldType.Name() < optionalFields[j].FieldType.Name()
	})

	return optionalFields
}

// getFieldType returns the field type for a field definition (template safe version)
func getFieldType(fieldDef *datadictionary.FieldDef) *datadictionary.FieldType {
	fieldType, err := getGlobalFieldType(fieldDef)
	if err != nil {
		// Return nil if field type not found, template will skip it
		return nil
	}
	return fieldType
}

// extractPackageName extracts the package name from go_package path
func extractPackageName(goPackagePath string) string {
	parts := strings.Split(goPackagePath, "/")
	if len(parts) == 0 {
		return "proto"
	}
	return parts[len(parts)-1]
}

// getRequiredComponents returns required component references from a MessageDef, sorted by name
func getRequiredComponents(msgDef *datadictionary.MessageDef) []componentReference {
	var requiredComponents []componentReference
	for _, part := range msgDef.Parts {
		if component, ok := part.(datadictionary.Component); ok && part.Required() {
			requiredComponents = append(requiredComponents, componentReference{
				Name:     component.ComponentType.Name(),
				Required: true,
			})
		}
	}

	// Sort required components by name for consistent ordering
	sort.Slice(requiredComponents, func(i, j int) bool {
		return requiredComponents[i].Name < requiredComponents[j].Name
	})

	return requiredComponents
}

// getOptionalComponents returns optional component references from a MessageDef, sorted by name
func getOptionalComponents(msgDef *datadictionary.MessageDef) []componentReference {
	var optionalComponents []componentReference
	for _, part := range msgDef.Parts {
		if component, ok := part.(datadictionary.Component); ok && !part.Required() {
			optionalComponents = append(optionalComponents, componentReference{
				Name:     component.ComponentType.Name(),
				Required: false,
			})
		}
	}

	// Sort optional components by name for consistent ordering
	sort.Slice(optionalComponents, func(i, j int) bool {
		return optionalComponents[i].Name < optionalComponents[j].Name
	})

	return optionalComponents
}

// componentReference represents a reference to a component in a message
type componentReference struct {
	Name     string
	Required bool
}

// getAllGroups returns all group fields from a MessageDef, sorted by field name
func getAllGroups(msgDef *datadictionary.MessageDef) []*datadictionary.FieldDef {
	var allGroups []*datadictionary.FieldDef
	for _, field := range msgDef.Fields {
		if field.IsGroup() {
			allGroups = append(allGroups, field)
		}
	}

	// Sort all groups by field name for consistent ordering
	sort.Slice(allGroups, func(i, j int) bool {
		return allGroups[i].FieldType.Name() < allGroups[j].FieldType.Name()
	})

	return allGroups
}

// generateGroupMessageName generates a protobuf message name for a group
func generateGroupMessageName(groupField *datadictionary.FieldDef) string {
	// Group名称通常以"No"开头，表示数量字段，我们生成对应的条目消息名
	groupName := groupField.FieldType.Name()
	if strings.HasPrefix(groupName, "No") {
		// 例如：NoAllocs -> AllocGroup
		return strings.TrimPrefix(groupName, "No") + "Group"
	}
	// 如果不是以"No"开头，直接添加"Group"后缀
	return groupName + "Group"
}

// dict creates a new map for template use
func dict(values ...interface{}) map[string]interface{} {
	if len(values)%2 != 0 {
		panic("dict requires an even number of arguments")
	}
	dict := make(map[string]interface{})
	for i := 0; i < len(values); i += 2 {
		key := fmt.Sprintf("%v", values[i])
		dict[key] = values[i+1]
	}
	return dict
}

// hasKey checks if a key exists in a map
func hasKey(dict map[string]interface{}, key string) bool {
	_, exists := dict[key]
	return exists
}

// set sets a key-value pair in a map
func set(dict map[string]interface{}, key string, value interface{}) string {
	dict[key] = value
	return "" // Return empty string since this is used for side effects only
}

var templateFuncs = template.FuncMap{
	"toProtoType":              toProtoType,
	"sanitizeProtoFieldName":   sanitizeProtoFieldName,
	"add":                      add,
	"getRequiredFields":        getRequiredFields,
	"getOptionalFields":        getOptionalFields,
	"getFieldType":             getFieldType,
	"extractPackageName":       extractPackageName,
	"getRequiredComponents":    getRequiredComponents,
	"getOptionalComponents":    getOptionalComponents,
	"getAllGroups":             getAllGroups,
	"generateGroupMessageName": generateGroupMessageName,
	"getAllEnumDefinitions":    getAllEnumDefinitions,
	"isEnumType":               isEnumType,
	"hasEnumDefinition":        hasEnumDefinition,
	"dict":                     dict,
	"hasKey":                   hasKey,
	"set":                      set,
}
