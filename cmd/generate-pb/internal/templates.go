package internal

import "text/template"

// EnumProtoTemplate generates protobuf enum definitions
var EnumProtoTemplate = template.Must(template.New("enums.proto").Funcs(templateFuncs).Parse(`
syntax = "proto3";

package {{extractPackageName .GoPackagePrefix}};

option go_package = "{{.GoPackagePrefix}}";

{{range .FieldTypes}}{{if hasEnums .}}
// {{.Name}} enum values
enum {{toProtoEnumName .Name}} {
  {{toProtoEnumName .Name}}_UNSPECIFIED = 0;
{{$fieldName := .Name}}{{$enumName := toProtoEnumName .Name}}{{$fieldType := .}}{{range $index, $enumVal := getEnumValues .}}{{$enum := index $fieldType.Enums $enumVal}}{{$generatedKey := generateEnumValueKeyWithDescription $enumName $enumVal $enum.Description}}{{recordEnumMapping $fieldName $enumVal $generatedKey}}  {{$generatedKey}} = {{add $index 1}};
{{end}}}

{{end}}{{end}}
`))

// MessageProtoTemplate generates protobuf message definitions
var MessageProtoTemplate = template.Must(template.New("message.proto").Funcs(templateFuncs).Parse(`
syntax = "proto3";

package {{extractPackageName .GoPackagePrefix}};

option go_package = "{{.GoPackagePrefix}}";

import "enums.proto";

// {{.Name}} message definition
message {{.Name}} {
{{$fieldNum := 1}}{{range $field := getRequiredFields .MessageDef}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Required
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}{{range $field := getOptionalFields .MessageDef}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Optional
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}}
`))

// AllMessagesProtoTemplate generates all message definitions in a single file
var AllMessagesProtoTemplate = template.Must(template.New("messages.proto").Funcs(templateFuncs).Parse(`
syntax = "proto3";

package {{extractPackageName .GoPackagePrefix}};

option go_package = "{{.GoPackagePrefix}}";

import "enums.proto";

{{range .Messages}}
// {{.Name}} message definition (from {{.Package}} specification)
message {{.Name}} {
{{$fieldNum := 1}}{{range $field := getRequiredFields .MessageDef}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Required
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}{{range $field := getOptionalFields .MessageDef}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Optional
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}}

{{end}}
`))

// EnumExtensionTemplate generates Go code for enum string conversion
var EnumExtensionTemplate = template.Must(template.New("enum_extensions.go").Funcs(templateFuncs).Parse(`
package {{extractPackageName .GoPackagePrefix}}

// Generated enum string conversion functions

{{range .FieldTypes}}{{if hasEnums .}}
// {{.Name}}ToString converts enum value to string
func {{.Name}}ToString(val {{toProtoEnumName .Name}}) string {
	switch val {
{{$enumName := toProtoEnumName .Name}}{{range $enumVal, $enum := .Enums}}	case {{generateEnumValueKeyWithDescriptionDouble $enumName $enumVal $enum.Description}}:
		return "{{$enumVal}}"
{{end}}	default:
		return ""
	}
}

// StringTo{{.Name}} converts string to enum value
func StringTo{{.Name}}(val string) {{toProtoEnumName .Name}} {
	switch val {
{{$enumName := toProtoEnumName .Name}}{{range $enumVal, $enum := .Enums}}	case "{{$enumVal}}":
		return {{generateEnumValueKeyWithDescriptionDouble $enumName $enumVal $enum.Description}}
{{end}}	default:
		return {{$enumName}}_{{$enumName}}_UNSPECIFIED
	}
}

{{end}}{{end}}
`))
