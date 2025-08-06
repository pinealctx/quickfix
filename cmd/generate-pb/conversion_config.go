package main

import (
	"github.com/quickfixgo/quickfix/datadictionary"
)

// ConversionConfig holds configuration for conversion generation
type ConversionConfig struct {
	GoPackagePrefix string
	QuickfixRoot    string
	Messages        []MessageInfo
}

// MessageInfo holds information about a FIX message for conversion
type MessageInfo struct {
	Name    string
	Package string
	*datadictionary.MessageDef
}

// GetImportedPackages returns the list of packages needed for conversion code
func (c *ConversionConfig) GetImportedPackages() []string {
	var imports []string

	// Always need decimal package for price/amount fields
	imports = append(imports, "github.com/shopspring/decimal")

	// Add specific package imports based on message types
	for _, msg := range c.Messages {
		pkgPath := c.QuickfixRoot + "/" + msg.Package
		if !contains(imports, pkgPath) {
			imports = append(imports, pkgPath)
		}
	}

	return imports
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
