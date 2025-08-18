package main

import (
	"fmt"
	"os"
)

// listFiles lists the contents of the given directories and files.
// For directories, it prints the directory name followed by a colon and lists all files.
// For regular files, it prints the file path directly.
// Blank lines are printed between different items for readability.
func listFiles(_ command, directories []string) error {
	// Pre-validate all paths first
	srcInfos := make([]os.FileInfo, len(directories))
	for i, src := range directories {
		srcInfo, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("cannot stat '%s': %w", src, err)
		}

		srcInfos[i] = srcInfo
	}

	var hasErrors bool
	needsBlankLine := true // track printing lines between directories
	for i, path := range directories {
		if i > 0 && needsBlankLine {
			fmt.Fprintln(console.Out) // Blank line between directories
		}

		info := srcInfos[i]

		if !info.IsDir() {
			printPath(path)
			needsBlankLine = true // Files should have blank lines after them
			continue
		}

		fmt.Fprintf(console.Out, "%s:\n", path)

		files, err := os.ReadDir(path)
		if err != nil {
			errorLogger.Printf("Error reading %s: %v", path, err)
			hasErrors = true
			continue
		}

		for _, f := range files {
			printPath(f.Name())
		}

		needsBlankLine = true // Directories should have blank lines after them
	}

	if hasErrors {
		return fmt.Errorf("some directories could not be read")
	}
	return nil
}
