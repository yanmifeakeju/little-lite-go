package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// console provides global access to I/O streams for input, output, and error logging
// Can be overridden in tests for easier testing
var console = struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}{
	In:  os.Stdin,
	Out: os.Stdout,
	Err: os.Stderr,
}

// errorLogger writes error messages to stderr with consistent formatting
// Can be overridden in tests
var errorLogger = log.New(console.Err, "fmn: ", 0)

// command holds the configuration flags for the file management operations.
// It contains options for both copy and list operations.
type command struct {
	// Copy options
	copy        bool
	recursive   bool
	force       bool
	interactive bool
	verbose     bool
	dryRun      bool
}

func main() {
	// Copy options
	copy := flag.Bool("copy", false, "Enable copying")
	recursive := flag.Bool("r", false, "Copy files recursively")
	force := flag.Bool("f", false, "Force overwrite of existing files")
	interactive := flag.Bool("i", false, "Prompt before overwrite")
	verbose := flag.Bool("v", false, "Show ")
	dryRun := flag.Bool("dry-run", false, "Show what would be copied without actually copying")

	flag.Parse()

	cmd := command{
		copy:        *copy,
		recursive:   *recursive,
		force:       *force,
		interactive: *interactive,
		verbose:     *verbose,
		dryRun:      *dryRun,
	}

	// Get remaining args as paths to process (files or directories)
	dirs := flag.Args()

	if err := run(cmd, dirs); err != nil {
		flag.Usage()
		errorLogger.Fatal(err)
	}
}

func run(cmd command, directories []string) error {
	if cmd.copy {
		if len(directories) == 0 {
			return errors.New("copiles requires at least one source path")
		}

		if len(directories) == 1 {
			directories = append(directories, ".") // Add default destination
		}

		if directories[0] == directories[len(directories)-1] {
			return errors.New("cannot copy a path to itself") // Quick catch for . . or file to file
		}

		return copyFile(cmd, directories)
	}

	if len(directories) == 0 {
		directories = []string{"."} // Default to current directory
	}

	return listFiles(cmd, directories)
}

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

// copyFile copies files and directories from source(s) to destination.
// The last argument in directories is treated as the destination, all others as sources.
// It supports recursive copying with the -r flag, preserves file permissions and timestamps,
// and handles same-file detection and directory validation.
// Errors are reported to stderr and the function returns an error if any copies fail.
func copyFile(cmd command, directories []string) error {
	// fmt.Println(directories)
	lastIndex := len(directories) - 1
	dest := directories[lastIndex]
	sources := directories[:lastIndex]

	var hasErrors bool

	destInfo, err := os.Stat(dest)
	if err != nil {
		return err
	}

	if len(sources) > 1 && !destInfo.IsDir() {
		return fmt.Errorf("target '%s': Not a directory", dest)
	}

	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	// Pre-validate and collect source info
	srcInfos := make([]os.FileInfo, len(sources))
	for i, src := range sources {
		if src == dest {
			return fmt.Errorf("cannot copy '%s' to itself", src)
		}

		srcInfo, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("cannot stat '%s': %w", src, err)
		}

		if srcInfo.IsDir() && !destInfo.IsDir() {
			return fmt.Errorf("cannot overwrite non-directory '%s' with directory '%s'", dest, src)
		}

		srcInfos[i] = srcInfo
	}

	for i, src := range sources {
		srcAbs, err := filepath.Abs(src)
		if err != nil {
			errorLogger.Printf("%v", err)
			hasErrors = true
			continue
		}

		srcInfo := srcInfos[i]

		if srcInfo.IsDir() {
			if !cmd.recursive {
				errorLogger.Printf("omitting directory '%s' (use -r for recursive)", src)
				hasErrors = true
				continue
			}

			// Recursive directory copying
			if err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Get relative path from source root to current path
				relPath, err := filepath.Rel(src, path)
				if err != nil {
					return err
				}

				// Build target path: dest + relative path
				targetPath := filepath.Join(dest, relPath)

				// Check if target already exists
				targetInfo, err := os.Stat(targetPath)
				shouldCopy, isError := shouldOverwrite(targetPath, targetInfo, cmd)
				if !shouldCopy {
					if isError && err == nil {
						// File exists but we shouldn't overwrite
						return nil // Skip this file/directory
					} else if err != nil && !os.IsNotExist(err) {
						// Some other error occurred
						return err
					}
					return nil // User said no or file doesn't exist
				}

				if d.IsDir() {
					// Create directory
					return createDir(targetPath, cmd)
				} else {
					// It's a file - ensure parent directory exists first
					parentDir := filepath.Dir(targetPath)
					if err := createDir(parentDir, cmd); err != nil {
						return err
					}

					// Get file info for copying permissions
					fileInfo, err := d.Info()
					if err != nil {
						return err
					}

					// Copy the file
					return copySrcToDest(path, targetPath, fileInfo, cmd)
				}
			}); err != nil {
				errorLogger.Printf("Error copying directory '%s': %v", src, err)
				hasErrors = true
				continue
			}

		} else {
			// this is a regular file and we can just copy it
			// after checking if they are the same file
			var finalDest string
			if destInfo.IsDir() {
				finalDest = filepath.Join(destAbs, filepath.Base(srcAbs))
			} else {
				finalDest = destAbs
			}

			if finalDestInfo, err := os.Stat(finalDest); err == nil {
				if os.SameFile(srcInfo, finalDestInfo) {
					errorLogger.Printf("'%s' are the same file '%s'", dest, src)
					hasErrors = true
					continue
				}

				// Check if we should overwrite using existing stat info
				shouldCopy, isError := shouldOverwrite(finalDest, finalDestInfo, cmd)
				if !shouldCopy {
					if isError {
						hasErrors = true
					}
					continue
				}
			} else if !os.IsNotExist(err) {
				// Some other error occurred during stat
				errorLogger.Printf("Error checking destination '%s': %v", finalDest, err)
				hasErrors = true
				continue
			}
			// If file doesn't exist (os.IsNotExist), we proceed with copying

			if err := copySrcToDest(srcAbs, finalDest, srcInfo, cmd); err != nil {
				errorLogger.Printf("%v", err)
				hasErrors = true
				continue
			}
		}

	}

	if hasErrors {
		return fmt.Errorf("some files could not be copied")
	}
	return nil
}
