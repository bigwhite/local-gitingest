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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			_, fileContents, err := buildDirectoryStructure(tempDir, tt.excludeList, tt.includeSizeLimit, tt.sizeLimit, nil)

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

// TestParsePattern tests the ParsePattern function using table-driven testing.
func TestParsePattern(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *Pattern
		wantErr bool
	}{
		{
			name: "simple wildcard",
			line: "*.log",
			want: &Pattern{
				raw:      "*.log",
				anchored: false,
				isDir:    false,
				wildcard: true,
			},
		},
		{
			name: "anchored pattern",
			line: "/temp/",
			want: &Pattern{
				raw:      "/temp/",
				anchored: true,
				isDir:    true,
				wildcard: false,
			},
		},
		{
			name: "directory-only pattern",
			line: "build/",
			want: &Pattern{
				raw:      "build/",
				anchored: false,
				isDir:    true,
				wildcard: false,
			},
		},
		{
			name: "anchored wildcard",
			line: "/*.tmp",
			want: &Pattern{
				raw:      "/*.tmp",
				anchored: true,
				isDir:    false,
				wildcard: true,
			},
		},
		{
			name: "character range",
			line: "*.tmp",
			want: &Pattern{
				raw:      "*.tmp",
				anchored: false,
				isDir:    false,
				wildcard: true,
			},
		},
		{
			name: "comment line",
			line: "# This is a comment",
			want: nil,
		},
		{
			name: "empty line",
			line: "",
			want: nil,
		},
		{
			name: "whitespace only",
			line: "   ",
			want: nil,
		},
		{
			name: "comment with whitespace",
			line: "  # comment with leading space",
			want: nil,
		},
		{
			name: "pattern with question mark",
			line: "file?.txt",
			want: &Pattern{
				raw:      "file?.txt",
				anchored: false,
				isDir:    false,
				wildcard: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePattern(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For nil returns (comments, empty lines)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParsePattern() = %v, want nil", got)
				}
				return
			}

			// For non-nil returns, check all fields
			if got == nil {
				t.Errorf("ParsePattern() = nil, want %v", tt.want)
				return
			}

			if got.raw != tt.want.raw {
				t.Errorf("ParsePattern().raw = %q, want %q", got.raw, tt.want.raw)
			}
			if got.anchored != tt.want.anchored {
				t.Errorf("ParsePattern().anchored = %v, want %v", got.anchored, tt.want.anchored)
			}
			if got.isDir != tt.want.isDir {
				t.Errorf("ParsePattern().isDir = %v, want %v", got.isDir, tt.want.isDir)
			}
			if got.wildcard != tt.want.wildcard {
				t.Errorf("ParsePattern().wildcard = %v, want %v", got.wildcard, tt.want.wildcard)
			}
		})
	}
}

