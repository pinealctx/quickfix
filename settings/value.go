package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Integer is a QuickFIX integer setting that accepts a JSON number or string.
type Integer int

// UnmarshalJSON accepts values such as 30 and "30".
func (i *Integer) UnmarshalJSON(data []byte) error {
	var value int
	if err := json.Unmarshal(data, &value); err == nil {
		*i = Integer(value)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("settings: expected integer or numeric string: %w", err)
	}
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return fmt.Errorf("settings: invalid integer %q: %w", text, err)
	}
	*i = Integer(value)
	return nil
}

func (i Integer) quickFIXValue() string { return strconv.Itoa(int(i)) }

// Boolean is a QuickFIX Y/N setting that accepts JSON booleans and common
// textual boolean forms.
type Boolean bool

// UnmarshalJSON accepts booleans and strings such as "Y", "N", "yes", and
// "no".
func (b *Boolean) UnmarshalJSON(data []byte) error {
	var value bool
	if err := json.Unmarshal(data, &value); err == nil {
		*b = Boolean(value)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("settings: expected boolean or boolean string: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true", "y", "yes", "1":
		*b = true
	case "false", "n", "no", "0":
		*b = false
	default:
		return fmt.Errorf("settings: invalid boolean %q", text)
	}
	return nil
}

func (b Boolean) quickFIXValue() string {
	if b {
		return "Y"
	}
	return "N"
}

// Value is an additional QuickFIX setting value. It accepts a JSON string,
// number, or boolean and converts it to the textual representation expected by
// SessionSettings.
type Value string

// UnmarshalJSON converts primitive JSON settings to QuickFIX text values.
func (v *Value) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("settings: invalid additional setting: %w", err)
	}

	switch value := raw.(type) {
	case string:
		*v = Value(value)
	case json.Number:
		*v = Value(value.String())
	case bool:
		*v = Value(Boolean(value).quickFIXValue())
	default:
		return fmt.Errorf("settings: additional setting must be a string, number, or boolean")
	}
	return nil
}
