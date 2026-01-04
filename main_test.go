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

			_, fileContents, err := buildDirectoryStructure(tempDir, "", []string{}, tt.excludeList, tt.includeSizeLimit, tt.sizeLimit, nil)

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
		name             string
		gitignoreContent string
		wantPatternCount int
		wantWarnCount    int
		wantErr          bool
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
		name      string
		pattern   *Pattern
		relPath   string
		isDir     bool
		wantMatch bool
	}{
		{
			name:      "anchored pattern matches root file",
			pattern:   &Pattern{raw: "/*.log", anchored: true, isDir: false, wildcard: true},
			relPath:   "debug.log",
			isDir:     false,
			wantMatch: true,
		},
		{
			name:      "anchored pattern does not match nested file",
			pattern:   &Pattern{raw: "/*.log", anchored: true, isDir: false, wildcard: true},
			relPath:   "logs/debug.log",
			isDir:     false,
			wantMatch: false,
		},
		{
			name:      "non-anchored wildcard matches any level",
			pattern:   &Pattern{raw: "*.log", anchored: false, isDir: false, wildcard: true},
			relPath:   "debug.log",
			isDir:     false,
			wantMatch: true,
		},
		{
			name:      "non-anchored wildcard matches nested file",
			pattern:   &Pattern{raw: "*.log", anchored: false, isDir: false, wildcard: true},
			relPath:   "logs/debug.log",
			isDir:     false,
			wantMatch: true,
		},
		{
			name:      "directory-only pattern matches directory",
			pattern:   &Pattern{raw: "build/", anchored: false, isDir: true, wildcard: false},
			relPath:   "build",
			isDir:     true,
			wantMatch: true,
		},
		{
			name:      "directory-only pattern does not match file",
			pattern:   &Pattern{raw: "build/", anchored: false, isDir: true, wildcard: false},
			relPath:   "build",
			isDir:     false,
			wantMatch: false,
		},
		{
			name:      "anchored directory pattern",
			pattern:   &Pattern{raw: "/temp/", anchored: true, isDir: true, wildcard: false},
			relPath:   "temp",
			isDir:     true,
			wantMatch: true,
		},
		{
			name:      "anchored directory pattern does not match nested",
			pattern:   &Pattern{raw: "/temp/", anchored: true, isDir: true, wildcard: false},
			relPath:   "subdir/temp",
			isDir:     true,
			wantMatch: false,
		},
		{
			name:      "non-anchored directory matches nested",
			pattern:   &Pattern{raw: "node_modules/", anchored: false, isDir: true, wildcard: false},
			relPath:   "frontend/node_modules",
			isDir:     true,
			wantMatch: true,
		},
		{
			name:      "exact filename match",
			pattern:   &Pattern{raw: "config.json", anchored: false, isDir: false, wildcard: false},
			relPath:   "config.json",
			isDir:     false,
			wantMatch: true,
		},
		{
			name:      "exact filename match in subdirectory",
			pattern:   &Pattern{raw: "config.json", anchored: false, isDir: false, wildcard: false},
			relPath:   "app/config.json",
			isDir:     false,
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
		name             string
		gitignoreContent string
		relPath          string
		isDir            bool
		wantIgnored      bool
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

// TestValidateTargetDir tests the validateTargetDir function using table-driven testing.
func TestValidateTargetDir(t *testing.T) {
	tests := []struct {
		name           string
		rootDir        string
		targetDir      string
		setup          func(string) error
		wantErr        bool
		errMsgContains string
	}{
		{
			name:      "empty target dir is valid",
			rootDir:   "/tmp/test",
			targetDir: "",
			wantErr:   false,
		},
		{
			name:           "nonexistent directory",
			rootDir:        "/tmp/test",
			targetDir:      "nonexistent",
			wantErr:        true,
			errMsgContains: "not found",
		},
		{
			name:      "valid subdirectory",
			rootDir:   "/tmp/test",
			targetDir: "cmd",
			setup: func(root string) error {
				return os.Mkdir(filepath.Join(root, "cmd"), 0755)
			},
			wantErr: false,
		},
		{
			name:      "target is a file not directory",
			rootDir:   "/tmp/test",
			targetDir: "README.md",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "README.md"), []byte("# Test"), 0644)
			},
			wantErr:        true,
			errMsgContains: "not a directory",
		},
		{
			name:      "hidden directory is valid",
			rootDir:   "/tmp/test",
			targetDir: ".github",
			setup: func(root string) error {
				return os.Mkdir(filepath.Join(root, ".github"), 0755)
			},
			wantErr: false,
		},
		{
			name:           "path traversal attack",
			rootDir:        "/tmp/test",
			targetDir:      "../../etc",
			wantErr:        true,
			errMsgContains: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "validate-target-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Update rootDir to use temp directory
			rootDir := tt.rootDir
			if rootDir == "/tmp/test" {
				rootDir = tempDir
			}

			// Setup test environment
			if tt.setup != nil {
				if err := tt.setup(rootDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Run function under test
			err = validateTargetDir(rootDir, tt.targetDir)

			// Check results
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTargetDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsgContains != "" {
				if !strings.Contains(err.Error(), tt.errMsgContains) {
					t.Errorf("validateTargetDir() error = %v, want error containing %q", err, tt.errMsgContains)
				}
			}
		})
	}
}

