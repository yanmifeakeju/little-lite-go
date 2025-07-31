package main

import (
	"fmt"
	"io"
	"os"
)

func printPath(path string, w io.Writer) error {
	_, err := fmt.Fprintln(w, path)
	return err
}

func copySrcToDest(src, dst string, srcInfo os.FileInfo, out io.Writer, cmd command) error {

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
		fmt.Fprintf(out, "'%s' -> '%s'\n", src, dst)
	}

	return nil
}

func prompt() bool {
	return false
}
