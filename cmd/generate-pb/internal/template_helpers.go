package internal

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/quickfixgo/quickfix/datadictionary"
)

// Template helper functions for protobuf generation

// toProtoType converts FIX field types to protobuf types, using base type resolution
func toProtoType(fixType string) string {
	// If we have a field type name, try to get its base type
	baseType := fixType
	if globalFieldType, ok := globalFieldTypesLookup[fixType]; ok {
		baseType = getBaseFieldType(globalFieldType)
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

// sanitizeProtoFieldName ensures field names are valid for protobuf
func sanitizeProtoFieldName(name string) string {
	// Convert to snake_case and ensure it's a valid proto field name
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")
	return result
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
	"dict":                     dict,
	"hasKey":                   hasKey,
	"set":                      set,
}
