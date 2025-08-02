package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFile(t *testing.T) {

	testCases := []struct {
		name           string
		fileCount      int
		directoryCount int
	}{
		{
			name:      "SingleDirectoryWithFiles",
			fileCount: 4,
		},
		{
			name:      "EmptyDirectory",
			fileCount: 0,
		},
		{
			name:           "MultipleDirectories",
			fileCount:      2,
			directoryCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var dirs []string
			directoryCount := tc.directoryCount
			if directoryCount == 0 {
				directoryCount = 1
			}

			for i := 0; i < directoryCount; i++ {
				// Create test Files
				files := make([]testFile, tc.fileCount)
				for i := 0; i < tc.fileCount; i++ {
					files[i] = testFile{path: fmt.Sprintf("file%d.txt", i+1), content: "", mode: 0}
				}

				dir := setupTestDirWithFiles(t, files)
				dirs = append(dirs, dir)
			}

			var buf bytes.Buffer
			cmd := command{}

			if err := listFiles(cmd, dirs, &buf); err != nil {
				t.Fatal(err)
			}

			output := buf.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")

			// Each directory: header + files
			// Plus blank lines between directories (directoryCount - 1)
			expectedLines := directoryCount*(tc.fileCount+1) + (directoryCount - 1)
			if len(lines) != expectedLines {
				t.Errorf("expected %d lines, got %d", expectedLines, len(lines))
			}

			// Validate structure for multiple directories
			lineIndex := 0
			for dirNum := 0; dirNum < directoryCount; dirNum++ {
				// Check directory header
				if lineIndex >= len(lines) {
					t.Errorf("missing directory header for directory %d", dirNum+1)
					break
				}
				if !strings.HasSuffix(lines[lineIndex], ":") {
					t.Errorf("line %d should be directory header, got %q", lineIndex+1, lines[lineIndex])
				}
				lineIndex++

				// Check files for this directory
				for fileNum := 0; fileNum < tc.fileCount; fileNum++ {
					if lineIndex >= len(lines) {
						t.Errorf("missing file %d for directory %d", fileNum+1, dirNum+1)
						break
					}
					if lines[lineIndex] == "" {
						t.Errorf("line %d should not be empty (file %d of dir %d)", lineIndex+1, fileNum+1, dirNum+1)
					}
					lineIndex++
				}

				// Check for blank line between directories (except after last directory)
				if dirNum < directoryCount-1 {
					if lineIndex >= len(lines) {
						t.Errorf("missing blank line after directory %d", dirNum+1)
						break
					}
					if lines[lineIndex] != "" {
						t.Errorf("line %d should be blank line between directories, got %q", lineIndex+1, lines[lineIndex])
					}
					lineIndex++
				}
			}

		})
	}

}

func TestListFileSingleFile(t *testing.T) {
	// Create a single test file
	dir := t.TempDir()
	testFilePath := filepath.Join(dir, "testfile.txt")

	if err := os.WriteFile(testFilePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := command{}

	// Pass the FILE path, not directory
	if err := listFiles(cmd, []string{testFilePath}, &buf); err != nil {
		t.Fatal(err)
	}

	output := strings.TrimSpace(buf.String())

	// Should be the full file path - no directory header
	if output != testFilePath {
		t.Errorf("expected %q, got %q", testFilePath, output)
	}
}

func TestCopyFile(t *testing.T) {}

type testFile struct {
	path    string
	content string
	mode    os.FileMode
}

func setupTestDirWithFiles(t *testing.T, files []testFile) string {
	t.Helper()

	dir := t.TempDir()

	for _, f := range files {
		fullPath := filepath.Join(dir, f.path)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", fullPath, err)
		}

		mode := f.mode
		if mode == 0 {
			mode = 0644
		}

		if err := os.WriteFile(fullPath, []byte(f.content), f.mode); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}
	return dir
}
