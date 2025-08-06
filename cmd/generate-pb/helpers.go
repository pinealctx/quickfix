package main

import (
	"fmt"
	"os"
)

// ErrorHandler handles errors and tracks if any occurred
type ErrorHandler struct {
	ReturnCode int
}

// Handle processes an error and sets the return code
func (h *ErrorHandler) Handle(err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		h.ReturnCode = 1
	}
}

// Err returns whether any errors were handled
func (h *ErrorHandler) Err() error {
	if h.ReturnCode != 0 {
		return fmt.Errorf("errors occurred during generation")
	}
	return nil
}

// WriteFile writes content to a file, creating directories as needed
func WriteFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", filename, closeErr)
		}
	}()

	_, err = file.WriteString(content)
	return err
}
