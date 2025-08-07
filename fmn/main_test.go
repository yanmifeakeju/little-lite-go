package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
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

func TestListFileSingle(t *testing.T) {
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

func TestListFileMixed(t *testing.T) {
	tDir := []string{}
	directoryCount := 2
	fileCount := 2
	fileIndex := 1

	// we are creating two directories with two files
	for len(tDir) < directoryCount {
		var dir []testFile
		for i := range fileCount {
			dir = append(dir, testFile{path: fmt.Sprintf("file%d.txt", i+1), content: "", mode: 0})
		}

		dirName := setupTestDirWithFiles(t, dir)
		tDir = append(tDir, dirName)
	}

	// Creating the file
	fName := "testing-file.txt"
	fDir := t.TempDir()
	tFile := filepath.Join(fDir, fName)
	if err := os.WriteFile(tFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	tDir = slices.Insert(tDir, fileIndex, tFile)
	if err := listFiles(command{}, tDir, &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Sanity check - should have at least some lines
	if len(lines) < 5 {
		t.Fatalf("too few lines: %d", len(lines))
	}

	// Expected structure: [dir1, file, dir2] creates:
	// Line 0: dir1 header
	// Line 1-2: dir1 files
	// Line 3: blank
	// Line 4: standalone file
	// Line 5: blank
	// Line 6: dir2 header
	// Line 7-8: dir2 files

	// Line 0: should be directory header
	if !strings.HasSuffix(lines[0], ":") {
		t.Errorf("line 0 should be directory header, got %q", lines[0])
	}

	// Lines 1-2: should be files (not empty, not headers)
	for i := 1; i <= 2; i++ {
		if lines[i] == "" || strings.HasSuffix(lines[i], ":") {
			t.Errorf("line %d should be a file, got %q", i, lines[i])
		}
	}

	// Line 3: should be blank
	if lines[3] != "" {
		t.Errorf("line 3 should be blank, got %q", lines[3])
	}

	// Line 4: should be the standalone file path
	if lines[4] != tFile {
		t.Errorf("line 4 should be %q, got %q", tFile, lines[4])
	}

	// Line 5: should be blank
	if len(lines) > 5 && lines[5] != "" {
		t.Errorf("line 5 should be blank, got %q", lines[5])
	}

	// Line 6: should be directory header
	if len(lines) > 6 && !strings.HasSuffix(lines[6], ":") {
		t.Errorf("line 6 should be directory header, got %q", lines[6])
	}

	// Lines 7-8: should be files
	for i := 7; i <= 8 && i < len(lines); i++ {
		if lines[i] == "" || strings.HasSuffix(lines[i], ":") {
			t.Errorf("line %d should be a file, got %q", i, lines[i])
		}
	}

}

func TestListFileErrors(t *testing.T) {
	var buf bytes.Buffer
	t.Run("NonExistentFile", func(t *testing.T) {

		err := listFiles(command{}, []string{"no-file-like-this.txt"}, &buf)
		if err == nil {

			t.Errorf("expected error but got nil")
		}

		if !strings.Contains(err.Error(), "stat no-file-like-this.txt: no such file or directory") {
			t.Errorf("expected %v lines, got %v", "stat no-file-like-this.txt: no such file or directory", err.Error())
		}
	})

	t.Run("FilePermissonDenied", func(t *testing.T) {
		// Save original logger and restore after test
		oldLogger := errorLogger
		defer func() { errorLogger = oldLogger }()

		// Capture error messages
		var errBuf bytes.Buffer
		errorLogger = log.New(&errBuf, "fmn: ", 0)

		dir := t.TempDir()

		// Create a subdirectory and files inside it
		subdir := filepath.Join(dir, "restricted")
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatal(err)
		}

		// Add some files so we know it's not empty
		testFile := filepath.Join(subdir, "file.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}

		// Remove read permission from directory (but keep execute so we can stat)
		if err := os.Chmod(subdir, 0311); err != nil {
			t.Fatal(err)
		}

		// Restore permissions after test
		defer os.Chmod(subdir, 0755)

		var buf bytes.Buffer
		err := listFiles(command{}, []string{subdir}, &buf)
		if err == nil {
			t.Error("expected error due to permission denied")
		}

		// Should show directory header but no files due to permission error
		output := buf.String()
		if !strings.Contains(output, subdir+":") {
			t.Error("should show directory header despite permission error")
		}

		// Check that error was logged
		errOutput := errBuf.String()
		if !strings.Contains(errOutput, "Error reading") {
			t.Errorf("expected 'Error reading' in error log, got: %q", errOutput)
		}
		if !strings.Contains(errOutput, "permission denied") {
			t.Errorf("expected 'permission denied' in error log, got: %q", errOutput)
		}

	})
}

func TestCopyFile(t *testing.T) {

	t.Run("SingleFile", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		// Create a source file with known content
		srcFiles := []testFile{{path: "file1.txt", content: "test content", mode: 0644}}
		srcDir := setupTestDirWithFiles(t, srcFiles)
		srcFilePath := filepath.Join(srcDir, "file1.txt")

		destDir := t.TempDir()

		// [source1, source2, ..., destination]
		directories := []string{srcFilePath, destDir}

		if err := copyFile(cmd, directories, &buf); err != nil {
			t.Fatal(err)
		}

		expectedDestFile := filepath.Join(destDir, "file1.txt")

		// Check the copied file exist in the destination directory
		if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
			t.Fatalf("file was not copied to destination directory: %s", expectedDestFile)
		}

		// Check content
		copiedContent, err := os.ReadFile(expectedDestFile)
		if err != nil {
			t.Fatalf("failed to read copied file: %v", err)
		}

		expectedContent := "test content"
		if string(copiedContent) != expectedContent {
			t.Errorf("copied content mismatch: expected %q, got %q", expectedContent, string(copiedContent))
		}

		// Check if permissions are preserved
		srcInfo, err := os.Stat(srcFilePath)
		if err != nil {
			t.Fatalf("failed to stat source file: %v", err)
		}

		destInfo, err := os.Stat(expectedDestFile)
		if err != nil {
			t.Fatalf("failed to stat destination file: %v", err)
		}

		srcMode := srcInfo.Mode().Perm()
		destMode := destInfo.Mode().Perm()

		if srcMode != destMode {
			t.Errorf("permissions not preserved: source %o, destination %o", srcMode, destMode)
		}

		// Check if modification times are preserved
		srcModTime := srcInfo.ModTime()
		destModTime := destInfo.ModTime()

		// Using Equal() method for time comparison is more reliable than ==
		if !srcModTime.Equal(destModTime) {
			t.Errorf("modification times not preserved: source %v, destination %v",
				srcModTime, destModTime)
		}
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		// Create  source files with known content
		srcFiles := []testFile{
			{path: "file1.txt", content: "content of file1", mode: 0644},
			{path: "file2.txt", content: "content of file2", mode: 0644},
			{path: "file3.txt", content: "content of file3", mode: 0644},
			{path: "file4.txt", content: "content of file4", mode: 0644},
		}
		srcDir := setupTestDirWithFiles(t, srcFiles)

		// [source1, source2, ..., destination]
		directories := make([]string, len(srcFiles))
		for i, files := range srcFiles {
			directories[i] = filepath.Join(srcDir, files.path)
		}

		destDir := t.TempDir()

		directories = append(directories, destDir)

		if err := copyFile(cmd, directories, &buf); err != nil {
			t.Fatal(err)
		}

		for i, f := range srcFiles {
			expectedDestFile := filepath.Join(destDir, f.path)

			// Check the copied file exist in the destination directory
			if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
				t.Fatalf("file was not copied to destination directory: %s", expectedDestFile)
			}

			// Check content
			copiedContent, err := os.ReadFile(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to read copied file: %v", err)
			}

			expectedContent := f.content
			if string(copiedContent) != expectedContent {
				t.Errorf("copied content mismatch: expected %q, got %q", expectedContent, string(copiedContent))
			}

			// Check if permissions are preserved
			srcInfo, err := os.Stat(directories[i])
			if err != nil {
				t.Fatalf("failed to stat source file: %v", err)
			}

			destInfo, err := os.Stat(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to stat destination file: %v", err)
			}

			srcMode := srcInfo.Mode().Perm()
			destMode := destInfo.Mode().Perm()

			if srcMode != destMode {
				t.Errorf("permissions not preserved: source %o, destination %o", srcMode, destMode)
			}

			// Check if modification times are preserve
			srcModTime := srcInfo.ModTime()
			destModTime := destInfo.ModTime()

			if !srcModTime.Equal(destModTime) {
				t.Errorf("modification times not preserved for %s: source %v, destination %v",
					f.path, srcModTime, destModTime)
			}
		}

	})

	// t.Run("RecursiveDirectory", func(t *testing.T) { ... })
	// t.Run("ErrorCases", func(t *testing.T) { ... })
}

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

		if err := os.WriteFile(fullPath, []byte(f.content), mode); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}
	return dir
}
