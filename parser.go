package main

import (
	"bufio"
	"strings"
)

// splits a command line text string into separate individual arguments
func splitArgs(command string) []string {
	var args []string
	var current strings.Builder

	hasArguments := false
	inQuotes := false
	inDoubleQuotes := false

	for i := 0; i < len(command); i++ {
		c := command[i]

		// checks the character type to process formatting rules
		switch c {

		case '\'':
			// toggles the single quote mode if not inside double quotes
			if !inDoubleQuotes {
				inQuotes = !inQuotes
				hasArguments = true
			} else {
				current.WriteByte(c)
				hasArguments = true
			}

		case '"':
			// toggles the double quote mode if not inside single quotes
			if !inQuotes {
				inDoubleQuotes = !inDoubleQuotes
				hasArguments = true
			} else {
				current.WriteByte(c)
				hasArguments = true
			}

		case ' ', '\t':
			// saves the completed word when encountering a space separator
			if inQuotes || inDoubleQuotes {
				current.WriteByte(c)
				hasArguments = true
			} else if hasArguments {
				args = append(args, current.String())
				current.Reset()
				hasArguments = false
			}

		default:
			// handles escape backslashes and standard characters
			if c == '\\' {

				// evaluates backslash escaping rules based on current quote mode
				if inQuotes {
					// saves the backslash literally when single quotes are active
					current.WriteByte(c)
					hasArguments = true

				} else if inDoubleQuotes {

					// processes escape characters allowed inside double quotes
					if i+1 < len(command) {
						next := command[i+1]

						if next == '"' || next == '\\' {
							current.WriteByte(next)
							i++
						} else {
							current.WriteByte(c)
						}
					} else {
						current.WriteByte(c)
					}

					hasArguments = true

				} else {

					// escapes the next single character in normal text mode
					if i+1 < len(command) {
						i++
						current.WriteByte(command[i])
					} else {
						current.WriteByte(c)
					}

					hasArguments = true
				}

			} else {
				current.WriteByte(c)
				hasArguments = true
			}
		}
	}

	// saves the final word to the list if text still remains
	if hasArguments {
		args = append(args, current.String())
	}

	return args
}

// reads an input text line from a reader stream and trims whitespace
func readCommand(reader *bufio.Reader) (string, error) {
	command, err := reader.ReadString('\n')
	return strings.TrimSpace(command), err
}