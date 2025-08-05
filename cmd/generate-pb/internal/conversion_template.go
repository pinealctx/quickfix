package internal

import "text/template"

// ConversionTemplate generates conversion functions between protobuf and QuickFIX structs
var ConversionTemplate = template.Must(template.New("conversions.go").Funcs(templateFuncs).Parse(`
package {{extractPackageName .GoPackagePrefix}}

import (
	"strconv"
	"strings"
	"time"
	"github.com/shopspring/decimal"
	"github.com/quickfixgo/quickfix"
	"{{.QuickfixRoot}}/field"
	"{{.QuickfixRoot}}/enum"
	"{{.QuickfixRoot}}/tag"
)

// Generated conversion functions between protobuf and QuickFIX structs

{{range .Messages}}
// {{.Name}}ToProto converts QuickFIX {{.Name}} message to protobuf
func {{.Name}}ToProto(msg quickfix.Messagable) *{{.Name}} {
	proto := &{{.Name}}{}
	message := msg.ToMessage()
	
{{range $field := getRequiredFields .MessageDef}}{{if $field.IsGroup}}	// Convert group: {{$field.FieldType.Name}}
	var {{sanitizeProtoFieldName $field.FieldType.Name}}Field field.{{$field.FieldType.Name}}Field
	if err := message.Body.GetField(tag.{{$field.FieldType.Name}}, &{{sanitizeProtoFieldName $field.FieldType.Name}}Field); err == nil {
		numGroups := {{sanitizeProtoFieldName $field.FieldType.Name}}Field.Int()
		for i := 0; i < numGroups; i++ {
			groupEntry := &{{generateGroupMessageName $field}}{}
			// Extract group fields from the message - this would need custom logic per group type
			// For now, create empty group entry
			proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}} = append(proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}}, groupEntry)
		}
	}
{{else}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert field: {{$fieldType.Name}}
	var {{sanitizeProtoFieldName $fieldType.Name}}Field field.{{$fieldType.Name}}Field
	if err := message.Body.GetField(tag.{{$fieldType.Name}}, &{{sanitizeProtoFieldName $fieldType.Name}}Field); err == nil {
{{if hasEnums $fieldType}}		if enumValue := StringTo{{$fieldType.Name}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}}); enumValue != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
			proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = enumValue
		}
{{else}}		proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = convertQuickFixTo{{toProtoTypeCapitalized $fieldType.Type}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}})
{{end}}	}
{{end}}{{end}}{{end}}{{range $field := getOptionalFields .MessageDef}}{{if $field.IsGroup}}	// Convert optional group: {{$field.FieldType.Name}}
	var {{sanitizeProtoFieldName $field.FieldType.Name}}Field field.{{$field.FieldType.Name}}Field
	if err := message.Body.GetField(tag.{{$field.FieldType.Name}}, &{{sanitizeProtoFieldName $field.FieldType.Name}}Field); err == nil {
		numGroups := {{sanitizeProtoFieldName $field.FieldType.Name}}Field.Int()
		for i := 0; i < numGroups; i++ {
			groupEntry := &{{generateGroupMessageName $field}}{}
			// Extract group fields from the message - this would need custom logic per group type
			// For now, create empty group entry
			proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}} = append(proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}}, groupEntry)
		}
	}
{{else}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert optional field: {{$fieldType.Name}}
	var {{sanitizeProtoFieldName $fieldType.Name}}Field field.{{$fieldType.Name}}Field
	if err := message.Body.GetField(tag.{{$fieldType.Name}}, &{{sanitizeProtoFieldName $fieldType.Name}}Field); err == nil {
{{if hasEnums $fieldType}}		if enumValue := StringTo{{$fieldType.Name}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}}); enumValue != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
			proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = enumValue
		}
{{else}}		proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = convertQuickFixTo{{toProtoTypeCapitalized $fieldType.Type}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}})
{{end}}	}
{{end}}{{end}}{{end}}
	return proto
}

// {{.Name}}FromProto converts protobuf {{.Name}} to QuickFIX message
func {{.Name}}FromProto(proto *{{.Name}}) *quickfix.Message {
	msg := quickfix.NewMessage()
	
{{range $field := getRequiredFields .MessageDef}}{{if $field.IsGroup}}	// Convert group from proto: {{$field.FieldType.Name}}
	if len(proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}}) > 0 {
		msg.Body.SetField(tag.{{$field.FieldType.Name}}, field.New{{$field.FieldType.Name}}(len(proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}})))
		// Note: Individual group entries would need to be added to a repeating group structure
		// This requires more complex logic to properly reconstruct the repeating group
	}
{{else}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert field from proto: {{$fieldType.Name}}
{{if hasEnums $fieldType}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(enum.{{$fieldType.Name}}({{$fieldType.Name}}ToString(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}}))))
	}
{{else}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{getZeroValue $fieldType.Type}} {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(convertProtoTo{{toProtoTypeCapitalized $fieldType.Type}}(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}})))
	}
{{end}}{{end}}{{end}}{{end}}{{range $field := getOptionalFields .MessageDef}}{{if $field.IsGroup}}	// Convert optional group from proto: {{$field.FieldType.Name}}
	if len(proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}}) > 0 {
		msg.Body.SetField(tag.{{$field.FieldType.Name}}, field.New{{$field.FieldType.Name}}(len(proto.{{sanitizeProtoFieldNameForGo $field.FieldType.Name}})))
		// Note: Individual group entries would need to be added to a repeating group structure
		// This requires more complex logic to properly reconstruct the repeating group
	}
{{else}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert optional field from proto: {{$fieldType.Name}}
{{if hasEnums $fieldType}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(enum.{{$fieldType.Name}}({{$fieldType.Name}}ToString(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}}))))
	}
{{else}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{getZeroValue $fieldType.Type}} {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(convertProtoTo{{toProtoTypeCapitalized $fieldType.Type}}(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}})))
	}
{{end}}{{end}}{{end}}{{end}}
	return msg
}

{{end}}

{{/* Generate group conversion functions */}}
{{$seenGroups := dict}}{{range .Messages}}{{range $group := getAllGroups .MessageDef}}{{$groupName := generateGroupMessageName $group}}{{if not (hasKey $seenGroups $groupName)}}{{set $seenGroups $groupName true}}
// {{$groupName}}ToProto converts QuickFIX group to protobuf
func {{$groupName}}ToProto(msg quickfix.Messagable) *{{$groupName}} {
	proto := &{{$groupName}}{}
	message := msg.ToMessage()
	
{{range $field := $group.RequiredFields}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert required group field: {{$fieldType.Name}}
	var {{sanitizeProtoFieldName $fieldType.Name}}Field field.{{$fieldType.Name}}Field
	if err := message.Body.GetField(tag.{{$fieldType.Name}}, &{{sanitizeProtoFieldName $fieldType.Name}}Field); err == nil {
{{if hasEnums $fieldType}}		if enumValue := StringTo{{$fieldType.Name}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}}); enumValue != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
			proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = enumValue
		}
{{else}}		proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = convertQuickFixTo{{toProtoTypeCapitalized $fieldType.Type}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}})
{{end}}	}
{{end}}{{end}}{{range $field := $group.Fields}}{{$isRequired := false}}{{range $req := $group.RequiredFields}}{{if eq $req.FieldType.Tag $field.FieldType.Tag}}{{$isRequired = true}}{{end}}{{end}}{{if not $isRequired}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert optional group field: {{$fieldType.Name}}
	var {{sanitizeProtoFieldName $fieldType.Name}}Field field.{{$fieldType.Name}}Field
	if err := message.Body.GetField(tag.{{$fieldType.Name}}, &{{sanitizeProtoFieldName $fieldType.Name}}Field); err == nil {
{{if hasEnums $fieldType}}		if enumValue := StringTo{{$fieldType.Name}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}}); enumValue != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
			proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = enumValue
		}
{{else}}		proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} = convertQuickFixTo{{toProtoTypeCapitalized $fieldType.Type}}({{getFieldValueConversionCall $fieldType.Name $fieldType.Type}})
{{end}}	}
{{end}}{{end}}{{end}}
	return proto
}

// {{$groupName}}FromProto converts protobuf group to QuickFIX
func {{$groupName}}FromProto(proto *{{$groupName}}) *quickfix.Message {
	msg := quickfix.NewMessage()
	
{{range $field := $group.RequiredFields}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert required group field from proto: {{$fieldType.Name}}
{{if hasEnums $fieldType}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(enum.{{$fieldType.Name}}({{$fieldType.Name}}ToString(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}}))))
	}
{{else}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{getZeroValue $fieldType.Type}} {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(convertProtoTo{{toProtoTypeCapitalized $fieldType.Type}}(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}})))
	}
{{end}}{{end}}{{end}}{{range $field := $group.Fields}}{{$isRequired := false}}{{range $req := $group.RequiredFields}}{{if eq $req.FieldType.Tag $field.FieldType.Tag}}{{$isRequired = true}}{{end}}{{end}}{{if not $isRequired}}{{$fieldType := getFieldType $field}}{{if $fieldType}}	// Convert optional group field from proto: {{$fieldType.Name}}
{{if hasEnums $fieldType}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{toProtoEnumName $fieldType.Name}}_{{toProtoEnumName $fieldType.Name}}_UNSPECIFIED {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(enum.{{$fieldType.Name}}({{$fieldType.Name}}ToString(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}}))))
	}
{{else}}	if proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}} != {{getZeroValue $fieldType.Type}} {
		msg.Body.SetField(tag.{{$fieldType.Name}}, field.New{{$fieldType.Name}}(convertProtoTo{{toProtoTypeCapitalized $fieldType.Type}}(proto.{{sanitizeProtoFieldNameForGo $fieldType.Name}})))
	}
{{end}}{{end}}{{end}}{{end}}
	return msg
}

{{end}}{{end}}{{end}}

// Type conversion helper functions
func convertQuickFixToString(value string) string { return value }
func convertQuickFixToInt32(value string) int32 { 
	if i, err := strconv.ParseInt(value, 10, 32); err == nil { return int32(i) }
	return 0
}
func convertQuickFixToUint32(value string) uint32 { 
	if i, err := strconv.ParseUint(value, 10, 32); err == nil { return uint32(i) }
	return 0
}
func convertQuickFixToDouble(value string) float64 { 
	if f, err := strconv.ParseFloat(value, 64); err == nil { return f }
	return 0.0
}
// convertQuickFixToDecimal returns string representation for decimal types
func convertQuickFixToDecimal(value string) string { 
	// For decimal types, return the string representation directly to preserve precision
	return value
}

// convertQuickFixToTime returns string representation for time types  
func convertQuickFixToTime(value string) string {
	// For time types, return the string representation directly
	return value
}

func convertProtoToString(value string) string { return value }
func convertProtoToInt32(value int32) string { return strconv.FormatInt(int64(value), 10) }
func convertProtoToUint32(value uint32) string { return strconv.FormatUint(uint64(value), 10) }
func convertProtoToDouble(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

// convertProtoToDecimalValue extracts the decimal value from string
func convertProtoToDecimalValue(value string) decimal.Decimal { 
	if value == "" {
		return decimal.Zero
	}
	
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero
	}
	
	return d
}

// convertProtoToDecimalScale extracts the scale from string
func convertProtoToDecimalScale(value string) int32 { 
	// Determine appropriate scale based on the value
	// Count decimal places in the original string
	scale := int32(0)
	if dotIndex := strings.Index(value, "."); dotIndex != -1 {
		scale = int32(len(value) - dotIndex - 1)
	}
	
	// Set reasonable limits for scale
	if scale > 8 {
		scale = 8 // Max scale for financial precision
	}
	if scale < 0 {
		scale = 0
	}
	
	// Default scale for empty values
	if value == "" {
		scale = 2
	}
	
	return scale
}

// convertProtoToDecimal creates a FIXDecimal from string (for single parameter functions)
func convertProtoToDecimal(value string) quickfix.FIXDecimal {
	// Parse the decimal string and create FIXDecimal with appropriate scale
	if value == "" {
		return quickfix.FIXDecimal{Decimal: decimal.Zero, Scale: 2} // Default scale of 2
	}
	
	d, err := decimal.NewFromString(value)
	if err != nil {
		return quickfix.FIXDecimal{Decimal: decimal.Zero, Scale: 2}
	}
	
	// Determine appropriate scale based on the value
	scale := int32(0)
	if dotIndex := strings.Index(value, "."); dotIndex != -1 {
		scale = int32(len(value) - dotIndex - 1)
	}
	
	// Set reasonable limits for scale
	if scale > 8 {
		scale = 8 // Max scale for financial precision
	}
	if scale < 0 {
		scale = 0
	}
	
	return quickfix.FIXDecimal{Decimal: d, Scale: scale}
}

// convertProtoToTimeValue extracts the time.Time value from string
func convertProtoToTimeValue(value string) time.Time {
	if value == "" {
		return time.Time{} // Zero time
	}
	
	// Try parsing different time formats
	// RFC3339 format (ISO 8601)
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}
	
	// RFC3339Nano format
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t
	}
	
	// FIX timestamp formats
	formats := []string{
		"20060102-15:04:05.000000000", // Nanos
		"20060102-15:04:05.000000",    // Micros  
		"20060102-15:04:05.000",       // Millis
		"20060102-15:04:05",           // Seconds
		"20060102",                    // Date only
		"15:04:05",                    // Time only
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t
		}
	}
	
	return time.Time{} // Return zero time if parsing fails
}

// convertProtoToTimePrecision extracts the timestamp precision from string
func convertProtoToTimePrecision(value string) quickfix.TimestampPrecision {
	if value == "" {
		return quickfix.Millis // Default precision
	}
	
	// Determine precision based on string length and format
	switch len(value) {
	case 17: // "20060102-15:04:05"
		return quickfix.Seconds
	case 21: // "20060102-15:04:05.000"
		return quickfix.Millis
	case 24: // "20060102-15:04:05.000000"
		return quickfix.Micros
	case 27: // "20060102-15:04:05.000000000"
		return quickfix.Nanos
	default:
		// For RFC3339 formats, check for fractional seconds
		if strings.Contains(value, ".") {
			dotIndex := strings.Index(value, ".")
			if dotIndex != -1 {
				remaining := value[dotIndex+1:]
				// Remove timezone info
				if zIndex := strings.IndexAny(remaining, "Z+-"); zIndex != -1 {
					remaining = remaining[:zIndex]
				}
				switch len(remaining) {
				case 1, 2, 3:
					return quickfix.Millis
				case 4, 5, 6:
					return quickfix.Micros
				case 7, 8, 9:
					return quickfix.Nanos
				}
			}
		}
		return quickfix.Millis // Default
	}
}

func convertProtoToBool(value bool) string { if value { return "Y" } else { return "N" } }
`))