// TestLoadGitignore tests the LoadGitignore function.
func TestLoadGitignore(t *testing.T) {
	tests := []struct {
		name           string
		gitignoreContent string
		wantPatternCount int
		wantWarnCount    int
		wantErr         bool
	}{
		{
			name:             "missing .gitignore",
			gitignoreContent: "",
			wantPatternCount: 0,
			wantWarnCount:    0,
			wantErr:          false,
		},
		{
			name:             "valid patterns",
			gitignoreContent: "*.log\n*.tmp\nbuild/\n",
			wantPatternCount: 3,
			wantWarnCount:    0,
			wantErr:          false,
		},
		{
			name:             "comments and empty lines",
			gitignoreContent: "# Comment\n\n*.log\n\n  # Another comment\n*.tmp\n",
			wantPatternCount: 2,
			wantWarnCount:    0,
			wantErr:          false,
		},
		{
			name:             "mixed valid and comment lines",
			gitignoreContent: "# Build artifacts\n*.o\n*.exe\n\n# Logs\n*.log\n",
			wantPatternCount: 3,
			wantWarnCount:    0,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir, err := os.MkdirTemp("", "gitignore-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .gitignore if content is provided
			if tt.gitignoreContent != "" {
				gitignorePath := filepath.Join(tempDir, ".gitignore")
				err := os.WriteFile(gitignorePath, []byte(tt.gitignoreContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create .gitignore: %v", err)
				}
			}

			// Load gitignore
			matcher, err := LoadGitignore(tempDir, false)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadGitignore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if matcher == nil {
				t.Fatal("LoadGitignore() returned nil matcher")
			}

			// Check pattern count
			if len(matcher.patterns) != tt.wantPatternCount {
				t.Errorf("LoadGitignore() pattern count = %d, want %d", len(matcher.patterns), tt.wantPatternCount)
			}

			// Check warning count (we're not testing stderr output here, just the counter)
			if matcher.warnCount != tt.wantWarnCount {
				t.Errorf("LoadGitignore() warnCount = %d, want %d", matcher.warnCount, tt.wantWarnCount)
			}
		})
	}
}

// TestMatchesPattern tests the matchesPattern method.
func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  *Pattern
		relPath  string
		isDir    bool
		wantMatch bool
	}{
		{
			name: "anchored pattern matches root file",
			pattern: &Pattern{raw: "/*.log", anchored: true, isDir: false, wildcard: true},
			relPath:  "debug.log",
			isDir:    false,
			wantMatch: true,
		},
		{
			name: "anchored pattern does not match nested file",
			pattern: &Pattern{raw: "/*.log", anchored: true, isDir: false, wildcard: true},
			relPath:  "logs/debug.log",
			isDir:    false,
			wantMatch: false,
		},
		{
			name: "non-anchored wildcard matches any level",
			pattern: &Pattern{raw: "*.log", anchored: false, isDir: false, wildcard: true},
			relPath:  "debug.log",
			isDir:    false,
			wantMatch: true,
		},
		{
			name: "non-anchored wildcard matches nested file",
			pattern: &Pattern{raw: "*.log", anchored: false, isDir: false, wildcard: true},
			relPath:  "logs/debug.log",
			isDir:    false,
			wantMatch: true,
		},
		{
			name: "directory-only pattern matches directory",
			pattern: &Pattern{raw: "build/", anchored: false, isDir: true, wildcard: false},
			relPath:  "build",
			isDir:    true,
			wantMatch: true,
		},
		{
			name: "directory-only pattern does not match file",
			pattern: &Pattern{raw: "build/", anchored: false, isDir: true, wildcard: false},
			relPath:  "build",
			isDir:    false,
			wantMatch: false,
		},
		{
			name: "anchored directory pattern",
			pattern: &Pattern{raw: "/temp/", anchored: true, isDir: true, wildcard: false},
			relPath:  "temp",
			isDir:    true,
			wantMatch: true,
		},
		{
			name: "anchored directory pattern does not match nested",
			pattern: &Pattern{raw: "/temp/", anchored: true, isDir: true, wildcard: false},
			relPath:  "subdir/temp",
			isDir:    true,
			wantMatch: false,
		},
		{
			name: "non-anchored directory matches nested",
			pattern: &Pattern{raw: "node_modules/", anchored: false, isDir: true, wildcard: false},
			relPath:  "frontend/node_modules",
			isDir:    true,
			wantMatch: true,
		},
		{
			name: "exact filename match",
			pattern: &Pattern{raw: "config.json", anchored: false, isDir: false, wildcard: false},
			relPath:  "config.json",
			isDir:    false,
			wantMatch: true,
		},
		{
			name: "exact filename match in subdirectory",
			pattern: &Pattern{raw: "config.json", anchored: false, isDir: false, wildcard: false},
			relPath:  "app/config.json",
			isDir:    false,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GitignoreMatcher{patterns: []*Pattern{tt.pattern}, verbose: false}
			got := m.matchesPattern(tt.pattern, tt.relPath, tt.isDir)
			if got != tt.wantMatch {
				t.Errorf("matchesPattern() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

// TestIsIgnored tests the IsIgnored method.
func TestIsIgnored(t *testing.T) {
	tests := []struct {
		name         string
		gitignoreContent string
		relPath         string
		isDir           bool
		wantIgnored     bool
	}{
		{
			name:             "file matches wildcard pattern",
			gitignoreContent: "*.log\n",
			relPath:          "debug.log",
			isDir:            false,
			wantIgnored:      true,
		},
		{
			name:             "file does not match any pattern",
			gitignoreContent: "*.log\n*.tmp\n",
			relPath:          "main.go",
			isDir:            false,
			wantIgnored:      false,
		},
		{
			name:             "directory matches pattern",
			gitignoreContent: "build/\ndist/\n",
			relPath:          "build",
			isDir:            true,
			wantIgnored:      true,
		},
		{
			name:             "empty gitignore ignores nothing",
			gitignoreContent: "",
			relPath:          "file.txt",
			isDir:            false,
			wantIgnored:      false,
		},
		{
			name:             "multiple patterns, file matches second",
			gitignoreContent: "*.tmp\n*.log\n*.o\n",
			relPath:          "debug.log",
			isDir:            false,
			wantIgnored:      true,
		},
		{
			name:             "anchored pattern matches root file",
			gitignoreContent: "/*.log\n",
			relPath:          "debug.log",
			isDir:            false,
			wantIgnored:      true,
		},
		{
			name:             "anchored pattern does not match nested file",
			gitignoreContent: "/*.log\n",
			relPath:          "logs/debug.log",
			isDir:            false,
			wantIgnored:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir, err := os.MkdirTemp("", "gitignore-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .gitignore if content is provided
			if tt.gitignoreContent != "" {
				gitignorePath := filepath.Join(tempDir, ".gitignore")
				err := os.WriteFile(gitignorePath, []byte(tt.gitignoreContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create .gitignore: %v", err)
				}
			}

			// Load gitignore
			matcher, err := LoadGitignore(tempDir, false)
			if err != nil {
				t.Fatalf("LoadGitignore() failed: %v", err)
			}

			// Test IsIgnored
			got := matcher.IsIgnored(tt.relPath, tt.isDir)
			if got != tt.wantIgnored {
				t.Errorf("IsIgnored() = %v, want %v", got, tt.wantIgnored)
			}
		})
	}
}
