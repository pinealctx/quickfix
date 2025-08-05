package internal

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"unicode"

	"github.com/quickfixgo/quickfix/datadictionary"
)

// Template helper functions for protobuf generation

// enumKeyTracker tracks used enum keys to avoid conflicts (case-insensitive)
var (
	enumKeyTracker = make(map[string]string)
	enumKeyMutex   sync.RWMutex
)

// enumValueMapping tracks the mapping between original enum values and generated enum keys
var (
	enumValueMapping = make(map[string]map[string]string) // fieldName -> originalValue -> generatedKey
	enumMappingMutex sync.RWMutex
)

// toProtoType converts FIX field types to protobuf types
func toProtoType(fixType string) string {
	switch strings.ToUpper(fixType) {
	case "INT", "SEQNUM", "NUMINGROUP", "DAYOFMONTH":
		return "int32"
	case "LENGTH", "TAGNUM":
		return "uint32"
	case "FLOAT", "PRICE", "PRICEOFFSET", "QTY", "PERCENTAGE", "AMT":
		return "double"
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

// toProtoEnumName converts field name to protobuf enum name with conflict resolution
func toProtoEnumName(fieldName string) string {
	baseName := strings.ToUpper(fieldName) + "_ENUM"
	lowerKey := strings.ToLower(baseName)

	enumKeyMutex.RLock()
	// Check if this key already exists (case-insensitive)
	if existing, exists := enumKeyTracker[lowerKey]; exists {
		enumKeyMutex.RUnlock()
		if existing != baseName {
			// Find a unique suffix
			counter := 2
			for {
				newName := fmt.Sprintf("%s_%d", baseName, counter)
				newLowerKey := strings.ToLower(newName)
				enumKeyMutex.RLock()
				if _, exists := enumKeyTracker[newLowerKey]; !exists {
					enumKeyMutex.RUnlock()
					enumKeyMutex.Lock()
					enumKeyTracker[newLowerKey] = newName
					enumKeyMutex.Unlock()
					return newName
				}
				enumKeyMutex.RUnlock()
				counter++
			}
		}
		return existing
	}
	enumKeyMutex.RUnlock()

	enumKeyMutex.Lock()
	enumKeyTracker[lowerKey] = baseName
	enumKeyMutex.Unlock()
	return baseName
}

// toProtoEnumValue converts enum value to valid protobuf enum value
func toProtoEnumValue(enumVal string, index int) string {
	// Try to convert string to number for protobuf enum value
	if val, err := strconv.Atoi(enumVal); err == nil {
		return strconv.Itoa(val)
	}
	// If not a number, use index-based numbering
	return strconv.Itoa(index)
}

// sanitizeProtoFieldName ensures field names are valid for protobuf
func sanitizeProtoFieldName(name string) string {
	// Convert to snake_case and ensure it's a valid proto field name
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")
	return result
}

// sanitizeEnumValue ensures enum values are valid and unique
func sanitizeEnumValue(enumName, enumVal string) string {
	// Remove invalid characters and ensure it starts with letter or underscore
	sanitized := strings.ToUpper(enumVal)
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")

	// Ensure it starts with letter or underscore
	if len(sanitized) > 0 && !unicode.IsLetter(rune(sanitized[0])) && sanitized[0] != '_' {
		sanitized = "VAL_" + sanitized
	}

	return enumName + "_" + sanitized
}

// hasEnums checks if a field type has enum values
func hasEnums(fieldType *datadictionary.FieldType) bool {
	return len(fieldType.Enums) > 0
}

// getEnumValues returns sorted enum values for a field type
func getEnumValues(fieldType *datadictionary.FieldType) []string {
	var values []string
	for val := range fieldType.Enums {
		values = append(values, val)
	}
	return values
}

// add function for template arithmetic
func add(a, b int) int {
	return a + b
}

// getRequiredFields returns required fields from a MessageDef
func getRequiredFields(msgDef *datadictionary.MessageDef) []*datadictionary.FieldDef {
	var requiredFields []*datadictionary.FieldDef
	for tag := range msgDef.RequiredTags {
		if field, ok := msgDef.Fields[tag]; ok {
			requiredFields = append(requiredFields, field)
		}
	}
	return requiredFields
}

// getOptionalFields returns optional fields from a MessageDef
func getOptionalFields(msgDef *datadictionary.MessageDef) []*datadictionary.FieldDef {
	var optionalFields []*datadictionary.FieldDef
	for tag, field := range msgDef.Fields {
		if _, isRequired := msgDef.RequiredTags[tag]; !isRequired {
			optionalFields = append(optionalFields, field)
		}
	}
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

// sanitizeEnumKey ensures enum keys are valid protobuf identifiers
func sanitizeEnumKey(key string) string {
	// Convert to uppercase first
	result := strings.ToUpper(key)

	// Replace invalid characters with letters
	replacements := map[rune]string{
		'.':  "DOT",
		'-':  "DASH",
		'/':  "SLASH",
		'\\': "BACKSLASH",
		'@':  "AT",
		'#':  "HASH",
		'$':  "DOLLAR",
		'%':  "PERCENT",
		'^':  "CARET",
		'&':  "AND",
		'*':  "STAR",
		'(':  "LPAREN",
		')':  "RPAREN",
		'+':  "PLUS",
		'=':  "EQUAL",
		'[':  "LBRACKET",
		']':  "RBRACKET",
		'{':  "LBRACE",
		'}':  "RBRACE",
		'|':  "PIPE",
		';':  "SEMICOLON",
		':':  "COLON",
		'"':  "QUOTE",
		'\'': "SQUOTE",
		'<':  "LT",
		'>':  "GT",
		',':  "COMMA",
		'?':  "QUESTION",
		'!':  "EXCLAMATION",
		'~':  "TILDE",
		'`':  "BACKTICK",
		' ':  "SPACE",
	}

	var builder strings.Builder
	for _, r := range result {
		if replacement, exists := replacements[r]; exists {
			builder.WriteString(replacement)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			builder.WriteRune(r)
		} else {
			// For any other invalid character, use a generic replacement
			builder.WriteString("X")
		}
	}

	result = builder.String()

	// Ensure it starts with letter or underscore
	if len(result) > 0 && !unicode.IsLetter(rune(result[0])) && result[0] != '_' {
		result = "E_" + result
	}

	return result
}

// generateUniqueEnumKey generates a unique enum key with conflict resolution
func generateUniqueEnumKey(baseKey string) string {
	enumKeyMutex.Lock()
	defer enumKeyMutex.Unlock()

	lowerKey := strings.ToLower(baseKey)
	if _, exists := enumKeyTracker[lowerKey]; !exists {
		enumKeyTracker[lowerKey] = baseKey
		return baseKey
	}

	// Find unique suffix
	counter := 2
	for {
		newKey := fmt.Sprintf("%s_%d", baseKey, counter)
		newLowerKey := strings.ToLower(newKey)
		if _, exists := enumKeyTracker[newLowerKey]; !exists {
			enumKeyTracker[newLowerKey] = newKey
			return newKey
		}
		counter++
	}
}

// printf function for template formatting
func printf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// recordEnumMapping records the mapping between original enum value and generated enum key
func recordEnumMapping(fieldName, originalValue, generatedKey string) string {
	enumMappingMutex.Lock()
	defer enumMappingMutex.Unlock()

	if enumValueMapping[fieldName] == nil {
		enumValueMapping[fieldName] = make(map[string]string)
	}
	enumValueMapping[fieldName][originalValue] = generatedKey

	// Return empty string since this is used for side effects only
	return ""
}

// getEnumKey returns the generated enum key for a field and original value
func getEnumKey(fieldName, originalValue string) string {
	enumMappingMutex.RLock()
	defer enumMappingMutex.RUnlock()

	if fieldMap, exists := enumValueMapping[fieldName]; exists {
		if key, exists := fieldMap[originalValue]; exists {
			return key
		}
	}

	// Fallback: generate the key using the same logic
	enumName := toProtoEnumName(fieldName)
	sanitizedValue := sanitizeEnumKey(originalValue)
	baseKey := fmt.Sprintf("%s_%s", enumName, sanitizedValue)
	return generateUniqueEnumKey(baseKey)
}

// generateEnumValueKey generates the complete enum value key with proper protobuf naming
// while preserving the original semantic meaning using the enum description
func generateEnumValueKey(enumName, enumValue string) string {
	// 首先尝试获取全局字段类型来查找枚举描述
	// 但是在模板上下文中我们可能需要从当前字段类型获取枚举信息
	sanitizedValue := sanitizeEnumKey(enumValue)

	// 保留原始语义：ENUMNAME_ORIGINALVALUE 格式
	baseKey := fmt.Sprintf("%s_%s", enumName, sanitizedValue)

	// 检查是否有冲突，如果有冲突则添加最小的数字后缀
	enumKeyMutex.Lock()
	defer enumKeyMutex.Unlock()

	lowerKey := strings.ToLower(baseKey)
	if _, exists := enumKeyTracker[lowerKey]; !exists {
		enumKeyTracker[lowerKey] = baseKey
		return baseKey
	}

	// 有冲突时，添加最小的数字后缀来区分
	counter := 2
	for {
		newKey := fmt.Sprintf("%s_%d", baseKey, counter)
		newLowerKey := strings.ToLower(newKey)
		if _, exists := enumKeyTracker[newLowerKey]; !exists {
			enumKeyTracker[newLowerKey] = newKey
			return newKey
		}
		counter++
	}
}

// generateEnumValueKeyWithDescription generates enum key using description if available
func generateEnumValueKeyWithDescription(enumName, enumValue, enumDescription string) string {
	var sanitizedValue string

	// 如果有描述信息，优先使用描述
	if enumDescription != "" {
		sanitizedValue = sanitizeEnumKey(enumDescription)
	} else {
		// 否则使用原始值
		sanitizedValue = sanitizeEnumKey(enumValue)
	}

	baseKey := fmt.Sprintf("%s_%s", enumName, sanitizedValue)

	enumKeyMutex.Lock()
	defer enumKeyMutex.Unlock()

	lowerKey := strings.ToLower(baseKey)
	if _, exists := enumKeyTracker[lowerKey]; !exists {
		enumKeyTracker[lowerKey] = baseKey
		return baseKey
	}

	// 有冲突时，添加最小的数字后缀来区分
	counter := 2
	for {
		newKey := fmt.Sprintf("%s_%d", baseKey, counter)
		newLowerKey := strings.ToLower(newKey)
		if _, exists := enumKeyTracker[newLowerKey]; !exists {
			enumKeyTracker[newLowerKey] = newKey
			return newKey
		}
		counter++
	}
}

// clearEnumKeyTracker clears the enum key tracker for fresh generation
func clearEnumKeyTracker() string {
	enumKeyMutex.Lock()
	defer enumKeyMutex.Unlock()
	enumKeyTracker = make(map[string]string)
	return ""
}

// generateEnumValueKeyWithDescriptionDouble generates enum key using description with double prefix
func generateEnumValueKeyWithDescriptionDouble(enumName, enumValue, enumDescription string) string {
	var sanitizedValue string

	// 如果有描述信息，优先使用描述
	if enumDescription != "" {
		sanitizedValue = sanitizeEnumKey(enumDescription)
	} else {
		// 否则使用原始值
		sanitizedValue = sanitizeEnumKey(enumValue)
	}

	// 生成双重前缀格式：ENUMNAME_ENUMNAME_DESCRIPTION
	baseKey := fmt.Sprintf("%s_%s_%s", enumName, enumName, sanitizedValue)

	enumKeyMutex.Lock()
	defer enumKeyMutex.Unlock()

	lowerKey := strings.ToLower(baseKey)
	// 检查是否已经存在，如果存在就直接返回（不再认为是冲突）
	if existingKey, exists := enumKeyTracker[lowerKey]; exists {
		return existingKey
	}

	// 如果不存在，记录并返回
	enumKeyTracker[lowerKey] = baseKey
	return baseKey
}

var templateFuncs = template.FuncMap{
	"toProtoType":                         toProtoType,
	"toProtoEnumName":                     toProtoEnumName,
	"toProtoEnumValue":                    toProtoEnumValue,
	"sanitizeProtoFieldName":              sanitizeProtoFieldName,
	"hasEnums":                            hasEnums,
	"getEnumValues":                       getEnumValues,
	"add":                                 add,
	"getRequiredFields":                   getRequiredFields,
	"getOptionalFields":                   getOptionalFields,
	"getFieldType":                        getFieldType,
	"sanitizeEnumValue":                   sanitizeEnumValue,
	"extractPackageName":                  extractPackageName,
	"sanitizeEnumKey":                     sanitizeEnumKey,
	"generateUniqueEnumKey":               generateUniqueEnumKey,
	"generateEnumValueKey":                generateEnumValueKey,
	"generateEnumValueKeyWithDescription": generateEnumValueKeyWithDescription,
	"generateEnumValueKeyWithDescriptionDouble": generateEnumValueKeyWithDescriptionDouble,
	"clearEnumKeyTracker":                       clearEnumKeyTracker,
	"printf":                                    printf,
	"recordEnumMapping":                         recordEnumMapping,
	"getEnumKey":                                getEnumKey,
}