// TestCheckTargetExcludeConflict tests the checkTargetExcludeConflict function using table-driven testing.
func TestCheckTargetExcludeConflict(t *testing.T) {
	tests := []struct {
		name           string
		targetDir      string
		excludeDirs    []string
		wantErr        bool
		errMsgContains string
	}{
		{
			name:        "no conflict - empty target",
			targetDir:   "",
			excludeDirs: []string{"vendor", "node_modules"},
			wantErr:     false,
		},
		{
			name:        "no conflict - different names",
			targetDir:   "cmd",
			excludeDirs: []string{"vendor", "node_modules"},
			wantErr:     false,
		},
		{
			name:           "conflict - exact match",
			targetDir:      "vendor",
			excludeDirs:    []string{"vendor", "node_modules"},
			wantErr:        true,
			errMsgContains: "conflicts",
		},
		{
			name:        "no conflict - case sensitive",
			targetDir:   "Vendor",
			excludeDirs: []string{"vendor"},
			wantErr:     false,
		},
		{
			name:           "conflict - first in list",
			targetDir:      "node_modules",
			excludeDirs:    []string{"node_modules", "vendor"},
			wantErr:        true,
			errMsgContains: "conflicts",
		},
		{
			name:        "no conflict - empty exclude list",
			targetDir:   "cmd",
			excludeDirs: []string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTargetExcludeConflict(tt.targetDir, tt.excludeDirs)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkTargetExcludeConflict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsgContains != "" {
				if !strings.Contains(err.Error(), tt.errMsgContains) {
					t.Errorf("checkTargetExcludeConflict() error = %v, want error containing %q", err, tt.errMsgContains)
				}
			}
		})
	}
}

// TestIsOutsideTargetDir tests the isOutsideTargetDir function using table-driven testing.
func TestIsOutsideTargetDir(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		rootDir     string
		targetDir   string
		wantOutside bool
		wantErr     bool
	}{
		{
			name:        "no target restriction",
			path:        "/tmp/test/vendor",
			rootDir:     "/tmp/test",
			targetDir:   "",
			wantOutside: false,
			wantErr:     false,
		},
		{
			name:        "inside target directory",
			path:        "/tmp/test/cmd/server",
			rootDir:     "/tmp/test",
			targetDir:   "cmd",
			wantOutside: false,
			wantErr:     false,
		},
		{
			name:        "outside target directory",
			path:        "/tmp/test/vendor",
			rootDir:     "/tmp/test",
			targetDir:   "cmd",
			wantOutside: true,
			wantErr:     false,
		},
		{
			name:        "target directory itself",
			path:        "/tmp/test/cmd",
			rootDir:     "/tmp/test",
			targetDir:   "cmd",
			wantOutside: false,
			wantErr:     false,
		},
		{
			name:        "deeply nested inside target",
			path:        "/tmp/test/cmd/server/internal",
			rootDir:     "/tmp/test",
			targetDir:   "cmd",
			wantOutside: false,
			wantErr:     false,
		},
		{
			name:        "sibling directory outside target",
			path:        "/tmp/test/vendor",
			rootDir:     "/tmp/test",
			targetDir:   "cmd",
			wantOutside: true,
			wantErr:     false,
		},
		{
			name:        "parent directory (root)",
			path:        "/tmp/test",
			rootDir:     "/tmp/test",
			targetDir:   "cmd",
			wantOutside: false, // Root dir should not be skipped so we can traverse to target
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outside, err := isOutsideTargetDir(tt.path, tt.rootDir, tt.targetDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("isOutsideTargetDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if outside != tt.wantOutside {
				t.Errorf("isOutsideTargetDir() = %v, want %v", outside, tt.wantOutside)
			}
		})
	}
}

