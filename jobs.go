package main

import (
	"fmt"
	"strings"
	"syscall"
	"os"
	"sync"
)

// stores the list of all active or finished background tasks
var jobsList []Job
// mutual exclusion lock to stop multiple threads from changing the job list at the same time
var jobsMutex sync.Mutex

// loops through active jobs to find the next free numeric identifier
func getNextJobID() int {
	used := make(map[int]bool)

	for _, job := range jobsList {
		used[job.ID] = true
	}

	id := 1

	for {
		if !used[id] {
			return id
		}
		id++
	}
}

// prints out the running or completed status of all background tasks
func handleJobs() {
	var remaining []Job

	for i, job := range jobsList {
		marker := " "

		// assigns shell symbols to denote the current and previous background tasks
		if i == len(jobsList)-1 {
			marker = "+"
		} else if i == len(jobsList)-2 {
			marker = "-"
		}

		// prints a completed message if the background task has finished working
		if job.Done {
			fmt.Printf(
				"[%d]%s  %-24s%s\n",
				job.ID,
				marker,
				"Done",
				strings.TrimSuffix(job.Command, " &"),
			)
		} else {
			fmt.Printf(
				"[%d]%s  %-24s%s\n",
				job.ID,
				marker,
				"Running",
				job.Command,
			)

			remaining = append(remaining, job)
		}
	}

	jobsList = remaining
}

// finds completed background tasks, prints their status, and cleans them out of memory
func reapJobs() {
	var remaining []Job

	for i, job := range jobsList {

		marker := " "

		if i == len(jobsList)-1 {
			marker = "+"
		} else if i == len(jobsList)-2 {
			marker = "-"
		}

		if job.Done {

			fmt.Printf(
				"[%d]%s  %-24s%s\n",
				job.ID,
				marker,
				"Done",
				strings.TrimSuffix(job.Command, " &"),
			)

		} else {
			remaining = append(remaining, job)
		}
	}

	jobsList = remaining
}

// pings the operating system kernel to check if running background processes have stopped
func refreshJobs() {
	for i := range jobsList {

		if jobsList[i].Done {
			continue
		}

		p, err := os.FindProcess(jobsList[i].PID)
		if err != nil {
			continue
		}

		// sends signal zero to safely check if the process id is still alive or dead
		err = p.Signal(syscall.Signal(0))

		if err != nil {
			jobsList[i].Done = true
		}
	}
}