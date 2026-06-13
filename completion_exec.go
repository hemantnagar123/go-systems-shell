package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// looks for executable programs that match the partial name typed by the user
func findExecutableMatches(partial string) []string {
	// holds unique matching executable paths to prevent duplicates
	candidates := map[string]struct{}{}

	// checks if the user typed a direct file path containing a slash separator
	if strings.Contains(partial, string(filepath.Separator)) {

		// extracts the directory portion of the typed path
		dir := filepath.Dir(partial)
		// extracts the file prefix portion of the typed path
		base := filepath.Base(partial)

		// reads all files inside the target directory
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}

		// loops through each file entry in the directory
		for _, entry := range entries {
			name := entry.Name()

			// checks if the filename starts with the prefix and is a valid executable program
			if strings.HasPrefix(name, base) &&
				isExecutablePath(filepath.Join(dir, name)) {

				// saves the full file path to the list of unique candidates
				candidates[filepath.Join(dir, name)] = struct{}{}
			}
		}

	} else {

		// fetches the list of system directories from the path environment variable
		pathEnv := os.Getenv("PATH")

		// loops through each individual directory found in the system path
		for _, dir := range strings.Split(pathEnv, string(os.PathListSeparator)) {

			// defaults to the current directory if an entry in the system path is empty
			if dir == "" {
				dir = "."
			}

			// reads all files inside the current system path directory
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}

			// loops through each file entry inside this system directory
			for _, entry := range entries {

				name := entry.Name()

				// checks if the file matches what the user typed and is a valid executable program
				if strings.HasPrefix(name, partial) &&
					isExecutablePath(filepath.Join(dir, name)) {

					// saves the short command name to the list of unique candidates
					candidates[name] = struct{}{}
				}
			}
		}
	}

	// creates an empty text slice to hold the final sorted list
	ordered := make([]string, 0, len(candidates))

	// loops through and moves each unique candidate from the map into the slice list
	for candidate := range candidates {
		ordered = append(ordered, candidate)
	}

	// sorts the final list of matching executable commands in alphabetical order
	sort.Strings(ordered)

	return ordered
}

// verifies if a given file path points to a real executable file
func isExecutablePath(path string) bool {
	// fetches the metadata and status information for the file path
	info, err := os.Stat(path)
	// returns false if the file does not exist or if it is a folder directory
	if err != nil || info.IsDir() {
		return false
	}

	// checks if the shell is running on a windows operating system
	if runtime.GOOS == "windows" {
		return true
	}

	// checks if linux or mac permission bits allow the file to execute
	return info.Mode()&0o111 != 0
}