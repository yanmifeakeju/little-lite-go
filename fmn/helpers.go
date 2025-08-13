package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func printPath(path string) error {
	_, err := fmt.Fprintln(console.Out, path)
	return err
}

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

func createDir(path string, cmd command) error {
	if cmd.dryRun {
		fmt.Fprintf(console.Out, "would create directory '%s'\n", path)
		return nil
	}

	return os.MkdirAll(path, 0755)
}

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
