package main

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
	// First, check if this exact field type has enum values
	if globalEnumRegistry != nil {
		if enum, exists := globalEnumRegistry.GetEnum(fixType); exists {
			return enum.ProtoName
		}
	}

	// If we have a field type name, try to get its definition and check for enums
	if globalFieldType, ok := globalFieldTypesLookup[fixType]; ok {
		// Check if this field type has enum values
		if globalEnumRegistry != nil && len(globalFieldType.Enums) > 0 {
			if enum, exists := globalEnumRegistry.GetEnum(fixType); exists {
				return enum.ProtoName
			}
		}

		// Get the base type for non-enum fields
		baseType := getBaseFieldType(globalFieldType)

		// Check if the base type has enum values (for derived types)
		if globalEnumRegistry != nil && globalEnumRegistry.HasEnum(baseType) {
			if enum, exists := globalEnumRegistry.GetEnum(baseType); exists {
				return enum.ProtoName
			}
		}

		// Use the base type for mapping to protobuf primitive types
		fixType = baseType
	}

	// Map to protobuf primitive types
	switch strings.ToUpper(fixType) {
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

// getProtoTypeForField converts a field definition to the correct protobuf type
// This function handles both enum and non-enum fields correctly
func getProtoTypeForField(fieldDef *datadictionary.FieldDef) string {
	if fieldDef == nil {
		return "string"
	}

	fieldType, err := getGlobalFieldType(fieldDef)
	if err != nil {
		return "string"
	}

	// First check if this field has enum values by field name
	if globalEnumRegistry != nil {
		if enum, exists := globalEnumRegistry.GetEnum(fieldType.Name()); exists {
			return enum.ProtoName
		}
	}

	// If no enum found, use the base type mapping
	return toProtoType(fieldType.Type)
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

// protoFieldNameToGoFieldName converts snake_case to PascalCase for Go struct field names
// This tries to match protoc's own field name conversion rules
func protoFieldNameToGoFieldName(name string) string {
	// Handle special cases first
	specialCases := map[string]string{
		"rule80a": "Rule80A",
		// Add more special cases as needed
	}

	if goName, exists := specialCases[name]; exists {
		return goName
	}

	parts := strings.Split(name, "_")
	var result strings.Builder

	for _, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter and append rest
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}

	return result.String()
}

// add function for template arithmetic
func add(a, b int) int {
	return a + b
}

// getFields gets all fields from a MessageDef, sorted by field name
func getFields(msgDef *datadictionary.MessageDef) []*datadictionary.FieldDef {
	var fields []*datadictionary.FieldDef
	for _, field := range msgDef.Fields {
		fields = append(fields, field)
	}

	// Sort fields by field name for consistent ordering
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].FieldType.Name() < fields[j].FieldType.Name()
	})

	return fields
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

// getFixFieldValue generates code to get a field value from FIX message
func getFixFieldValue(fieldDef *datadictionary.FieldDef, msgVar string) string {
	return fmt.Sprintf("%s.Get%s()", msgVar, fieldDef.FieldType.Name())
}

// getFixGroupValue generates code to get a group from FIX message
func getFixGroupValue(groupField *datadictionary.FieldDef, msgVar string) string {
	// Use the quickfix Group method with the group tag to get repeating group
	return fmt.Sprintf("%s.Group(tag.%s)", msgVar, groupField.FieldType.Name())
}

// convertFixFieldToProto generates code to convert a FIX field to proto field
func convertFixFieldToProto(fieldDef *datadictionary.FieldDef, pbMsgVar, fixMsgVar string) string {
	fieldType, err := getGlobalFieldType(fieldDef)
	if err != nil {
		return "string(fieldValue)"
	}

	fieldName := fieldDef.FieldType.Name()

	// Check if field has enum values
	if globalEnumRegistry != nil && globalEnumRegistry.HasEnum(fieldName) {
		return fmt.Sprintf("Convert%sFromFIX(fieldValue)", fieldName)
	}

	// Handle different field types
	switch strings.ToUpper(fieldType.Type) {
	case "INT", "SEQNUM", "NUMINGROUP", "DAYOFMONTH":
		return "func() int32 { if v, e := strconv.Atoi(fieldValue); e == nil { return int32(v) }; return 0 }()"
	case "LENGTH", "TAGNUM":
		return "func() uint32 { if v, e := strconv.ParseUint(fieldValue, 10, 32); e == nil { return uint32(v) }; return 0 }()"
	case "FLOAT":
		return "func() float64 { if v, e := strconv.ParseFloat(fieldValue, 64); e == nil { return v }; return 0 }()"
	case "BOOLEAN":
		return "func() bool { return fieldValue == \"Y\" || fieldValue == \"true\" || fieldValue == \"1\" }()"
	case "PRICE", "PRICEOFFSET", "QTY", "PERCENTAGE", "AMT":
		return "fieldValue" // Keep as string to preserve precision
	default:
		return "fieldValue"
	}
}

