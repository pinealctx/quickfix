// Package settings provides a typed JSON representation of QuickFIX/Go
// settings. It complements quickfix.ParseSettings, which reads the traditional
// INI-style configuration format.
//
// JSON boolean settings accept booleans or common string forms such as "Y",
// "N", "true", and "false". Integer settings accept JSON numbers or numeric
// strings. Unknown JSON fields are rejected so misspelled settings do not get
// silently ignored.
package settings
