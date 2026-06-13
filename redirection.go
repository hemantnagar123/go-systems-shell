package main

import (
	"fmt"
	"os"
)

// extracts redirection operators and opens files for writing output or errors
func parseRedirection(fields []string) ([]string, *os.File, *os.File, error) {
	var stderrFile *os.File
	var stdoutFile *os.File
	var cleaned []string

	// loops through command arguments to find redirection symbols
	for i := 0; i < len(fields); i++ {
		field := fields[i]

		// evaluates the current token against standard shell redirection operators
		switch {
		case isStdoutRedirect(field):
			if i+1 >= len(fields) {
				return nil, nil, nil, fmt.Errorf("missing output file")
			}
			
			// closes any previously opened output file to prevent resource leaks
			if stdoutFile != nil {
				stdoutFile.Close()
			}
			// creates a fresh file or overwrites an existing one to catch output
			file, err := os.Create(fields[i+1])
			if err != nil {
				return nil, nil, nil, err
			}
			stdoutFile = file
			i++

		case isStdoutAppendRedirect(field):
			if i+1 >= len(fields) {
				return nil, nil, nil, fmt.Errorf("missing output file")
			}

			// opens a file in append mode or creates it if it is missing
			file, err := os.OpenFile(fields[i+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, nil, nil, err
			}
			stdoutFile = file
			i++

		case isStderrRedirect(field):
			if i+1 >= len(fields) {
				return nil, nil, nil, fmt.Errorf("missing error file")
			}

			// creates a fresh file or overwrites an existing one to catch error messages
			file, err := os.Create(fields[i+1])
			if err != nil {
				return nil, nil, nil, err
			}
			stderrFile = file
			i++

		case isStderrAppendRedirect(field):
			// opens an error log file in append mode or creates it if it is missing
			file, err := os.OpenFile(fields[i+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, nil, nil, err
			}
			stderrFile = file
			i++

		default:
			// keeps the argument as a regular command token if no redirection matched
			cleaned = append(cleaned, field)
		}
	}

	return cleaned, stdoutFile, stderrFile, nil
}

// checks if an argument matches the standard output overwrite symbol
func isStdoutRedirect(field string) bool {
	return field == ">" || field == "1>"
}

// checks if an argument matches the standard output append symbol
func isStdoutAppendRedirect(field string) bool {
	return field == ">>" || field == "1>>"
}

// checks if an argument matches the standard error overwrite symbol
func isStderrRedirect(field string) bool {
	return field == "2>"
}

// checks if an argument matches the standard error append symbol
func isStderrAppendRedirect(field string) bool {
	return field == "2>>"
}