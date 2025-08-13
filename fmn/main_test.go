package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestList is a table-driven test for the list functionality.
func TestList(t *testing.T) {
	var testFile1, testDir1, testDir2 string

	testCases := []struct {
		name                  string
		setup                 func(t *testing.T) []string
		cmd                   command
		wantErr               bool
		wantErrContains       string
		wantOutputContains    []string
		wantOutputNotContains []string
		wantErrLogContains    string
	}{
		{
			name: "List single directory with files",
			setup: func(t *testing.T) []string {
				testDir1, _ = setupTestDirWithFiles(t, []testFile{
					{filename: "file1.txt"},
					{filename: "file2.txt"},
				})
				return []string{testDir1}
			},
			wantOutputContains: []string{
				fmt.Sprintf("%s:", testDir1),
				"file1.txt",
				"file2.txt",
			},
		},
		{
			name: "List single file",
			setup: func(t *testing.T) []string {
				_, files := setupTestDirWithFiles(t, []testFile{{filename: "file1.txt"}})
				testFile1 = files[0]
				return []string{testFile1}
			},
			wantOutputContains:    []string{testFile1},
			wantOutputNotContains: []string{":"},
		},
		{
			name: "List mixed files and directories",
			setup: func(t *testing.T) []string {
				testDir1, _ = setupTestDirWithFiles(t, []testFile{{filename: "dir1file.txt"}})
				testDir2, _ = setupTestDirWithFiles(t, []testFile{{filename: "dir2file.txt"}})
				_, files := setupTestDirWithFiles(t, []testFile{{filename: "standalone.txt"}})
				testFile1 = files[0]
				return []string{testDir1, testFile1, testDir2}
			},
			wantOutputContains: []string{
				fmt.Sprintf("%s:", testDir1),
				"dir1file.txt",
				testFile1,
				fmt.Sprintf("%s:", testDir2),
				"dir2file.txt",
			},
		},
		{
			name: "Error on non-existent file",
			setup: func(t *testing.T) []string {
				return []string{"nonexistent.txt"}
			},
			wantErr:         true,
			wantErrContains: "cannot stat 'nonexistent.txt'",
		},
		{
			name: "Error on directory with no read permission",
			setup: func(t *testing.T) []string {
				testDir1, _ = setupTestDirWithFiles(t, []testFile{{filename: "file.txt"}})
				// Change permissions to be non-readable
				if err := os.Chmod(testDir1, 0300); err != nil {
					t.Fatalf("Failed to change permissions: %v", err)
				}
				t.Cleanup(func() {
					os.Chmod(testDir1, 0755)
				})
				return []string{testDir1}
			},
			wantErr:            true,
			wantErrContains:    "some directories could not be read",
			wantOutputContains: []string{fmt.Sprintf("%s:", testDir1)}, // Still prints the header
			wantErrLogContains: "permission denied",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Setup ---
			oldConsole := console
			oldLogger := errorLogger
			defer func() {
				console = oldConsole
				errorLogger = oldLogger
			}()

			var outBuf, errBuf bytes.Buffer
			console.Out = &outBuf
			console.Err = &errBuf
			errorLogger = log.New(&errBuf, "fmn: ", 0)

			args := tc.setup(t)

			// --- Execute ---
			err := run(tc.cmd, args)

			// --- Assert ---
			output := outBuf.String()
			errOutput := errBuf.String()

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				} else if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("expected error to contain %q, got %q", tc.wantErrContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("did not expect an error, but got: %v", err)
			}

			for _, want := range tc.wantOutputContains {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, but it did not. Got:\n%s", want, output)
				}
			}

			for _, notWant := range tc.wantOutputNotContains {
				if strings.Contains(output, notWant) {
					t.Errorf("expected output to NOT contain %q, but it did. Got:\n%s", notWant, output)
				}
			}

			if tc.wantErrLogContains != "" && !strings.Contains(errOutput, tc.wantErrLogContains) {
				t.Errorf("expected error log to contain %q, but it did not. Got:\n%s", tc.wantErrLogContains, errOutput)
			}
		})
	}
}

