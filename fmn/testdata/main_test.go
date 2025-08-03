package main

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test the restore function with real files
func TestRestore(t *testing.T) {
	// Setup the test directories
	archiveDir := setUpTestDir(t)
	destDir := setUpTestDir(t)

	// create test data
	createTestGzFile(t, archiveDir, "test1.txt", "Hello World")

	// create subdirectory instructure in the archive
	subArchiveDir := filepath.Join(archiveDir, "subdir")
	createTestGzFile(t, subArchiveDir, "test2.txt", "Hello Subdir")

	t.Run("List mode", func(t *testing.T) {
		err := restore(archiveDir, destDir, true, false)
		if err != nil {
			t.Fatalf("Restore failed: %v", err)
		}

		// Verify files were NOT actually restored
		if _, err := os.Stat(filepath.Join(destDir, "test1.txt")); err == nil {
			t.Error("File should not exist in list mode")
		}
	})

	t.Run("Actual Restore", func(t *testing.T) {
		// force=true to skip prompts
		if err := restore(archiveDir, destDir, false, true); err != nil {
			t.Fatalf("Restore failed: %v", err)
		}

		content1, err := os.ReadFile(filepath.Join(destDir, "test1.txt"))
		if err != nil {
			t.Fatalf("Failed to read restored file: %v", err)
		}

		if string(content1) != "Hello World" {
			t.Errorf("Expected 'Hello World', got %q", string(content1))
		}

		content2, err := os.ReadFile(filepath.Join(destDir, "subdir", "test2.txt"))
		if err != nil {
			t.Fatalf("Failed to read restored file: %v", err)
		}

		if string(content2) != "Hello Subdir" {
			t.Errorf("Expected 'Hello Subdir', got %q", string(content2))
		}

	})

}

func TestAskConfirmation(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Yes lowercase", "y\n", true},
		{"Yes uppercase", "Y\n", true},
		{"Yes full world", "yes\n", true},
		{"Yes full word uppercase", "YES\n", true},
		{"No lowercase", "n\n", false},
		{"No full word", "\no", false},
		{"Empty input", "\n", false},
		{"With spaces", "  y\n", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run(tc.name, func(t *testing.T) {
				reader := strings.NewReader(tc.input)
				result := askConfirmationFromReader("Test prompt: ", reader)
				if result != tc.expected {
					t.Errorf("Expected %v, got %v for input %q", tc.expected, result, tc.input)
				}
			})
		})
	}
}

// setupTestDir creates a temporary directory for testing with automatic cleanup
func setUpTestDir(t *testing.T) string {
	return t.TempDir()
}

// createTestGzFile createa a gzipped file with the given content
func createTestGzFile(t *testing.T, dir, filename, content string) {
	fullPath := filepath.Join(dir, filename+".gz")

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("Failed to created dir for %s: %v", fullPath, err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("Failed to created file %s: %v", fullPath, err)
	}

	defer file.Close()

	zw := gzip.NewWriter(file)
	zw.Name = filename
	zw.ModTime = time.Now()

	if _, err := io.WriteString(zw, content); err != nil {
		t.Fatalf("Failed to write content to %s: %v", fullPath, err)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer for %s: %v", fullPath, err)
	}

}
