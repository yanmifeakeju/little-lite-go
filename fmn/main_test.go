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
					files[i] = testFile{filename: fmt.Sprintf("file%d.txt", i+1), content: "test content", mode: 0}
				}

				dir, _ := setupTestDirWithFiles(t, files)
				dirs = append(dirs, dir)
			}

			var buf bytes.Buffer
			cmd := command{}

			if err := run(cmd, dirs, &buf); err != nil {
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
	if err := run(cmd, []string{testFilePath}, &buf); err != nil {
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
			dir = append(dir, testFile{path: ".", filename: fmt.Sprintf("file%d.txt", i+1), content: "test content", mode: 0})
		}

		dirName, _ := setupTestDirWithFiles(t, dir)
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

		err := run(command{}, []string{"no-file-like-this.txt"}, &buf)
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
		err := run(command{}, []string{subdir}, &buf)
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
		srcFiles := []testFile{{path: ".", filename: "file1.txt", content: "test content", mode: 0644}}
		srcDir, _ := setupTestDirWithFiles(t, srcFiles)
		srcFilePath := filepath.Join(srcDir, "file1.txt")

		destDir := t.TempDir()

		// [source1, source2, ..., destination]
		directories := []string{srcFilePath, destDir}

		if err := run(cmd, directories, &buf); err != nil {
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
			{path: "", filename: "file1.txt", content: "content of file1", mode: 0644},
			{path: "", filename: "file2.txt", content: "content of file2", mode: 0644},
			{path: "", filename: "file3.txt", content: "content of file3", mode: 0644},
			{path: "", filename: "file4.txt", content: "content of file4", mode: 0644},
		}

		_, directories := setupTestDirWithFiles(t, srcFiles)

		destDir := t.TempDir()
		directories = append(directories, destDir)

		if err := run(cmd, directories, &buf); err != nil {
			t.Fatal(err)
		}

		for i, f := range srcFiles {
			expectedDestFile := filepath.Join(destDir, f.filename)

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

	t.Run("RecursiveSingleDirectory", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true, recursive: true}

		// Create  source files with known content
		srcFiles := []testFile{
			{path: "dir1", filename: "subdir/test1.txt", content: "test content"},
			{path: "dir1", filename: "subdir2/test2.txt", content: "test content"},
			{path: "dir1", filename: "text1.txt", content: "test content"},
		}

		dir, _ := setupTestDirWithFiles(t, srcFiles)

		srcDir := filepath.Join(dir, "dir1")
		destDir := t.TempDir()

		if err := run(cmd, []string{srcDir, destDir}, &buf); err != nil {
			t.Fatal(err)
		}

		// Verify each file was copied correctly
		for _, f := range srcFiles {
			expectedDestFile := filepath.Join(destDir, f.filename)
			srcFile := filepath.Join(srcDir, f.filename)

			// Check the copied file exists in the destination directory
			if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
				t.Fatalf("file was not copied to destination directory: %s", expectedDestFile)
			}

			// Verify content matches exactly
			copiedContent, err := os.ReadFile(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to read copied file: %v", err)
			}
			if string(copiedContent) != f.content {
				t.Errorf("content mismatch for %s: expected %q, got %q",
					f.filename, f.content, string(copiedContent))
			}

			// Check if file permissions are preserved
			srcInfo, err := os.Stat(srcFile)
			if err != nil {
				t.Fatalf("failed to stat source file %s: %v", srcFile, err)
			}

			destInfo, err := os.Stat(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to stat destination file %s: %v", expectedDestFile, err)
			}

			// Compare file permissions
			srcMode := srcInfo.Mode().Perm()
			destMode := destInfo.Mode().Perm()
			if srcMode != destMode {
				t.Errorf("permissions not preserved for %s: source %o, destination %o",
					f.filename, srcMode, destMode)
			}

			// Check if modification times are preserved
			srcModTime := srcInfo.ModTime()
			destModTime := destInfo.ModTime()
			if !srcModTime.Equal(destModTime) {
				t.Errorf("modification times not preserved for %s: source %v, destination %v",
					f.filename, srcModTime, destModTime)
			}

		}

	})
	t.Run("RecursiveMultipleDirectory", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true, recursive: true}

		// Create multiple source directories with nested structure
		srcFiles1 := []testFile{
			{path: "dir1", filename: "subdir/file1.txt", content: "content from dir1/subdir", mode: 0644},
			{path: "dir1", filename: "file2.txt", content: "root file in dir1", mode: 0755},
			{path: "dir1", filename: "nested/deep/file3.txt", content: "deeply nested file", mode: 0600},
		}

		srcFiles2 := []testFile{
			{path: "dir2", filename: "docs/readme.md", content: "documentation content", mode: 0644},
			{path: "dir2", filename: "scripts/build.sh", content: "#!/bin/bash\necho build", mode: 0755},
			{path: "dir2", filename: "config.json", content: "{\"version\": \"1.0\"}", mode: 0644},
		}

		// Setup first source directory
		baseDir1, _ := setupTestDirWithFiles(t, srcFiles1)
		srcDir1 := filepath.Join(baseDir1, "dir1")

		// Setup second source directory
		baseDir2, _ := setupTestDirWithFiles(t, srcFiles2)
		srcDir2 := filepath.Join(baseDir2, "dir2")

		// Create destination directory
		destDir := t.TempDir()

		// Copy multiple source directories to single destination
		// Format: [source1, source2, ..., destination]
		directories := []string{srcDir1, srcDir2, destDir}

		if err := run(cmd, directories, &buf); err != nil {
			t.Fatal(err)
		}

		// Verify files from first source directory
		for _, f := range srcFiles1 {
			expectedDestFile := filepath.Join(destDir, f.filename)
			srcFile := filepath.Join(srcDir1, f.filename)

			// Check file exists in destination
			if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
				t.Fatalf("file from dir1 was not copied: %s", expectedDestFile)
			}

			// Verify content integrity
			copiedContent, err := os.ReadFile(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to read copied file from dir1: %v", err)
			}
			if string(copiedContent) != f.content {
				t.Errorf("content mismatch for %s: expected %q, got %q",
					f.filename, f.content, string(copiedContent))
			}

			// Verify permissions preserved
			srcInfo, err := os.Stat(srcFile)
			if err != nil {
				t.Fatalf("failed to stat source file %s: %v", srcFile, err)
			}
			destInfo, err := os.Stat(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to stat dest file %s: %v", expectedDestFile, err)
			}

			if srcInfo.Mode().Perm() != destInfo.Mode().Perm() {
				t.Errorf("permissions not preserved for %s: source %o, dest %o",
					f.filename, srcInfo.Mode().Perm(), destInfo.Mode().Perm())
			}
		}

		// Verify files from second source directory
		for _, f := range srcFiles2 {
			expectedDestFile := filepath.Join(destDir, f.filename)
			srcFile := filepath.Join(srcDir2, f.filename)

			// Check file exists in destination
			if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
				t.Fatalf("file from dir2 was not copied: %s", expectedDestFile)
			}

			// Verify content integrity
			copiedContent, err := os.ReadFile(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to read copied file from dir2: %v", err)
			}
			if string(copiedContent) != f.content {
				t.Errorf("content mismatch for %s: expected %q, got %q",
					f.filename, f.content, string(copiedContent))
			}

			// Verify permissions preserved
			srcInfo, err := os.Stat(srcFile)
			if err != nil {
				t.Fatalf("failed to stat source file %s: %v", srcFile, err)
			}
			destInfo, err := os.Stat(expectedDestFile)
			if err != nil {
				t.Fatalf("failed to stat dest file %s: %v", expectedDestFile, err)
			}

			if srcInfo.Mode().Perm() != destInfo.Mode().Perm() {
				t.Errorf("permissions not preserved for %s: source %o, dest %o",
					f.filename, srcInfo.Mode().Perm(), destInfo.Mode().Perm())
			}
		}

		// Verify directory structure is correctly merged
		// All files should coexist in destination without conflicts
		expectedFileCount := len(srcFiles1) + len(srcFiles2)
		actualFiles := 0

		err := filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				actualFiles++
			}
			return nil
		})

		if err != nil {
			t.Fatalf("failed to walk destination directory: %v", err)
		}

		if actualFiles != expectedFileCount {
			t.Errorf("expected %d files in destination, found %d", expectedFileCount, actualFiles)
		}
	})

}

