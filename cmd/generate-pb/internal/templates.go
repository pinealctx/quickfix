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

// AllMessagesProtoTemplate generates all message definitions in a single file
var AllMessagesProtoTemplate = template.Must(template.New("messages.proto").Funcs(templateFuncs).Parse(`
syntax = "proto3";

package {{extractPackageName .GoPackagePrefix}};

option go_package = "{{.GoPackagePrefix}}";

import "enums.proto";

{{range .Messages}}
// {{.Name}} message definition (from {{.Package}} specification)
message {{.Name}} {
{{$fieldNum := 1}}{{range $field := getRequiredFields .MessageDef}}{{if $field.IsGroup}}  repeated {{generateGroupMessageName $field}} {{sanitizeProtoFieldName $field.FieldType.Name}} = {{$fieldNum}}; // Required group
{{$fieldNum = add $fieldNum 1}}{{else}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Required field
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}{{end}}{{range $component := getRequiredComponents .MessageDef}}  {{$component.Name}} {{sanitizeProtoFieldName $component.Name}} = {{$fieldNum}}; // Required component
{{$fieldNum = add $fieldNum 1}}{{end}}{{range $field := getOptionalFields .MessageDef}}{{if $field.IsGroup}}  repeated {{generateGroupMessageName $field}} {{sanitizeProtoFieldName $field.FieldType.Name}} = {{$fieldNum}}; // Optional group
{{$fieldNum = add $fieldNum 1}}{{else}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Optional field
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}{{end}}{{range $component := getOptionalComponents .MessageDef}}  {{$component.Name}} {{sanitizeProtoFieldName $component.Name}} = {{$fieldNum}}; // Optional component
{{$fieldNum = add $fieldNum 1}}{{end}}}

{{end}}

{{/* Generate unique group message definitions */}}
{{$seenGroups := dict}}{{range .Messages}}{{range $group := getAllGroups .MessageDef}}{{$groupName := generateGroupMessageName $group}}{{if not (hasKey $seenGroups $groupName)}}{{set $seenGroups $groupName true}}
// {{$groupName}} represents a single entry in the {{$group.FieldType.Name}} repeating group
message {{$groupName}} {
{{$fieldNum := 1}}{{range $field := $group.RequiredFields}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Required group field
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}{{range $field := $group.Fields}}{{$isRequired := false}}{{range $req := $group.RequiredFields}}{{if eq $req.FieldType.Tag $field.FieldType.Tag}}{{$isRequired = true}}{{end}}{{end}}{{if not $isRequired}}{{$fieldType := getFieldType $field}}{{if $fieldType}}  {{if hasEnums $fieldType}}{{toProtoEnumName $fieldType.Name}}{{else}}{{toProtoType $fieldType.Type}}{{end}} {{sanitizeProtoFieldName $fieldType.Name}} = {{$fieldNum}}; // Optional group field
{{$fieldNum = add $fieldNum 1}}{{end}}{{end}}{{end}}}

{{end}}{{end}}{{end}}
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
