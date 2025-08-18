package main

import (
	"fmt"
	"os"
)

// printPath outputs a file or directory path to the console.
func printPath(path string) error {
	_, err := fmt.Fprintln(console.Out, path)
	return err
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
