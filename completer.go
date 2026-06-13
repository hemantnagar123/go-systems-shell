package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// holds the mapping of command names to their custom autocomplete scripts
var completions = make(map[string]string)

// runs an external completion script to fetch custom autocomplete suggestions
func runCompleter(
	script string,
	command string,
	current string,
	previous string,
	line string,
) ([]string, bool) {

	// prepares the external script command with standard autocompletion arguments
	cmd := exec.Command(
		script,
		command,
		current,
		previous,
	)

	// adds special environment variables that completion scripts expect to receive
	cmd.Env = append(
		os.Environ(),
		"COMP_LINE="+line,
		fmt.Sprintf("COMP_POINT=%d", len(line)),
	)

	// runs the script and catches its printed output text
	output, err := cmd.Output()
	// exits early if the script fails to run properly
	if err != nil {
		return nil, false
	}

	// splits the script output text into individual lines
	lines := strings.Split(
		strings.TrimSpace(string(output)),
		"\n",
	)

	var matches []string

	// loops through each line to clean up spaces and collect valid matches
	for _, line := range lines {

		line = strings.TrimSpace(line)

		// only saves the line if it is not blank
		if line != "" {
			matches = append(matches, line)
		}
	}

	// sorts the autocomplete matches in alphabetical order
	sort.Strings(matches)

	// returns false if no valid suggestions were found
	if len(matches) == 0 {
		return nil, false
	}

	return matches, true
}