// TestIsExcludedDir tests the isExcludedDir function using table-driven testing.
func TestIsExcludedDir(t *testing.T) {
	tests := []struct {
		name         string
		dirPath      string
		excludeDirs  []string
		rootDir      string
		wantExcluded bool
		wantErr      bool
	}{
		{
			name:         "no exclude list",
			dirPath:      "/tmp/test/vendor",
			excludeDirs:  []string{},
			rootDir:      "/tmp/test",
			wantExcluded: false,
			wantErr:      false,
		},
		{
			name:         "root-level vendor excluded",
			dirPath:      "/tmp/test/vendor",
			excludeDirs:  []string{"vendor"},
			rootDir:      "/tmp/test",
			wantExcluded: true,
			wantErr:      false,
		},
		{
			name:         "nested vendor not excluded",
			dirPath:      "/tmp/test/subdir/vendor",
			excludeDirs:  []string{"vendor"},
			rootDir:      "/tmp/test",
			wantExcluded: false,
			wantErr:      false,
		},
		{
			name:         "multiple excludes",
			dirPath:      "/tmp/test/node_modules",
			excludeDirs:  []string{"vendor", "node_modules"},
			rootDir:      "/tmp/test",
			wantExcluded: true,
			wantErr:      false,
		},
		{
			name:         "not in exclude list",
			dirPath:      "/tmp/test/internal",
			excludeDirs:  []string{"vendor"},
			rootDir:      "/tmp/test",
			wantExcluded: false,
			wantErr:      false,
		},
		{
			name:         "root directory itself",
			dirPath:      "/tmp/test",
			excludeDirs:  []string{"vendor"},
			rootDir:      "/tmp/test",
			wantExcluded: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded, err := isExcludedDir(tt.dirPath, tt.excludeDirs, tt.rootDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("isExcludedDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if excluded != tt.wantExcluded {
				t.Errorf("isExcludedDir() = %v, want %v", excluded, tt.wantExcluded)
			}
		})
	}
}

// TestCheckPathTraversal tests the checkPathTraversal function using table-driven testing.
func TestCheckPathTraversal(t *testing.T) {
	tests := []struct {
		name           string
		rootDir        string
		targetPath     string
		wantErr        bool
		errMsgContains string
	}{
		{
			name:       "valid subdirectory",
			rootDir:    "/tmp/test",
			targetPath: "/tmp/test/cmd",
			wantErr:    false,
		},
		{
			name:           "path traversal with ../..",
			rootDir:        "/tmp/test",
			targetPath:     "/tmp/test/../../etc",
			wantErr:        true,
			errMsgContains: "path traversal",
		},
		{
			name:           "path traversal with absolute path outside",
			rootDir:        "/tmp/test",
			targetPath:     "/etc/passwd",
			wantErr:        true,
			errMsgContains: "outside project root",
		},
		{
			name:       "same directory as root",
			rootDir:    "/tmp/test",
			targetPath: "/tmp/test",
			wantErr:    false,
		},
		{
			name:       "nested subdirectory",
			rootDir:    "/tmp/test",
			targetPath: "/tmp/test/cmd/server",
			wantErr:    false,
		},
		{
			name:           "deep path traversal",
			rootDir:        "/tmp/test",
			targetPath:     "/tmp/test/../../../",
			wantErr:        true,
			errMsgContains: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPathTraversal(tt.rootDir, tt.targetPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkPathTraversal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsgContains != "" {
				if !strings.Contains(err.Error(), tt.errMsgContains) {
					t.Errorf("checkPathTraversal() error = %v, want error containing %q", err, tt.errMsgContains)
				}
			}
		})
	}
}

// ============================================================
// Integration Tests - Acceptance Criteria (AC-1 to AC-8)
// ============================================================