// TestCopy is a table-driven test for the copy functionality, covering various
// scenarios including force and interactive modes.
func TestCopy(t *testing.T) {
	testCases := []struct {
		name            string
		cmd             command
		setup           func(t *testing.T) (srcPaths []string, destPath string)
		userInput       string // For interactive tests
		wantErr         bool
		wantErrContains string
		wantContent     map[string]string // map[filepath]content
		wantNoContent   []string          // list of filepaths that should NOT exist
	}{
		// --- Success Cases ---
		{
			name: "Copy single file to directory",
			cmd:  command{copy: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				_, srcFiles := setupTestDirWithFiles(t, []testFile{
					{filename: "file1.txt", content: "test content"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{})
				return srcFiles, destDir
			},
			wantErr: false,
			wantContent: map[string]string{
				"file1.txt": "test content",
			},
		},
		{
			name: "Copy multiple files to directory",
			cmd:  command{copy: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				_, srcFiles := setupTestDirWithFiles(t, []testFile{
					{filename: "file1.txt", content: "content1"},
					{filename: "file2.txt", content: "content2"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{})
				return srcFiles, destDir
			},
			wantErr: false,
			wantContent: map[string]string{
				"file1.txt": "content1",
				"file2.txt": "content2",
			},
		},
		{
			name: "Recursive copy",
			cmd:  command{copy: true, recursive: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				srcDir, _ := setupTestDirWithFiles(t, []testFile{
					{path: "src", filename: "root.txt", content: "root"},
					{path: "src/subdir", filename: "sub.txt", content: "sub"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{})
				return []string{filepath.Join(srcDir, "src")}, destDir
			},
			wantErr: false,
			wantContent: map[string]string{
				"root.txt":       "root",
				"subdir/sub.txt": "sub",
			},
		},
		// --- Overwrite Logic ---
		{
			name: "Overwrite with force",
			cmd:  command{copy: true, force: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				_, srcFiles := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "new content"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "old content"},
				})
				return srcFiles, destDir
			},
			wantErr: false,
			wantContent: map[string]string{
				"file.txt": "new content",
			},
		},
		{
			name: "Overwrite interactive - yes",
			cmd:  command{copy: true, interactive: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				_, srcFiles := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "new content"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "old content"},
				})
				return srcFiles, destDir
			},
			userInput: "y",
			wantErr:   false,
			wantContent: map[string]string{
				"file.txt": "new content",
			},
		},
		{
			name: "Overwrite interactive - no",
			cmd:  command{copy: true, interactive: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				_, srcFiles := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "new content"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "old content"},
				})
				return srcFiles, destDir
			},
			userInput: "n",
			wantErr:   false,
			wantContent: map[string]string{
				"file.txt": "old content",
			},
		},
		// --- Error Cases ---
		{
			name: "Fail on overwrite by default",
			cmd:  command{copy: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				_, srcFiles := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "new content"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{
					{filename: "file.txt", content: "old content"},
				})
				return srcFiles, destDir
			},
			wantErr:         true,
			wantErrContains: "already exists",
			wantContent: map[string]string{
				"file.txt": "old content",
			},
		},
		{
			name: "Source does not exist",
			cmd:  command{copy: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				destDir, _ := setupTestDirWithFiles(t, []testFile{})
				return []string{"nonexistent.txt"}, destDir
			},
			wantErr:         true,
			wantErrContains: "cannot stat source",
		},
		{
			name: "Copy directory without recursive flag",
			cmd:  command{copy: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				srcDir, _ := setupTestDirWithFiles(t, []testFile{
					{path: "src", filename: "file.txt", content: "content"},
				})
				destDir, _ := setupTestDirWithFiles(t, []testFile{})
				return []string{filepath.Join(srcDir, "src")}, destDir
			},
			wantErr:         true,
			wantErrContains: "omitting directory",
		},
		{
			name: "Multiple files to non-directory destination",
			cmd:  command{copy: true},
			setup: func(t *testing.T) (srcPaths []string, destPath string) {
				_, srcFiles := setupTestDirWithFiles(t, []testFile{
					{filename: "file1.txt", content: "content1"},
					{filename: "file2.txt", content: "content2"},
				})
				_, destFile := setupTestDirWithFiles(t, []testFile{
					{filename: "dest.txt", content: "existing"},
				})
				return srcFiles, destFile[0]
			},
			wantErr:         true,
			wantErrContains: "is not a directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Setup ---
			oldConsole := console
			defer func() { console = oldConsole }()

			var inBuf, outBuf bytes.Buffer
			console.Out = &outBuf
			console.Err = &outBuf

			if tc.userInput != "" {
				inBuf.WriteString(tc.userInput + "\n")
				console.In = &inBuf
			}

			srcPaths, destPath := tc.setup(t)
			args := append(srcPaths, destPath)

			// --- Execute ---
			err := run(tc.cmd, args)

			// --- Assert ---
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				} else if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("expected error to contain %q, got %q", tc.wantErrContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("did not expect an error, but got: %v", err)
			}

			// Verify content of files that should exist
			for file, expectedContent := range tc.wantContent {
				fullPath := filepath.Join(destPath, file)
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("could not read expected file '%s': %v", fullPath, err)
					continue
				}
				if string(content) != expectedContent {
					t.Errorf("content mismatch for '%s': got %q, want %q", file, string(content), expectedContent)
				}
			}

			// Verify files that should NOT exist
			for _, file := range tc.wantNoContent {
				fullPath := filepath.Join(destPath, file)
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("file '%s' should not exist, but it does", fullPath)
				}
			}
		})
	}
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
		// Set default content if empty, so we can create an empty file vs a directory
		if f.filename != "" && f.content == "" {
			f.content = ""
		}

		if f.filename != "" {
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
			dirPath := filepath.Join(dir, f.path)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dirPath, err)
			}
		}
	}
	return dir, paths
}
