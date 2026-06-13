package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"unicode"
)

// checks if a command is a builtin and runs it
func handleBuiltin(fields []string, stdoutFile *os.File) bool {
	cmd := fields[0]

	// routes the command name to its matching builtin handler function
	switch cmd {
	case "exit":
		// saves the shell command history and exits the program cleanly
		saveHistoryToEnv()
		os.Exit(0)

	case "echo":
		// runs the builtin echo command with given arguments and output file
		handleEcho(fields[1:], stdoutFile)
		return true

	case "type":
		// runs the builtin type command to locate or identify a command
		handleType(fields[1:])
		return true

	case "pwd":
		// runs the builtin pwd command to display the working directory
		handlePwd(stdoutFile)
		return true

	case "cd":
		// runs the builtin cd command to change directories
		handleCd(fields[1:])
		return true

	case "complete":
		// runs the complete command to manage programmable tab completions
		handleComplete(fields[1:])
		return true

	case "jobs":
		// runs the jobs command to display background tasks
		handleJobs()
		return true

	case "history":
		// runs the history command to show or manage old commands
		handleHistory(fields[1:])
		return true

	case "declare":
		// runs the declare command to manage custom shell variables
		handleDeclare(fields[1:])
		return true
	}
	return false
}

// replaces variable macros like $var or ${var} with their real values
func expandArgs(args []string) []string {
	var expanded []string
	for _, arg := range args {
		// scans and updates all variable matching patterns found in the argument
		newArg := varRegex.ReplaceAllStringFunc(arg, func(match string) string {
			varName := ""
			// extracts the variable name based on whether it uses braces or not
			if strings.HasPrefix(match, "${") && strings.HasSuffix(match, "}") {
				varName = match[2 : len(match)-1]
			} else {
				varName = match[1:]
			}

			// returns the value if it exists in local shell variables
			if val, ok := shellVariables[varName]; ok {
				return val
			}

			// falls back to looking up the variable in the system environment
			if envVal, envExists := os.LookupEnv(varName); envExists {
				return envVal
			}

			return ""
		})

		// only adds the argument to the list if it did not expand into an empty string
		if newArg != "" {
			expanded = append(expanded, newArg)
		}
	}
	return expanded
}

// processes variable creation, listing, or updates for the shell
func handleDeclare(args []string) {
	if len(args) == 0 {
		return
	}

	// prints the value of a variable if it is stored in the shell
	if args[0] == "-p" && len(args) >= 2 {
		varName := args[1]

		if value, exists := shellVariables[varName]; exists {
			fmt.Printf("declare -- %s=\"%s\"\n", varName, value)
		} else {
			fmt.Printf("declare: %s: not found\n", varName)
		}
		return
	}

	// stitches arguments with spaces to keep values together before assignment
	fullAssignment := strings.Join(args, " ")

	// splits the assignment string into a name and its value
	if strings.Contains(fullAssignment, "=") {
		parts := strings.SplitN(fullAssignment, "=", 2)
		varName := parts[0]
		varValue := parts[1]

		// stops the assignment if the variable name has illegal characters
		if !isValidIdentifier(varName) {
			fmt.Printf("declare: `%s': not a valid identifier\n", fullAssignment)
			return
		}

		shellVariables[varName] = varValue
		return
	}
}

// verifies if a variable name follows standard shell naming rules
func isValidIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}

	// checks if the first character is a valid letter or an underscore
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}

	// checks if all remaining characters are letters, digits, or underscores
	for _, r := range name[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}

	return true
}

// writes output for basic builtins directly to a given pipeline or file stream
func executeBuiltin(fields []string, in io.Reader, out io.Writer) {
	if len(fields) == 0 {
		return
	}
	// routes basic builtins to print to the requested output destination
	switch fields[0] {
	case "echo":
		fmt.Fprintln(out, strings.Join(fields[1:], " "))

	case "type":
		executeType(fields[1:], out)

	case "pwd":
		pwd, _ := os.Getwd()
		fmt.Fprintln(out, pwd)
	}
}

