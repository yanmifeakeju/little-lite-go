package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	archiveDir := flag.String("archive", "", "Archive directory to restor from")
	destDir := flag.String("dest", "", "Destination directory")
	list := flag.Bool("list", false, "List files that would be restored")
	force := flag.Bool("force", false, "Overwrite existing files without asking")

	flag.Parse()

	if *archiveDir == "" {
		fmt.Fprintln(os.Stderr, "Error: -archive flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *destDir == "" {
		*destDir = "."
	}

	if err := restore(*archiveDir, *destDir, *list, *force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func restore(archiveDir, destDir string, list, force bool) error {
	if d, err := os.Stat(archiveDir); err != nil || !d.IsDir() {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s is not directory", archiveDir)
	}

	if d, err := os.Stat(destDir); err != nil || !d.IsDir() {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s is not directory", destDir)
	}

	return filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only process .gz files
		if filepath.Ext(path) != ".gz" {
			return nil
		}

		relDir, err := filepath.Rel(archiveDir, filepath.Dir(path))
		if err != nil {
			return err
		}

		sf, err := os.Open(path)
		if err != nil {
			return err
		}

		defer sf.Close()

		zr, err := gzip.NewReader(sf)
		if err != nil {
			return err
		}

		defer zr.Close()

		dest := filepath.Join(destDir, relDir, zr.Name)

		if list {
			fmt.Printf("Would restore: %s -> %s\n", path, dest)
			return nil
		}

		// Check if file exists and ask for confirmation
		if !force {
			if _, err := os.Stat(dest); err == nil {
				if !askConfirmation(fmt.Sprintf("File %s already exists. Overwrite? (y/N): ", dest)) {
					fmt.Printf("Skipped: %s\n", dest)
					return nil
				}
			}
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		df, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		defer df.Close()

		if _, err := io.Copy(df, zr); err != nil {
			return err
		}

		// Preserve timestamp from gzip header if available
		if !zr.ModTime.IsZero() {
			if err := os.Chtimes(dest, zr.ModTime, zr.ModTime); err != nil {
				// Don't fail if we can't set timestamp, just warn
				fmt.Printf("Warning: Could not preserve timestamp for %s: %v\n", dest, err)
			}
		}

		fmt.Printf("Restored: %s\n", dest)
		return nil
	})

}

func askConfirmation(prompt string) bool {
	return askConfirmationFromReader(prompt, os.Stdin)
}

func askConfirmationFromReader(prompt string, reader io.Reader) bool {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(reader)
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}
