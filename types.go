package main

// represents a single background task tracked by the shell
type Job struct {
	// unique numeric identifier assigned to the job by the shell
	ID      int
	// operating system process id number for the running command
	PID     int
	// the original command line text string used to launch the job
	Command string
	// flag that marks whether the background process has finished running
	Done    bool
}