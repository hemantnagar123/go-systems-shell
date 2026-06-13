package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// stores the line text from the last tab press to check for double tabs
var lastTabPrefix string
// stores the matching options from the last tab press to print on double tab
var lastTabMatches []string
// tracks if the user is pressing tab for the second time in a row
var waitingSecondTab bool

// reads a line from the keyboard with history and tab completion support
func readCommandWithAutocomplete() (string, error) {
	var buf []rune

	tmp := make([]byte, 1)
	historyIndex = len(history)
	for {
		_, err := os.Stdin.Read(tmp)
		if err != nil {
			return "", err
		}

		c := rune(tmp[0])

		// checks for multi-byte escape codes like arrow keys
		if c == 27 {
			seq := make([]byte, 2)
			if _, err := os.Stdin.Read(seq); err != nil {
				continue
			}

			if seq[0] == '[' {
				// handles up and down arrow keys for history navigation
				switch seq[1] {
				case 'A': 
					if historyIndex > 0 {
						historyIndex--
						buf = []rune(history[historyIndex])
						redrawLine(string(buf))
						waitingSecondTab = false
					}
				case 'B': 
					if historyIndex < len(history)-1 {
						historyIndex++
						buf = []rune(history[historyIndex])
						redrawLine(string(buf))
						waitingSecondTab = false
					} else if historyIndex == len(history)-1 {
						historyIndex++
						buf = []rune("")
						redrawLine("")
						waitingSecondTab = false
					}
				}
			}
			continue
		}

		// handles special control characters, enter, tab, backspace, and normal text
		switch c {
		case 4: 
			// triggers a clean exit if ctrl+d is pressed on an empty line
			if len(buf) == 0 {
				fmt.Print("\r\n")
				return "", io.EOF 
			}

		case '\n', '\r':
			// processes the enter key to submit the current command line
			fmt.Print("\r\n")

			cmdStr := string(buf)
			
			if strings.TrimSpace(cmdStr) != "" {
				history = append(history, cmdStr)
			}

			historyIndex = len(history)

			waitingSecondTab = false
			return cmdStr, nil

		case '\t':
			// processes the tab key to handle autocomplete matching
			line := string(buf)
			prefix, partial := splitCompletionTarget(line)
			fields := strings.Fields(line)
			isCD := strings.HasPrefix(line, "cd ")
			if len(fields) > 0 {

				command := fields[0]

				// checks and runs custom programmable autocomplete scripts for specific commands
				if script, ok := completions[command]; ok {

					current := partial

					previous := ""

					if len(fields) >= 2 {
						if partial != "" {
							previous = fields[len(fields)-2]
						}
					}

					if matches, ok := runCompleter(
						script,
						command,
						current,
						previous,
						line,
					); ok {

						if len(matches) == 1 {

							buf = []rune(
								prefix + matches[0] + " ",
							)

							redrawLine(string(buf))

							waitingSecondTab = false

							continue
						}

						if len(matches) > 1 {

							lcp := longestCommonPrefix(matches)

							if lcp != current {

								buf = []rune(prefix + lcp)

								redrawLine(string(buf))

								waitingSecondTab = false

								continue
							}

							if !waitingSecondTab || lastTabPrefix != line {

								fmt.Print("\a")

								waitingSecondTab = true
								lastTabPrefix = line
								lastTabMatches = matches

								continue

							}

							fmt.Print("\r\n")
							fmt.Println(strings.Join(lastTabMatches, "  "))
							redrawLine(string(buf))

							waitingSecondTab = false

							continue
						}
					}
					if !ok {
						continue
					}
				}
			}
			
			// handles autocomplete for empty arguments right after a space
			if partial == "" && strings.HasSuffix(line, " ") {

				matches := findFileMatches("", isCD)

				switch len(matches) {
				case 0:
					fmt.Print("\a")

				case 1:
					buf = []rune(line + completeFileMatch(matches[0]))
					redrawLine(string(buf))
					waitingSecondTab = false

				default:
					if !waitingSecondTab || lastTabPrefix != line {
						fmt.Print("\a")
						waitingSecondTab = true
						lastTabPrefix = line
						lastTabMatches = matches
						continue
					}

					fmt.Print("\r\n")
					fmt.Println(strings.Join(lastTabMatches, "  "))
					redrawLine(string(buf))
					waitingSecondTab = false
				}

				continue
			}

			if partial == "" {
				fmt.Print("\a")
				continue
			}

			// completes matching shell builtin command names if it is the first word
			if prefix == "" {
				if completed, ok := completeBuiltin(partial); ok {
					buf = []rune(prefix + completed)
					redrawLine(string(buf))
					waitingSecondTab = false
					continue
				}
			}

			// completes file or directory paths for command arguments
			if prefix != "" {
				matches := findFileMatches(partial, isCD)

				if len(matches) > 1 {
					lcp := longestCommonPrefixFiles(matches)
					if lcp != partial {
						buf = []rune(prefix + lcp)
						redrawLine(string(buf))
						waitingSecondTab = false
						continue
					}
				}

				switch len(matches) {
				case 0:
					fmt.Print("\a")

				case 1:
					buf = []rune(prefix + completeFileMatch(matches[0]))
					redrawLine(string(buf))
					waitingSecondTab = false

				default:
					if !waitingSecondTab || lastTabPrefix != line {
						fmt.Print("\a")
						waitingSecondTab = true
						lastTabPrefix = line
						lastTabMatches = matches
						continue
					}

					fmt.Print("\r\n")
					fmt.Println(strings.Join(lastTabMatches, "  "))
					redrawLine(string(buf))
					waitingSecondTab = false
				}
				continue
			}

			// completes executable program names from system paths if no builtins matched
			matches := findExecutableMatches(partial)

			if len(matches) > 1 {
				lcp := longestCommonPrefix(matches)
				if lcp != partial {
					buf = []rune(prefix + lcp)
					redrawLine(string(buf))
					waitingSecondTab = false
					continue
				}
			}

			switch len(matches) {
			case 0:
				fmt.Print("\a")

			case 1:
				buf = []rune(prefix + matches[0] + " ")
				redrawLine(string(buf))
				waitingSecondTab = false

			default:
				if !waitingSecondTab || lastTabPrefix != line {
					fmt.Print("\a")
					waitingSecondTab = true
					lastTabPrefix = line
					lastTabMatches = matches
					continue
				}

				fmt.Print("\r\n")
				fmt.Println(strings.Join(lastTabMatches, "  "))
				redrawLine(string(buf))
				waitingSecondTab = false
			}

		case '\b', 127:
			// processes the backspace key to delete the last character from the input buffer
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]

				waitingSecondTab = false

				redrawLine(string(buf))
			}

		default:
			// prints normal readable letters and saves them to the text buffer
			if c >= 32 && c <= 126 {

				buf = append(buf, c)
				fmt.Printf("%c", c)
			}
		}
	}
}

// splits a line into everything before the last word and the last word itself
func splitCompletionTarget(line string) (string, string) {
	lastSpace := strings.LastIndexFunc(line, unicode.IsSpace)
	if lastSpace == -1 {
		return "", line
	}

	return line[:lastSpace+1], line[lastSpace+1:]
}

// searches for a builtin command name that starts with the typed text
func completeBuiltin(partial string) (string, bool) {
	for _, cmd := range builtins {
		if strings.HasPrefix(cmd, partial) {
			return cmd + " ", true
		}
	}

	return "", false
}

// clears the current line and rewrites the custom folder prompt with the updated text
func redrawLine(line string) {
	fmt.Print("\r")
	fmt.Print(getPrompt()) 
	fmt.Print(line)
	fmt.Print("\033[K")
}