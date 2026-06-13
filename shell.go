package main

import (
	"bufio"
	"fmt"
	"golang.org/x/term"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var history []string
var historyIndex int
var sessionHistoryStart int
var varRegex = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}|\$([a-zA-Z_][a-zA-Z0-9_]*)`)

var shellVariables = make(map[string]string)

// fetches the current working directory and formats it for the command prompt
func getPrompt() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "$ "
	}

	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + strings.TrimPrefix(cwd, home)
	}

	cwd = strings.ReplaceAll(cwd, "\\", "/")

	return cwd + " $ "
}

// starts the main interactive shell loop to read and execute commands
func runShell() {
	initHistoryFromEnv()
	for {
		refreshJobs()
		reapJobs()
		command, err := readCommandFromPrompt()
		if err != nil {
			// checks if the user pressed ctrl+d to signal end of file and exit
			if err == io.EOF {
				fmt.Println("exit")
				saveHistoryToEnv()
				os.Exit(0)
			}
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		handleCommand(command)
	}
}

// writes all command history items into the local tracking file
func saveHistoryToEnv() {
	histFile := getHistoryPath()
	if histFile == "" {
		return
	}

	file, err := os.OpenFile(histFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, cmd := range history {
		_, _ = writer.WriteString(cmd + "\n")
	}
	_ = writer.Flush()
}

// determines the storage file path using environment variables or fallback defaults
func getHistoryPath() string {
	path := os.Getenv("HISTFILE")
	if path == "" {
		return "my_shell_history.txt"
	}
	return path
}

type PipelineStage struct {
	Args []string
}

// reads saved commands from the history file at shell startup
func initHistoryFromEnv() {
	histFile := getHistoryPath()
	file, err := os.Open(histFile)
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
}

// puts the terminal into raw mode and prints the prompt to capture user input
func readCommandFromPrompt() (string, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	
	fmt.Print(getPrompt()) 
	return readCommandWithAutocomplete()
}

// parses and routes a command line to pipelines, builtins, or external programs
func handleCommand(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	background := false
	// checks if the entire command sequence should run in the background
	if strings.HasSuffix(command, "&") {
		background = true
		command = strings.TrimSpace(strings.TrimSuffix(command, "&"))
	}

	// routes the command text to the pipeline handler if pipe characters exist
	if strings.Contains(command, "|") {
		runPipeline(command, background)
		return
	}

	fields := splitArgs(command)
	fields = expandArgs(fields)

	fields, stdoutFile, stderrFile, err := parseRedirection(fields)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer closeRedirectionFiles(stdoutFile, stderrFile)

	if handleBuiltin(fields, stdoutFile) {
		return
	}

	runExternalCommand(fields, stdoutFile, stderrFile, background)
}

// executes multiple commands linked together with standard input and output pipes
func runPipeline(line string, background bool) {
	parts := strings.Split(line, "|")
	var stages []PipelineStage

	for _, part := range parts {
		args := splitArgs(strings.TrimSpace(part))
		args = expandArgs(args)
		stages = append(stages, PipelineStage{Args: args})
	}

	var commands []*exec.Cmd
	pipes := make([][2]*os.File, len(stages)-1)

	for i := 0; i < len(stages)-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return
		}
		pipes[i] = [2]*os.File{r, w}
	}

	var pipelineStdoutFile *os.File
	var pipelineStderrFile *os.File

	for i, stage := range stages {
		if len(stage.Args) == 0 {
			return
		}

		var stdin io.Reader = os.Stdin
		var stdout io.Writer = os.Stdout

		// checks if this stage needs to read input from a previous stage pipe
		if i > 0 {
			stdin = pipes[i-1][0]
		}

		// checks if this stage needs to send its output into a next stage pipe or file redirection
		if i < len(stages)-1 {
			stdout = pipes[i][1]
		} else {
			var stdoutFile, stderrFile *os.File
			var err error
			stage.Args, stdoutFile, stderrFile, err = parseRedirection(stage.Args)
			if err == nil {
				if stdoutFile != nil {
					stdout = stdoutFile
					pipelineStdoutFile = stdoutFile
				}
				if stderrFile != nil {
					pipelineStderrFile = stderrFile
				}
			}
		}

		// checks if a pipeline stage is an internal builtin command and runs it
		if isBuiltin(stage.Args[0]) {
			executeBuiltin(stage.Args, stdin, stdout)

			if i < len(stages)-1 {
				pipes[i][1].Close()
			}
			if f, ok := stdout.(*os.File); ok && f != os.Stdout {
				_ = f.Close()
			}
			continue
		}

		cmd := exec.Command(stage.Args[0], stage.Args[1:]...)
		cmd.Stdin = stdin
		cmd.Stdout = stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			fmt.Printf("%s: command not found\n", stage.Args[0])
			for _, p := range pipes {
				p[0].Close()
				p[1].Close()
			}
			if pipelineStdoutFile != nil { pipelineStdoutFile.Close() }
			if pipelineStderrFile != nil { pipelineStderrFile.Close() }
			return
		}
		commands = append(commands, cmd)
	}

	for _, p := range pipes {
		p[0].Close()
		p[1].Close()
	}

	// checks if the pipeline chain needs to run asynchronously as a background job
	if background && len(commands) > 0 {
		lastCmd := commands[len(commands)-1]
		job := Job{
			ID:      getNextJobID(),
			PID:     lastCmd.Process.Pid,
			Command: line + " &",
		}

		// locks the shared job list to prevent data races while updating background tasks
		jobsMutex.Lock()
		jobsList = append(jobsList, job)
		jobsMutex.Unlock()

		fmt.Printf("[%d] %d\n", job.ID, job.PID)

		// spawns an isolated thread to monitor pipeline processes and close shared file descriptors safely
		go func(jobID int, outFile *os.File, errFile *os.File) {
			for _, cmd := range commands {
				_ = cmd.Wait()
			}

			if outFile != nil { outFile.Close() }
			if errFile != nil { errFile.Close() }

			// locks the shared job list before marking this background pipeline task as finished
			jobsMutex.Lock()
			defer jobsMutex.Unlock()
			for i := range jobsList {
				if jobsList[i].ID == jobID {
					jobsList[i].Done = true
					return
				}
			}
		}(job.ID, pipelineStdoutFile, pipelineStderrFile)

		return
	}

	for _, cmd := range commands {
		cmd.Wait()
	}
	if pipelineStdoutFile != nil { pipelineStdoutFile.Close() }
	if pipelineStderrFile != nil { pipelineStderrFile.Close() }
}

// explicitly closes file descriptors allocated during stream redirection
func closeRedirectionFiles(stdoutFile *os.File, stderrFile *os.File) {
	if stdoutFile != nil {
		stdoutFile.Close()
	}
	if stderrFile != nil {
		stderrFile.Close()
	}
}