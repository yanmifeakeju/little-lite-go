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
	// --- Custom Usage Message ---
	flag.Usage = func() {
		// Use the standard error output defined in our console struct
		w := console.Err

		// Program description
		fmt.Fprintf(w, "fmn is a simple file management tool.\n\n")

		// Usage for the default (list) command
		fmt.Fprintf(w, "Usage: fmn [options] [path...]\n")
		fmt.Fprintf(w, "Lists the contents of one or more paths (defaults to current directory).\n\n")

		// Usage for the copy command
		fmt.Fprintf(w, "Usage: fmn -copy [options] <source> <destination>\n")
		fmt.Fprintf(w, "       fmn -copy [options] <source...> <directory>\n")
		fmt.Fprintf(w, "Copies files and directories.\n\n")

		// Print the list of available flags
		fmt.Fprintf(w, "Options:\n")
		flag.PrintDefaults()
	}

	// Copy options
	copy := flag.Bool("copy", false, "Enable copying")
	recursive := flag.Bool("r", false, "Copy files recursively")
	force := flag.Bool("f", false, "Force overwrite of existing files")
	interactive := flag.Bool("i", false, "Prompt before overwrite")
	verbose := flag.Bool("v", false, "Enable verbose output")
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
		errorLogger.Println(err)
		os.Exit(1)
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

// copyFile manages the overall copy operation. It validates the destination,
// then iterates through the source paths, calling copySource for each one.
// It collects and returns any errors that occur.
func copyFile(cmd command, directories []string) error {
	lastIndex := len(directories) - 1
	dest := directories[lastIndex]
	sources := directories[:lastIndex]

	destInfo, err := os.Stat(dest)
	if err != nil {
		return fmt.Errorf("cannot stat destination '%s': %w", dest, err)
	}

	if len(sources) > 1 && !destInfo.IsDir() {
		return fmt.Errorf("target '%s' is not a directory", dest)
	}

	var errs []error
	for _, src := range sources {
		if err := copySource(cmd, src, dest, destInfo); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// copySource handles the logic for copying a single source path (which can be
// a file or a directory) to the destination.
func copySource(cmd command, src, dest string, destInfo os.FileInfo) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("cannot stat source '%s': %w", src, err)
	}

	if srcInfo.IsDir() {
		return copyDirectory(cmd, src, dest, destInfo)
	}
	return copySingleFile(cmd, src, dest, srcInfo, destInfo)
}

// copyDirectory handles the logic for recursively copying a directory.
func copyDirectory(cmd command, src, dest string, destInfo os.FileInfo) error {
	if !cmd.recursive {
		return fmt.Errorf("omitting directory '%s' (use -r for recursive)", src)
	}

	if !destInfo.IsDir() {
		return fmt.Errorf("cannot overwrite non-directory '%s' with directory '%s'", dest, src)
	}

	// Walk the source directory
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err // Propagate errors from WalkDir itself
		}

		// Determine the corresponding path in the destination
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, relPath)

		// Nothing to do for the source directory itself
		if path == src {
			return nil
		}

		// Check if we should proceed
		targetInfo, statErr := os.Stat(targetPath)
		if statErr != nil && !os.IsNotExist(statErr) {
			return fmt.Errorf("failed to stat target '%s': %w", targetPath, statErr)
		}

		should, err := shouldOverwrite(targetPath, targetInfo, cmd)
		if err != nil {
			return err
		}
		if !should {
			// If we skip a directory, we must use SkipDir to prevent walking its contents.
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil // Skip file
		}

		// Perform the copy action
		if d.IsDir() {
			return createDir(targetPath, cmd)
		}

		fileInfo, err := d.Info()
		if err != nil {
			return err
		}
		return copySrcToDest(path, targetPath, fileInfo, cmd)
	})
}

// copySingleFile handles the logic for copying a single file to a destination.
func copySingleFile(cmd command, src, dest string, srcInfo, destInfo os.FileInfo) error {
	// Determine the final destination path.
	finalDest := dest
	if destInfo.IsDir() {
		finalDest = filepath.Join(dest, filepath.Base(src))
	}

	// Check for self-copy.
	if same, err := isSameFile(src, finalDest); err == nil && same {
		return fmt.Errorf("cannot copy '%s' to itself", src)
	}

	// Check if we should overwrite the destination.
	finalDestInfo, statErr := os.Stat(finalDest)
	if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("failed to check destination '%s': %w", finalDest, statErr)
	}

	should, err := shouldOverwrite(finalDest, finalDestInfo, cmd)
	if err != nil {
		return err
	}
	if !should {
		return nil // Skip file as requested.
	}

	// Perform the actual copy.
	return copySrcToDest(src, finalDest, srcInfo, cmd)
}

// isSameFile checks if two paths refer to the same underlying file.
func isSameFile(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(b)
	if err != nil {
		// If the destination doesn't exist, it can't be the same file.
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return os.SameFile(infoA, infoB), nil
}
