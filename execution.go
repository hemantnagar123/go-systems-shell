package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// launches external system programs handles file redirection and background tasks
func runExternalCommand(
	fields []string,
	output *os.File,
	stderr *os.File,
	background bool,
) {
	// prepares the system command using the first token as the program name
	cmd := exec.Command(fields[0], fields[1:]...)

	// routes standard output to a redirection file if provided or defaults to terminal output
	if output != nil {
		cmd.Stdout = output
	} else {
		cmd.Stdout = os.Stdout
	}

	// routes standard error to a redirection file if provided or defaults to terminal error
	if stderr != nil {
		cmd.Stderr = stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	// checks if the command should run asynchronously in the background
	if background {
		// attempts to start the program in the background and catches errors if it is missing
		if err := cmd.Start(); err != nil {
			fmt.Printf("%s: command not found\n", fields[0])
			return
		}

		// creates a tracking object with unique job metadata and process ids
		job := Job{
			ID:      getNextJobID(),
			PID:     cmd.Process.Pid,
			Command: strings.Join(fields, " ") + " &",
		}

		// locks the shared job list to prevent data races while saving the new job
		jobsMutex.Lock() 
		jobsList = append(jobsList, job)
		jobsMutex.Unlock()

		// displays the newly assigned job id and process id numbers to the screen
		fmt.Printf("[%d] %d\n", job.ID, job.PID)

		// spawns an isolated background thread to track when the process concludes
		go func(jobID int) {
			// holds the thread here until the underlying background command completes
			_ = cmd.Wait()
			
			// locks the shared job list before marking this specific job as finished
			jobsMutex.Lock() 
			defer jobsMutex.Unlock()
			// loops through the active jobs to update the done flag on the matching id
			for i := range jobsList {
				if jobsList[i].ID == jobID {
					jobsList[i].Done = true
					return
				}
			}
		}(job.ID)

		return
	}

	// executes the process in the foreground synchronously and waits for it to end
	if err := cmd.Run(); err != nil {

		var exitErr *exec.ExitError

		// avoids printing an error if the program ran but returned a non-zero exit code
		if errors.As(err, &exitErr) {
			return
		}

		fmt.Printf("%s: command not found\n", fields[0])
	}

}