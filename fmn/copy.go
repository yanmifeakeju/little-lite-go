package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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

// copySrcToDest performs the actual file copy operation with permission and timestamp preservation.
func copySrcToDest(src, dst string, srcInfo os.FileInfo, cmd command) error {
	if cmd.dryRun {
		fmt.Fprintf(console.Out, "would copy '%s' -> '%s'\n", src, dst)
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// Use the passed srcInfo for permissions and timestamps
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return err
	}

	if cmd.verbose {
		fmt.Fprintf(console.Out, "'%s' -> '%s'\n", src, dst)
	}

	return nil
}

// createDir creates a directory with appropriate permissions.
func createDir(path string, cmd command) error {
	if cmd.dryRun {
		fmt.Fprintf(console.Out, "would create directory '%s'\n", path)
		return nil
	}

	return os.MkdirAll(path, 0755)
}

// prompt asks the user for confirmation before overwriting a file.
func prompt(dst string) bool {
	fmt.Fprintf(console.Out, "overwrite '%s'? (y/n): ", dst)
	scanner := bufio.NewScanner(console.In)
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

// shouldOverwrite determines if a file or directory at targetPath should be overwritten.
// It returns a boolean indicating if the operation should proceed and an error
// if the operation should be aborted due to a file conflict.
func shouldOverwrite(targetPath string, targetInfo os.FileInfo, cmd command) (bool, error) {
	// If the target path doesn't exist, we can proceed.
	if targetInfo == nil {
		return true, nil
	}

	// If the target is a directory, we don't need to "overwrite" it.
	// The recursive copy logic will handle creating files/subdirectories inside it.
	if targetInfo.IsDir() {
		return true, nil
	}

	// At this point, the target is a file that already exists.
	// We need to decide whether to overwrite it based on the command flags.
	if cmd.force {
		// Force flag is set, so we overwrite.
		return true, nil
	}

	if cmd.interactive {
		// Interactive flag is set, so we ask the user.
		if prompt(targetPath) {
			return true, nil // User said yes.
		}
		// User said no; skip the file, but it's not an error.
		return false, nil
	}

	// Default behavior: file exists, and no -f or -i is provided.
	// This is an error condition.
	err := fmt.Errorf("'%s' already exists (use -f to force or -i for interactive)", targetPath)
	return false, err
}
