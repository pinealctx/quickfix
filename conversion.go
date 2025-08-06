package quickfix

import (
	"time"

	"github.com/quagmt/udecimal"
	"github.com/shopspring/decimal"
)

// GetBoolFieldValue retrieves a boolean field from a FIX message.
func GetBoolFieldValue(msg FieldMap, tag Tag) (bool, error) {
	if !msg.Has(tag) {
		return false, nil
	}
	var f FIXBoolean
	if err := msg.GetField(tag, &f); err != nil {
		return false, err
	}
	return f.Bool(), nil
}

// GetBytesFieldValue retrieves a byte slice field from a FIX message.
func GetBytesFieldValue(msg FieldMap, tag Tag) ([]byte, error) {
	if !msg.Has(tag) {
		return nil, nil
	}
	var f FIXBytes
	if err := msg.GetField(tag, &f); err != nil {
		return nil, err
	}
	return f, nil
}

// GetDecimalFieldValue retrieves a decimal field from a FIX message.
func GetDecimalFieldValue(msg FieldMap, tag Tag) (decimal.Decimal, error) {
	if !msg.Has(tag) {
		return decimal.Decimal{}, nil
	}
	var f FIXDecimal
	if err := msg.GetField(tag, &f); err != nil {
		return decimal.Decimal{}, err
	}
	return f.Decimal, nil
}

// GetFloatFieldValue retrieves a float field from a FIX message.
func GetFloatFieldValue(msg FieldMap, tag Tag) (float64, error) {
	if !msg.Has(tag) {
		return 0, nil
	}
	var f FIXFloat
	if err := msg.GetField(tag, &f); err != nil {
		return 0, err
	}
	return f.Float64(), nil
}

// GetIntFieldValue retrieves an integer field from a FIX message.
func GetIntFieldValue(msg FieldMap, tag Tag) (int, error) {
	if !msg.Has(tag) {
		return 0, nil
	}
	var f FIXInt
	if err := msg.GetField(tag, &f); err != nil {
		return 0, err
	}
	return f.Int(), nil
}

// GetStringFieldValue retrieves a string field from a FIX message.
func GetStringFieldValue(msg FieldMap, tag Tag) (string, error) {
	if !msg.Has(tag) {
		return "", nil
	}
	var f FIXString
	if err := msg.GetField(tag, &f); err != nil {
		return "", err
	}
	return f.String(), nil
}

// GetUDecimalFieldValue retrieves a UDecimal field from a FIX message.
func GetUDecimalFieldValue(msg FieldMap, tag Tag) (udecimal.Decimal, error) {
	if !msg.Has(tag) {
		return udecimal.Decimal{}, nil
	}
	var f FIXUDecimal
	if err := msg.GetField(tag, &f); err != nil {
		return udecimal.Decimal{}, err
	}
	return f.Decimal, nil
}

// GetUTCTimestampFieldValue retrieves a UTC timestamp field from a FIX message.
func GetUTCTimestampFieldValue(msg FieldMap, tag Tag) (time.Time, error) {
	if !msg.Has(tag) {
		return time.Time{}, nil
	}
	var f FIXUTCTimestamp
	if err := msg.GetField(tag, &f); err != nil {
		return time.Time{}, err
	}
	return f.Time, nil
}