// setupTestProject creates a test project structure with Git repository
func setupTestProject(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "ac-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize Git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to configure git: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to configure git: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// TestAC1_TargetSubfolder_Basic tests basic target subdirectory functionality
func TestAC1_TargetSubfolder_Basic(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create structure: integration-tests/, unit-tests/, main.go
	os.MkdirAll(filepath.Join(tempDir, "integration-tests"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "unit-tests"), 0755)
	os.WriteFile(filepath.Join(tempDir, "integration-tests", "test_a.go"), []byte("package test\n// test_a"), 0644)
	os.WriteFile(filepath.Join(tempDir, "integration-tests", "test_b.go"), []byte("package test\n// test_b"), 0644)
	os.WriteFile(filepath.Join(tempDir, "unit-tests", "test_c.go"), []byte("package test\n// test_c"), 0644)
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n// main"), 0644)

	// Git add and commit
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Run buildDirectoryStructure with targetDir="integration-tests"
	_, fileContents, err := buildDirectoryStructure(tempDir, "integration-tests", []string{}, map[string]bool{}, false, 0, nil)
	if err != nil {
		t.Fatalf("buildDirectoryStructure() error = %v", err)
	}

	// Assert: Only integration-tests files included
	if _, ok := fileContents["integration-tests/test_a.go"]; !ok {
		t.Error("Expected integration-tests/test_a.go to be included")
	}
	if _, ok := fileContents["integration-tests/test_b.go"]; !ok {
		t.Error("Expected integration-tests/test_b.go to be included")
	}
	if _, ok := fileContents["unit-tests/test_c.go"]; ok {
		t.Error("Expected unit-tests/test_c.go to be excluded")
	}
	if _, ok := fileContents["main.go"]; ok {
		t.Error("Expected main.go to be excluded")
	}
}

// TestAC2_TargetSubfolder_WithExtension tests target with extension filter
func TestAC2_TargetSubfolder_WithExtension(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create structure
	os.MkdirAll(filepath.Join(tempDir, "integration-tests"), 0755)
	os.WriteFile(filepath.Join(tempDir, "integration-tests", "test.go"), []byte("package test\n// test.go"), 0644)
	os.WriteFile(filepath.Join(tempDir, "integration-tests", "test.md"), []byte("# Test"), 0644)

	// Git add and commit
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Run with targetDir and excludeExtensions
	excludeList := map[string]bool{".go": true}
	_, fileContents, err := buildDirectoryStructure(tempDir, "integration-tests", []string{}, excludeList, false, 0, nil)
	if err != nil {
		t.Fatalf("buildDirectoryStructure() error = %v", err)
	}

	// Assert: .go files excluded, .md included
	if _, ok := fileContents["integration-tests/test.go"]; ok {
		t.Error("Expected .go file to be excluded")
	}
	if _, ok := fileContents["integration-tests/test.md"]; !ok {
		t.Error("Expected .md file to be included")
	}
}

// TestAC3_ExcludeDirectories tests -exclude-dir functionality
func TestAC3_ExcludeDirectories(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create structure: vendor/, internal/, main.go
	os.MkdirAll(filepath.Join(tempDir, "vendor"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "internal"), 0755)
	os.WriteFile(filepath.Join(tempDir, "vendor", "dep.go"), []byte("package vendor"), 0644)
	os.WriteFile(filepath.Join(tempDir, "internal", "app.go"), []byte("package internal"), 0644)
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)

	// Git add and commit
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Run with excludeDirs=["vendor"]
	excludeDirs := []string{"vendor"}
	_, fileContents, err := buildDirectoryStructure(tempDir, "", excludeDirs, map[string]bool{}, false, 0, nil)
	if err != nil {
		t.Fatalf("buildDirectoryStructure() error = %v", err)
	}

	// Assert: vendor excluded, internal and main included
	if _, ok := fileContents["vendor/dep.go"]; ok {
		t.Error("Expected vendor/dep.go to be excluded")
	}
	if _, ok := fileContents["internal/app.go"]; !ok {
		t.Error("Expected internal/app.go to be included")
	}
	if _, ok := fileContents["main.go"]; !ok {
		t.Error("Expected main.go to be included")
	}
}

