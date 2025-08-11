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
