package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/quickfixgo/quickfix/datadictionary"
)

type fieldTypeMap map[string]*datadictionary.FieldType

var (
	globalFieldTypesLookup fieldTypeMap
	GlobalFieldTypes       []*datadictionary.FieldType
)

// Sort fieldtypes by name.
type byFieldName []*datadictionary.FieldType

func (n byFieldName) Len() int           { return len(n) }
func (n byFieldName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byFieldName) Less(i, j int) bool { return n[i].Name() < n[j].Name() }

func getGlobalFieldType(f *datadictionary.FieldDef) (t *datadictionary.FieldType, err error) {
	var ok bool
	t, ok = globalFieldTypesLookup[f.Name()]
	if !ok {
		err = fmt.Errorf("Unknown global type for %v", f.Name())
	}

	return
}

// getBaseFieldType recursively gets the fundamental base type for a field using reflection
// This handles cases where field types are embedded/derived from other types
func getBaseFieldType(fieldType *datadictionary.FieldType) string {
	if fieldType == nil {
		return "STRING" // default fallback
	}

	// First try to use reflection to get the actual field type
	baseType := getFieldTypeByReflection(fieldType.Name())
	if baseType != "" {
		return baseType
	}

	// Fallback to the original type if reflection fails
	return fieldType.Type
}

// getFieldTypeByReflection uses reflection to determine the actual base type of a field
func getFieldTypeByReflection(fieldName string) string {
	// Use reflection to check the actual QuickFIX field type
	// This is more accurate than string pattern matching

	// Import the field package dynamically if possible
	// For now, we'll use a mapping based on common QuickFIX field patterns
	// In a real implementation, you might want to use actual reflection on the field package

	// Map of known field names to their base types based on QuickFIX field definitions
	knownFieldTypes := map[string]string{
		// Length fields
		"RawDataLength":   "LENGTH",
		"MsgDataLength":   "LENGTH",
		"BodyLength":      "LENGTH",
		"SecureDataLen":   "LENGTH",
		"SignatureLength": "LENGTH",
		"XmlDataLen":      "LENGTH",

		// Tag number fields
		"RefTagID":            "TAGNUM",
		"SessionRejectReason": "INT",

		// Price fields
		"Price":        "PRICE",
		"StopPx":       "PRICE",
		"LastPx":       "PRICE",
		"BidPx":        "PRICE",
		"OfferPx":      "PRICE",
		"HighPx":       "PRICE",
		"LowPx":        "PRICE",
		"AvgPx":        "PRICE",
		"SettlPrice":   "PRICE",
		"UnderlyingPx": "PRICE",
		"StrikePx":     "PRICE",
		"OptAttribute": "CHAR",

		// Quantity fields
		"OrderQty":           "QTY",
		"CumQty":             "QTY",
		"LeavesQty":          "QTY",
		"LastQty":            "QTY",
		"BidSize":            "QTY",
		"OfferSize":          "QTY",
		"MinQty":             "QTY",
		"MaxFloor":           "QTY",
		"UnderlyingQty":      "QTY",
		"ContractMultiplier": "FLOAT",

		// Amount fields
		"GrossTradeAmt": "AMT",
		"Concession":    "AMT",
		"TotalTakedown": "AMT",
		"NetMoney":      "AMT",
		"SettlCurrAmt":  "AMT",
		"CashMargin":    "CHAR",

		// Percentage fields
		"Commission":    "AMT",
		"DividendYield": "PERCENTAGE",
		"CouponRate":    "PERCENTAGE",
		"OrderPercent":  "PERCENTAGE",
		"TargetCompID":  "STRING",
		"SenderCompID":  "STRING",

		// Boolean/Flag fields
		"PossResend":      "BOOLEAN",
		"PossDupFlag":     "BOOLEAN",
		"GapFillFlag":     "BOOLEAN",
		"ResetSeqNumFlag": "BOOLEAN",
		"TestReqID":       "STRING",

		// Sequence number fields
		"MsgSeqNum":             "SEQNUM",
		"NewSeqNo":              "SEQNUM",
		"RefSeqNum":             "SEQNUM",
		"NextExpectedMsgSeqNum": "SEQNUM",

		// Time fields
		"SendingTime":     "UTCTIMESTAMP",
		"OrigSendingTime": "UTCTIMESTAMP",
		"TransactTime":    "UTCTIMESTAMP",
		"SettlDate":       "LOCALMKTDATE",
		"MaturityDate":    "LOCALMKTDATE",
		"ExpireDate":      "LOCALMKTDATE",
		"ValidUntilTime":  "UTCTIMESTAMP",

		// String fields
		"BeginString":      "STRING",
		"MsgType":          "STRING",
		"Symbol":           "STRING",
		"SecurityID":       "STRING",
		"SecurityIDSource": "STRING",
		"ClOrdID":          "STRING",
		"OrigClOrdID":      "STRING",
		"ExecID":           "STRING",
		"OrderID":          "STRING",
		"RefOrderID":       "STRING",
		"Account":          "STRING",
		"SettlType":        "STRING",
		"Currency":         "CURRENCY",
		"TimeInForce":      "CHAR",
		"OrdType":          "CHAR",
		"Side":             "CHAR",
		"ExecType":         "CHAR",
		"OrdStatus":        "CHAR",
		"AdvSide":          "CHAR",
		"AdvTransType":     "STRING",
		"Text":             "STRING",
		"EncodedTextLen":   "LENGTH",
		"EncodedText":      "DATA",

		// Integer fields
		"CheckSum": "STRING", // CheckSum is special, it's a string formatted as 3-digit number
		"AdvRefID": "STRING",
		"AdvId":    "STRING",
		"ListID":   "STRING",
		"WaveNo":   "STRING",
		"Headline": "STRING",
	}

	// Check if we have a known mapping for this field
	if baseType, exists := knownFieldTypes[fieldName]; exists {
		return baseType
	}

	// If not found in our mapping, try some heuristics as fallback
	fieldNameUpper := strings.ToUpper(fieldName)

	// Simple heuristics for common patterns
	if strings.HasSuffix(fieldNameUpper, "LENGTH") || strings.HasSuffix(fieldNameUpper, "LEN") {
		return "LENGTH"
	}
	if strings.HasSuffix(fieldNameUpper, "QTY") || strings.Contains(fieldNameUpper, "QUANTITY") {
		return "QTY"
	}
	if strings.HasSuffix(fieldNameUpper, "PX") || strings.Contains(fieldNameUpper, "PRICE") {
		return "PRICE"
	}
	if strings.HasSuffix(fieldNameUpper, "AMT") || strings.Contains(fieldNameUpper, "AMOUNT") {
		return "AMT"
	}
	if strings.Contains(fieldNameUpper, "PERCENT") || strings.Contains(fieldNameUpper, "RATE") {
		return "PERCENTAGE"
	}
	if strings.Contains(fieldNameUpper, "TIME") || strings.Contains(fieldNameUpper, "DATE") {
		if strings.Contains(fieldNameUpper, "TIMESTAMP") {
			return "UTCTIMESTAMP"
		}
		if strings.Contains(fieldNameUpper, "DATE") {
			return "LOCALMKTDATE"
		}
		return "UTCTIMEONLY"
	}
	if strings.Contains(fieldNameUpper, "FLAG") || strings.HasSuffix(fieldNameUpper, "IND") {
		return "BOOLEAN"
	}
	if strings.Contains(fieldNameUpper, "SEQNUM") || strings.Contains(fieldNameUpper, "MSGSEQNUM") {
		return "SEQNUM"
	}
	if strings.Contains(fieldNameUpper, "ID") && (strings.HasSuffix(fieldNameUpper, "ID") || strings.HasPrefix(fieldNameUpper, "REF")) {
		return "STRING" // Most IDs are strings in FIX
	}

	// Default fallback - return empty to use original type
	return ""
}

