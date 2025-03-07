package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsGitRoot tests the isGitRoot function.
func TestIsGitRoot(t *testing.T) {
	// Create a temporary directory for testing.
	tempDir, err := os.MkdirTemp("", "local-gitingest-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test.

	// Test cases:
	tests := []struct {
		name     string
		setup    func(dir string) error // Function to set up the test environment
		expected bool                   // Expected result
	}{
		{
			name: "Not a Git repo",
			setup: func(dir string) error {
				return nil // No setup needed, just an empty directory.
			},
			expected: false,
		},
		{
			name: "Git repo (using .git directory)",
			setup: func(dir string) error {
				return os.Mkdir(filepath.Join(dir, ".git"), 0755)
			},
			expected: true,
		},
		{
			name: "Git repo (using git command, in root)",
			setup: func(dir string) error {
				// Initialize a Git repository in the temp directory.
				cmd := exec.Command("git", "init")
				cmd.Dir = dir
				return cmd.Run()
			},
			expected: true,
		},
		{
			name: "Git repo (using git command, in subdirectory)",
			setup: func(dir string) error {
				// Initialize a Git repository.
				cmd := exec.Command("git", "init")
				cmd.Dir = dir
				if err := cmd.Run(); err != nil {
					return err
				}
				// Create a subdirectory.
				return os.Mkdir(filepath.Join(dir, "subdir"), 0755)
			},
			expected: true, // Even in a subdirectory, it should return true.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change to the temporary directory for the test.
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current dir: %v", err)
			}
			testDir := filepath.Join(tempDir, tt.name)
			os.MkdirAll(testDir, 0755)  // Create test directory
			os.Chdir(testDir)           // Change to test directory.
			defer os.Chdir(originalDir) // Restore original directory after the test.

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			actual := isGitRoot()
			if actual != tt.expected {
				t.Errorf("isGitRoot() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

// TestBuildDirectoryStructure tests the buildDirectoryStructure function.
func TestBuildDirectoryStructure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "local-gitingest-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	structure := map[string]string{
		"file1.txt":       "Content of file1",
		"file2.go":        "package main\nfunc main() {}",
		"subdir/file3.md": "# Markdown Header",
		"subdir/file4.py": "print('Hello')",
		"subdir/":         "",
		".hiddenfile":     "Hidden file content",
		".hidden_dir/":    "",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		if strings.HasSuffix(path, "/") {
			os.MkdirAll(fullPath, 0755)
		} else {
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			err := os.WriteFile(fullPath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}
	}

	tests := []struct {
		name             string
		excludeList      map[string]bool
		includeSizeLimit bool
		sizeLimit        int64
		expectedFiles    []string // Expected file names (relative paths)
		setup            func()
		expectError      bool
	}{
		{
			name:          "No exclusions, no size limit",
			excludeList:   map[string]bool{},
			expectedFiles: []string{"file1.txt", "file2.go", ".hiddenfile", "subdir/file3.md", "subdir/file4.py"},
		},
		{
			name:          "Exclude .go and .md files",
			excludeList:   map[string]bool{".go": true, ".md": true},
			expectedFiles: []string{"file1.txt", ".hiddenfile", "subdir/file4.py"},
		},
		{
			name:             "Size limit of 20 bytes",
			excludeList:      map[string]bool{},
			includeSizeLimit: true,
			sizeLimit:        20,
			expectedFiles:    []string{"subdir/file4.py", ".hiddenfile", "file1.txt", "subdir/file3.md"}, // Corrected expected files
		},
		{
			name: "Error during WalkDir",
			setup: func() {
				err := os.WriteFile(filepath.Join(tempDir, "unreadable.txt"), []byte("unreadable"), 0222)
				if err != nil {
					t.Fatalf("Failed to create unreadable file: %v", err)
				}
			},
			excludeList: map[string]bool{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			_, fileContents, err := buildDirectoryStructure(tempDir, tt.excludeList, tt.includeSizeLimit, tt.sizeLimit)

			if tt.expectError {
				if err == nil {
					t.Error("Expected an error, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildDirectoryStructure() returned error: %v", err)
			}

			// Check if expected files exist and have non-empty content
			for _, expectedFile := range tt.expectedFiles {
				content, ok := fileContents[expectedFile]
				if !ok {
					t.Errorf("Expected file not found: %s", expectedFile)
				} else if len(content) == 0 {
					t.Errorf("File content is empty for: %s", expectedFile)
				}
			}

			// 检查实际存在的文件是否 *没有超出* 预期文件列表
			for actualFile := range fileContents {
				found := false
				for _, expectedFile := range tt.expectedFiles {
					if actualFile == expectedFile {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected file found: %s", actualFile)
				}
			}
		})
	}
}