func TestCopyFileErrors(t *testing.T) {

	t.Run("NonExistentSourceFile", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		destDir := t.TempDir()
		nonExistentFile := "/path/that/does/not/exist.txt"

		// Attempt to copy non-existent file
		err := run(cmd, []string{nonExistentFile, destDir}, &buf)
		if err == nil {
			t.Error("expected error when copying non-existent file, got nil")
		}

		// Should contain helpful error message about source not existing
		if !strings.Contains(err.Error(), "no such file or directory") &&
			!strings.Contains(err.Error(), "cannot find") {
			t.Errorf("expected 'no such file or directory' or 'cannot find' in error, got: %v", err)
		}
	})

	t.Run("NonExistentSourceDirectory", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true, recursive: true}

		destDir := t.TempDir()
		nonExistentDir := "/path/that/does/not/exist/"

		// Attempt to copy non-existent directory recursively
		err := run(cmd, []string{nonExistentDir, destDir}, &buf)
		if err == nil {
			t.Error("expected error when copying non-existent directory, got nil")
		}

		if !strings.Contains(err.Error(), "no such file or directory") &&
			!strings.Contains(err.Error(), "cannot find") {
			t.Errorf("expected path error in message, got: %v", err)
		}
	})

	t.Run("InvalidDestinationPath", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		// Create valid source file
		srcFiles := []testFile{{path: ".", filename: "test.txt", content: "test", mode: 0644}}
		_, files := setupTestDirWithFiles(t, srcFiles)
		srcFile := files[0]

		// Use invalid destination (file that cannot be created)
		invalidDest := "/dev/null/cannot/create/this/path"

		if err := run(cmd, []string{srcFile, invalidDest}, &buf); err == nil {
			t.Fatalf("Expected error")
		}

	})

	t.Run("SourceIsDirectory_NonRecursive", func(t *testing.T) {
		// Save original logger and restore after test
		oldLogger := errorLogger
		defer func() { errorLogger = oldLogger }()

		// Capture error messages
		var errBuf bytes.Buffer
		errorLogger = log.New(&errBuf, "fmn: ", 0)

		var buf bytes.Buffer
		cmd := command{copy: true} // Non-recursive

		// Create source directory with files
		srcFiles := []testFile{{path: "testdir", filename: "file.txt", content: "content", mode: 0644}}
		baseDir, _ := setupTestDirWithFiles(t, srcFiles)
		srcDir := filepath.Join(baseDir, "testdir")

		destDir := t.TempDir()

		// Should fail because trying to copy directory without -r flag
		err := run(cmd, []string{srcDir, destDir}, &buf)
		if err == nil {
			t.Error("expected error when copying directory without recursive flag, got nil")
		}

		// Error should indicate some files could not be copied
		if !strings.Contains(err.Error(), "some files could not be copied") {
			t.Errorf("expected 'some files could not be copied' error, got: %v", err)
		}

		// Check that the specific omission message was logged
		errOutput := errBuf.String()
		if !strings.Contains(errOutput, "omitting directory") {
			t.Errorf("expected 'omitting directory' in error log, got: %q", errOutput)
		}
		if !strings.Contains(errOutput, "use -r for recursive") {
			t.Errorf("expected 'use -r for recursive' in error log, got: %q", errOutput)
		}
		if !strings.Contains(errOutput, srcDir) {
			t.Errorf("expected source directory path in error log, got: %q", errOutput)
		}
	})

	t.Run("InsufficientPermissions", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		// Create source file
		srcFiles := []testFile{{path: ".", filename: "protected.txt", content: "secret content", mode: 0644}}
		_, files := setupTestDirWithFiles(t, srcFiles)
		srcFile := files[0]

		// Create destination directory with no write permissions
		destDir := t.TempDir()
		protectedDir := filepath.Join(destDir, "protected")
		if err := os.MkdirAll(protectedDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Remove write permission from destination directory
		if err := os.Chmod(protectedDir, 0555); err != nil {
			t.Fatal(err)
		}

		// Restore permissions after test
		defer os.Chmod(protectedDir, 0755)

		// Try to copy file into protected directory
		destFile := filepath.Join(protectedDir, "protected.txt")
		err := copyFile(cmd, []string{srcFile, destFile}, &buf)
		if err == nil {
			t.Error("expected error when copying to protected directory, got nil")
		}

		// Should contain permission-related error
		errStr := strings.ToLower(err.Error())
		if !strings.Contains(errStr, "permission") && !strings.Contains(errStr, "denied") {
			t.Errorf("expected permission error, got: %v", err)
		}
	})

	t.Run("TooFewArguments", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		// Test with only one argument (need at least 2: source + dest)
		err := run(cmd, []string{"onlyfile"}, &buf)
		if err == nil {
			t.Error("expected error with too few arguments, got nil")
		}

		// Test with empty arguments
		err = run(cmd, []string{}, &buf)
		if err == nil {
			t.Error("expected error with no arguments, got nil")
		}

		if !strings.Contains(err.Error(), "at least one source path") {
			t.Errorf("expected %s to contain %s", err.Error(), "at least one source path")
		}
	})

	t.Run("CopyFileToItself", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		// Create source file
		srcFiles := []testFile{{path: ".", filename: "selfcopy.txt", content: "content", mode: 0644}}
		_, files := setupTestDirWithFiles(t, srcFiles)
		srcFile := files[0]

		// Try to copy file to itself
		err := run(cmd, []string{srcFile, srcFile}, &buf)
		if err == nil {
			t.Error("expected error when copying file to itself, got nil")
		}

		// Should detect self-copy attempt
		errStr := strings.ToLower(err.Error())
		if !strings.Contains(errStr, "cannot copy a path to itself") {
			t.Errorf("expected self-copy error, got: %v", err)
		}
	})

	t.Run("MultipleFilesToNonDirectoryDestination", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := command{copy: true}

		// Create multiple source files
		srcFiles := []testFile{
			{path: ".", filename: "file1.txt", content: "content1", mode: 0644},
			{path: ".", filename: "file2.txt", content: "content2", mode: 0644},
		}
		_, files := setupTestDirWithFiles(t, srcFiles)

		// Create a regular file as destination (not a directory)
		destFiles := []testFile{{path: ".", filename: "dest.txt", content: "existing", mode: 0644}}
		_, destFile := setupTestDirWithFiles(t, destFiles)

		// Try to copy multiple files to a non-directory destination
		directories := []string{files[0], files[1], destFile[0]}
		err := run(cmd, directories, &buf)
		if err == nil {
			t.Error("expected error when copying multiple files to non-directory destination, got nil")
		}

		// Should contain error message about target not being a directory
		if !strings.Contains(err.Error(), "Not a directory") {
			t.Errorf("expected 'Not a directory' error, got: %v", err)
		}
	})

}

type testFile struct {
	path     string
	filename string
	content  string
	mode     os.FileMode
}

func setupTestDirWithFiles(t *testing.T, files []testFile) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	paths := []string{}

	for _, f := range files {
		if f.filename != "" && f.content != "" {
			// Creating a file
			fullPath := filepath.Join(dir, f.path, f.filename)
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
			paths = append(paths, fullPath)
		} else if f.path != "" {
			// Creating directory only
			dirPath := filepath.Join(dir, f.path)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dirPath, err)
			}
		}
	}
	return dir, paths
}
