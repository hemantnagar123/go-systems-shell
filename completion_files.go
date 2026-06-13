package main

import (
	"os"
	"sort"
	"strings"
)

// finds files or folders that match the partial path and can filter out non-directories
func findFileMatches(partial string, dirOnly bool) []string {
	dir := "."
	prefix := partial

	// locates the last slash character to separate the directory path from the file prefix
	lastSlash := strings.LastIndex(partial, "/")
	var leadingPath string
	if lastSlash != -1 {
		leadingPath = partial[:lastSlash+1] 
		dir = partial[:lastSlash]
		if dir == "" {
			dir = "/" 
		}
		prefix = partial[lastSlash+1:]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string

	for _, entry := range entries {
		name := entry.Name()

		if strings.HasPrefix(name, prefix) {
			
			// skips files if the command only accepts directories like cd
			if dirOnly && !entry.IsDir() {
				continue
			}

			// rebuilds the full path by combining the parent directory structure with the matching item name
			full := leadingPath + name

			if entry.IsDir() {
				matches = append(matches, full+"/")
			} else {
				matches = append(matches, full)
			}
		}
	}

	sort.Strings(matches)
	return matches
}

// appends a trailing space for file matches or keeps the trailing slash for folders
func completeFileMatch(match string) string {
	if strings.HasSuffix(match, "/") {
		return match
	}
	return match + " "
}

// trims the trailing slash from the end of a directory name
func normalizeMatch(match string) string {
	return strings.TrimSuffix(match, "/")
}

// removes trailing slashes from a list of matches to find their longest shared starting path
func longestCommonPrefixFiles(matches []string) string {
	if len(matches) == 0 {
		return ""
	}

	normalized := make([]string, len(matches))
	for i, m := range matches {
		normalized[i] = strings.TrimSuffix(m, "/")
	}

	return longestCommonPrefix(normalized)
}

// compares a list of text strings to extract the longest common starting prefix among them
func longestCommonPrefix(matches []string) string {
	if len(matches) == 0 {
		return ""
	}

	prefix := matches[0]
	for _, s := range matches[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}

	return prefix
}