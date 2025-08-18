// Package main implements fmn, a simple file management tool for listing and copying files.
// It provides functionality similar to basic ls and cp commands with additional features
// like dry-run mode, interactive prompts, and verbose output.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

// console provides global access to I/O streams for input, output, and error logging
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
			return errors.New("copy requires at least one source path")
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
