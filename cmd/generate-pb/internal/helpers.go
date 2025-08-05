package internal

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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		h.ReturnCode = 1
	}
}

// WriteFile writes content to a file, creating directories as needed
func WriteFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