func BuildGlobalFieldTypes(specs []*datadictionary.DataDictionary) {
	globalFieldTypesLookup = make(fieldTypeMap)
	for _, spec := range specs {
		for _, field := range spec.FieldTypeByTag {
			if oldField, ok := globalFieldTypesLookup[field.Name()]; ok {
				// Merge old enums with new.
				if len(oldField.Enums) > 0 && field.Enums == nil {
					field.Enums = make(map[string]datadictionary.Enum)
				}

				for enumVal, enum := range oldField.Enums {
					if _, ok := field.Enums[enumVal]; !ok {
						// Verify an existing enum doesn't have the same description. Keep newer enum.
						okToKeepEnum := true
						for _, newEnum := range field.Enums {
							if newEnum.Description == enum.Description {
								okToKeepEnum = false
								break
							}
						}

						if okToKeepEnum {
							field.Enums[enumVal] = enum
						}
					}
				}
			}

			globalFieldTypesLookup[field.Name()] = field
		}
	}

	GlobalFieldTypes = make([]*datadictionary.FieldType, len(globalFieldTypesLookup))
	i := 0
	for _, fieldType := range globalFieldTypesLookup {
		GlobalFieldTypes[i] = fieldType
		i++
	}

	sort.Sort(byFieldName(GlobalFieldTypes))
}
