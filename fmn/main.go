package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type command struct {
	// Copy options
	copy        bool
	recursive   bool
	force       bool
	interactive bool
	verbose     bool
}

func main() {
	// Copy options
	copy := flag.Bool("copy", false, "Enable copying")
	recursive := flag.Bool("r", false, "Copy files recursively")
	force := flag.Bool("f", false, "Force overwrite of existing files")
	interactive := flag.Bool("i", false, "Prompt before overwrite")
	verbose := flag.Bool("v", false, "Log all copies")

	flag.Parse()

	cmd := command{
		copy:        *copy,
		recursive:   *recursive,
		force:       *force,
		interactive: *interactive,
		verbose:     *verbose,
	}

	// Use remaining args as directories to list
	dirs := flag.Args()

	if len(dirs) == 0 {
		dirs = []string{"."} // Default to current directory
	}

	if err := run(cmd, dirs, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd command, directories []string, out io.Writer) error {
	// copy mode
	if cmd.copy {
		if directories[0] == "." {
			return errors.New("cannot copy '.' to itself")
		}

		if len(directories) == 1 {
			directories = append(directories, ".")
		}

		return copyFile(cmd, directories, out)
	}

	return listFiles(cmd, directories, out)
}

func listFiles(_ command, directories []string, out io.Writer) error {
	needsBlankLine := true // track printing lines between directories
	for i, path := range directories {
		if i > 0 && needsBlankLine {
			fmt.Fprintln(out) // Blank line between directories
		}

		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(out, "Error accessing %s: %v\n", path, err)
			continue // Continue with other directories
		}

		if !info.IsDir() {
			needsBlankLine = false
			printPath(path, out)
			continue
		}

		needsBlankLine = true
		fmt.Fprintf(out, "%s:\n", path)

		files, err := os.ReadDir(path)
		if err != nil {
			fmt.Fprintf(out, "Error reading %s: %v\n", path, err)
			continue
		}

		for _, f := range files {
			printPath(f.Name(), out)
		}
	}
	return nil
}

// copyFile copies file(s) from a source directory to a destination directory
// it tries to mimic the behavior of unix's mv command
func copyFile(cmd command, directories []string, out io.Writer) error {
	lastIndex := len(directories) - 1
	sources := directories[:lastIndex]
	dest := directories[lastIndex]

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
			fmt.Fprintln(out, err)
			hasErrors = true
			continue
		}

		srcInfo := srcInfos[i]

		if srcInfo.IsDir() {
			if !cmd.recursive {
				fmt.Fprintf(out, "omitting directory '%s' (use -r for recursive)", src)
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

				if d.IsDir() {
					// Create directory
					return os.MkdirAll(targetPath, 0755)
				} else {
					// It's a file - ensure parent directory exists first
					parentDir := filepath.Dir(targetPath)
					if err := os.MkdirAll(parentDir, 0755); err != nil {
						return err
					}

					// Get file info for copying permissions
					fileInfo, err := d.Info()
					if err != nil {
						return err
					}

					// Copy the file
					return copySrcToDest(path, targetPath, fileInfo, out, cmd)
				}
			}); err != nil {
				fmt.Fprintf(out, "Error copying directory '%s': %v\n", src, err)
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
					fmt.Fprintf(out, "'%s' are the same file '%s'\n", dest, src)
					hasErrors = true
					continue
				}
			}

			if err := copySrcToDest(srcAbs, finalDest, srcInfo, out, cmd); err != nil {
				fmt.Fprintln(out, err)
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