// TestAC4_MultipleExcludeDirectories tests multiple -exclude-dir flags
func TestAC4_MultipleExcludeDirectories(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create structure: vendor/, node_modules/, docs/, main.go
	os.MkdirAll(filepath.Join(tempDir, "vendor"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "node_modules"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "docs"), 0755)
	os.WriteFile(filepath.Join(tempDir, "vendor", "dep.go"), []byte("package vendor"), 0644)
	os.WriteFile(filepath.Join(tempDir, "node_modules", "index.js"), []byte("// js"), 0644)
	os.WriteFile(filepath.Join(tempDir, "docs", "readme.md"), []byte("# Docs"), 0644)
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)

	// Git add and commit
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Run with multiple excludeDirs
	excludeDirs := []string{"vendor", "node_modules", "docs"}
	_, fileContents, err := buildDirectoryStructure(tempDir, "", excludeDirs, map[string]bool{}, false, 0, nil)
	if err != nil {
		t.Fatalf("buildDirectoryStructure() error = %v", err)
	}

	// Assert: all three excluded, main.go included
	if _, ok := fileContents["vendor/dep.go"]; ok {
		t.Error("Expected vendor/dep.go to be excluded")
	}
	if _, ok := fileContents["node_modules/index.js"]; ok {
		t.Error("Expected node_modules/index.js to be excluded")
	}
	if _, ok := fileContents["docs/readme.md"]; ok {
		t.Error("Expected docs/readme.md to be excluded")
	}
	if _, ok := fileContents["main.go"]; !ok {
		t.Error("Expected main.go to be included")
	}
}

// TestAC5_ErrorHandling tests error handling for nonexistent directory
func TestAC5_ErrorHandling(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Try to validate nonexistent directory
	err := validateTargetDir(tempDir, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error containing 'not found', got: %v", err)
	}
}

// TestAC6_HiddenDirectory tests targeting hidden directory
func TestAC6_HiddenDirectory(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create .github/workflows/ structure
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte("name: CI"), 0644)
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)

	// Git add and commit
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Validate that .github directory can be targeted
	err := validateTargetDir(tempDir, ".github")
	if err != nil {
		t.Errorf("Expected .github directory to be valid, got error: %v", err)
	}

	// Run buildDirectoryStructure with targetDir=".github"
	_, fileContents, err := buildDirectoryStructure(tempDir, ".github", []string{}, map[string]bool{}, false, 0, nil)
	if err != nil {
		t.Fatalf("buildDirectoryStructure() error = %v", err)
	}

	// Assert: .github files included, main.go excluded
	if _, ok := fileContents[".github/workflows/ci.yml"]; !ok {
		t.Error("Expected .github/workflows/ci.yml to be included")
	}
	if _, ok := fileContents["main.go"]; ok {
		t.Error("Expected main.go to be excluded")
	}
}

// TestAC7_ConflictDetection tests conflict between target and exclude
func TestAC7_ConflictDetection(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create integration-tests directory
	os.MkdirAll(filepath.Join(tempDir, "integration-tests"), 0755)
	os.WriteFile(filepath.Join(tempDir, "integration-tests", "test.go"), []byte("package test"), 0644)

	// Test conflict detection
	err := checkTargetExcludeConflict("integration-tests", []string{"integration-tests"})
	if err == nil {
		t.Error("Expected error for conflict, got nil")
	}
	if !strings.Contains(err.Error(), "conflicts") {
		t.Errorf("Expected error containing 'conflicts', got: %v", err)
	}
}

// TestAC8_BackwardCompatibility tests backward compatibility
func TestAC8_BackwardCompatibility(t *testing.T) {
	tempDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create complex structure
	os.MkdirAll(filepath.Join(tempDir, "cmd"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "vendor"), 0755)
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tempDir, "cmd", "app.go"), []byte("package cmd"), 0644)
	os.WriteFile(filepath.Join(tempDir, "vendor", "dep.go"), []byte("package vendor"), 0644)

	// Git add and commit
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Run with empty targetDir and excludeList (backward compatible mode)
	excludeList := map[string]bool{".go": true}
	_, fileContents, err := buildDirectoryStructure(tempDir, "", []string{}, excludeList, false, 0, nil)
	if err != nil {
		t.Fatalf("buildDirectoryStructure() error = %v", err)
	}

	// Assert: No .go files included (extension filter works)
	for path := range fileContents {
		if !strings.HasSuffix(path, ".go") {
			t.Errorf("Expected only .go files to be excluded, but found: %s", path)
		}
	}

	// Run with excludeDirs (new feature but backward compatible behavior)
	excludeDirs := []string{"vendor"}
	_, fileContents, err = buildDirectoryStructure(tempDir, "", excludeDirs, map[string]bool{}, false, 0, nil)
	if err != nil {
		t.Fatalf("buildDirectoryStructure() error = %v", err)
	}

	// Assert: vendor excluded, other files included
	if _, ok := fileContents["vendor/dep.go"]; ok {
		t.Error("Expected vendor/dep.go to be excluded")
	}
	if _, ok := fileContents["cmd/app.go"]; !ok {
		t.Error("Expected cmd/app.go to be included")
	}
}