// checks if a command name is part of the registered builtin list
func isBuiltin(cmd string) bool {
	return slices.Contains(builtins, cmd)
}

// formats and prints arguments separated by spaces to a file or terminal
func handleEcho(args []string, output *os.File) {
	if output != nil {
		fmt.Fprintln(output, strings.Join(args, " "))
		return
	}

	executeBuiltin(
		append([]string{"echo"}, args...),
		nil,
		os.Stdout,
	)
}

// wraps the type execution logic defaulting to standard terminal output
func handleType(args []string) {
	executeType(args, os.Stdout)
}

// evaluates if a command is an internal builtin, an external system path, or missing
func executeType(args []string, out io.Writer) {
	if len(args) == 0 {
		return
	}

	target := args[0]

	if slices.Contains(builtins, target) {
		fmt.Fprintf(out, "%s is a shell builtin\n", target)
		return
	}

	if path, err := exec.LookPath(target); err == nil {
		fmt.Fprintf(out, "%s is %s\n", target, path)
		return
	}

	fmt.Fprintf(out, "%s: not found\n", target)
}

// fetches the current working directory from the system and prints it
func handlePwd(out *os.File) {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pwd: error getting current directory:", err)
		return
	}

	fmt.Fprintln(out, pwd)
}

// updates the current working directory of the shell process
func handleCd(args []string) {
	// defaults to the home directory if no target path is supplied
	if len(args) == 0 {
		args = append(args, "~")
	}
	// finds and changes to the user home directory path if tilde is used
	if args[0] == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", args[0])
			return
		}
		err = os.Chdir(home)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", args[0])
		}
		return
	}

	err := os.Chdir(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", args[0])
	}
}

// registers, removes, or shows custom tab completion scripts for commands
func handleComplete(args []string) {
	// registers a script to handle autocomplete for a specific command
	if len(args) >= 3 && args[0] == "-C" {
		script := args[1]
		command := args[2]
		completions[command] = script
		return
	}
	// removes a registered autocomplete configuration for a command
	if len(args) >= 2 && args[0] == "-r" {
		command := args[1]
		delete(completions, command)
		return
	}
	// lists the registered autocomplete configuration for a given command
	if len(args) >= 2 && args[0] == "-p" {
		command := args[1]

		if script, ok := completions[command]; ok {
			fmt.Printf("complete -C '%s' %s\n", script, command)
		} else {
			fmt.Printf("complete: %s: no completion specification\n", command)
		}
		return
	}
}

// loads, writes, appends, or displays the list of executed commands
func handleHistory(args []string) {
	if len(args) >= 2 {
		// reads a history file and appends non-empty lines to the session history
		if args[0] == "-r" {
			filePath := args[1]
			file, err := os.Open(filePath)
			if err != nil {
				return
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) != "" {
					history = append(history, line)
				}
			}

			sessionHistoryStart = len(history)
			return
		}

		// overwrites a file with the entire current session history
		if args[0] == "-w" {
			filePath := args[1]
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return
			}
			defer file.Close()

			writer := bufio.NewWriter(file)
			for _, cmd := range history {
				_, _ = writer.WriteString(cmd + "\n")
			}
			_ = writer.Flush()
			return
		}

		// appends only newly executed commands from this session to a history file
		if args[0] == "-a" {
			filePath := args[1]

			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return
			}
			defer file.Close()

			writer := bufio.NewWriter(file)

			for i := sessionHistoryStart; i < len(history); i++ {
				_, _ = writer.WriteString(history[i] + "\n")
			}
			_ = writer.Flush()

			sessionHistoryStart = len(history)
			return
		}
	}

	// checks if the user passed a number to limit the history display size
	start := 0
	if len(args) > 0 {
		var n int
		if _, err := fmt.Sscanf(args[0], "%d", &n); err == nil {
			if n < len(history) {
				start = len(history) - n
			}
		}
	}

	// loops through and prints the requested slice of command history entries
	for i := start; i < len(history); i++ {
		fmt.Printf("%5d  %s\n", i+1, history[i])
	}
}