// setProtoField generates code to set a proto field
func setProtoField(fieldDef *datadictionary.FieldDef, pbMsgVar, valueVar string) string {
	protoFieldName := sanitizeProtoFieldName(fieldDef.FieldType.Name())
	goFieldName := protoFieldNameToGoFieldName(protoFieldName)
	return fmt.Sprintf("%s.%s = %s", pbMsgVar, goFieldName, valueVar)
}

// convertProtoFieldToFix generates code to convert a proto field to FIX field
func convertProtoFieldToFix(fieldDef *datadictionary.FieldDef, fixMsgVar, pbMsgVar string) string {
	fieldType, err := getGlobalFieldType(fieldDef)
	if err != nil {
		return fmt.Sprintf("// Error: cannot convert field %s", fieldDef.FieldType.Name())
	}

	fieldName := fieldDef.FieldType.Name()
	protoFieldName := sanitizeProtoFieldName(fieldName)
	goFieldName := protoFieldNameToGoFieldName(protoFieldName)

	// Check if field has enum values
	if globalEnumRegistry != nil && globalEnumRegistry.HasEnum(fieldName) {
		return fmt.Sprintf(`if %s.%s != %s_UNSPECIFIED {
		%s.Set(field.New%s(Convert%sToFIX(%s.%s)))
	}`, pbMsgVar, goFieldName, getEnumProtoName(fieldName), fixMsgVar, fieldName, fieldName, pbMsgVar, goFieldName)
	}

	// Handle different field types
	switch strings.ToUpper(fieldType.Type) {
	case "INT", "SEQNUM", "NUMINGROUP", "DAYOFMONTH":
		return fmt.Sprintf(`if %s.%s != 0 {
		%s.Set(field.New%s(int(%s.%s)))
	}`, pbMsgVar, goFieldName, fixMsgVar, fieldName, pbMsgVar, goFieldName)
	case "LENGTH", "TAGNUM":
		return fmt.Sprintf(`if %s.%s != 0 {
		%s.Set(field.New%s(int(%s.%s)))
	}`, pbMsgVar, goFieldName, fixMsgVar, fieldName, pbMsgVar, goFieldName)
	case "FLOAT":
		return fmt.Sprintf(`if %s.%s != 0 {
		%s.Set(field.New%s(float64(%s.%s)))
	}`, pbMsgVar, goFieldName, fixMsgVar, fieldName, pbMsgVar, goFieldName)
	case "BOOLEAN":
		return fmt.Sprintf(`%s.Set(field.New%s(%s.%s))`, fixMsgVar, fieldName, pbMsgVar, goFieldName)
	case "PRICE", "PRICEOFFSET", "QTY", "PERCENTAGE", "AMT":
		return fmt.Sprintf(`if %s.%s != "" {
		if decimalValue, err := decimal.NewFromString(%s.%s); err == nil {
			%s.Set(field.New%s(decimalValue, 0))
		}
	}`, pbMsgVar, goFieldName, pbMsgVar, goFieldName, fixMsgVar, fieldName)
	default:
		return fmt.Sprintf(`if %s.%s != "" {
		%s.Set(field.New%s(%s.%s))
	}`, pbMsgVar, goFieldName, fixMsgVar, fieldName, pbMsgVar, goFieldName)
	}
}

// getEnumProtoName gets the protobuf enum name for a field
func getEnumProtoName(fieldName string) string {
	if globalEnumRegistry == nil {
		return fieldName
	}
	if enum, exists := globalEnumRegistry.GetEnum(fieldName); exists {
		return enum.ProtoName
	}
	return fieldName
}

// toGoFieldName converts a field name to Go struct field name (PascalCase)
func toGoFieldName(fieldName string) string {
	// Convert to protobuf field name first (snake_case), then to Go field name (PascalCase)
	protoName := sanitizeProtoFieldName(fieldName)
	return protoFieldNameToGoFieldName(protoName)
}

// hasEnumType checks if a field name has enum values defined
func hasEnumType(fieldName string) bool {
	if globalEnumRegistry == nil {
		return false
	}
	_, exists := globalEnumRegistry.GetEnum(fieldName)
	return exists
}

// getEnumTypeName gets the protobuf enum type name for a field name
func getEnumTypeName(fieldName string) string {
	if globalEnumRegistry == nil {
		return fieldName
	}
	if enum, exists := globalEnumRegistry.GetEnum(fieldName); exists {
		return enum.ProtoName
	}
	return fieldName
}

func getGoTypeForField(fieldName string) string {
	// If not found in our mapping, try some heuristics as fallback
	fieldNameUpper := strings.ToUpper(fieldName)

	// Simple heuristics for common patterns
	if strings.HasSuffix(fieldNameUpper, "LENGTH") || strings.HasSuffix(fieldNameUpper, "LEN") {
		return "int32"
	}
	if strings.HasSuffix(fieldNameUpper, "QTY") || strings.Contains(fieldNameUpper, "QUANTITY") {
		return "int32"
	}
	if strings.HasSuffix(fieldNameUpper, "PX") || strings.Contains(fieldNameUpper, "PRICE") {
		return "float64"
	}
	if strings.HasSuffix(fieldNameUpper, "AMT") || strings.Contains(fieldNameUpper, "AMOUNT") {
		return "float64"
	}
	if strings.Contains(fieldNameUpper, "PERCENT") || strings.Contains(fieldNameUpper, "RATE") {
		return "float64"
	}
	if strings.Contains(fieldNameUpper, "TIME") || strings.Contains(fieldNameUpper, "DATE") {
		if strings.Contains(fieldNameUpper, "TIMESTAMP") {
			return "time.Time"
		}
		if strings.Contains(fieldNameUpper, "DATE") {
			return "string" // LocalMktDate is often a string in Go
		}
		return "time.Time" // Default to UTC time only
	}
	if strings.Contains(fieldNameUpper, "FLAG") || strings.HasSuffix(fieldNameUpper, "IND") {
		return "bool"
	}
	if strings.Contains(fieldNameUpper, "SEQNUM") || strings.Contains(fieldNameUpper, "MSGSEQNUM") {
		return "int32"
	}
	if strings.Contains(fieldNameUpper, "ID") && (strings.HasSuffix(fieldNameUpper, "ID") || strings.HasPrefix(fieldNameUpper, "REF")) {
		return "string" // Most IDs are strings in FIX
	}

	// Default fallback - return string for unknown types
	return "string"
}

var templateFuncs = template.FuncMap{
	"toProtoType":                 toProtoType,
	"getProtoTypeForField":        getProtoTypeForField,
	"sanitizeProtoFieldName":      sanitizeProtoFieldName,
	"protoFieldNameToGoFieldName": protoFieldNameToGoFieldName,
	"toGoFieldName":               toGoFieldName,
	"hasEnumType":                 hasEnumType,
	"getEnumTypeName":             getEnumTypeName,
	"getGoTypeForField":           getGoTypeForField,
	"add":                         add,
	"getFields":                   getFields,
	"getRequiredFields":           getRequiredFields,
	"getOptionalFields":           getOptionalFields,
	"getFieldType":                getFieldType,
	"extractPackageName":          extractPackageName,
	"getRequiredComponents":       getRequiredComponents,
	"getOptionalComponents":       getOptionalComponents,
	"getAllGroups":                getAllGroups,
	"generateGroupMessageName":    generateGroupMessageName,
	"getAllEnumDefinitions":       getAllEnumDefinitions,
	"isEnumType":                  isEnumType,
	"hasEnumDefinition":           hasEnumDefinition,
	"dict":                        dict,
	"hasKey":                      hasKey,
	"set":                         set,
	"getFixFieldValue":            getFixFieldValue,
	"getFixGroupValue":            getFixGroupValue,
	"convertFixFieldToProto":      convertFixFieldToProto,
	"setProtoField":               setProtoField,
	"convertProtoFieldToFix":      convertProtoFieldToFix,
	"getEnumProtoName":            getEnumProtoName,
}
