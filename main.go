package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Pattern represents a single .gitignore rule
type Pattern struct {
	raw       string // Original pattern string from .gitignore
	anchored  bool   // Leading "/" means anchor to root
	isDir     bool   // Trailing "/" means directory-only
	wildcard  bool   // Contains "*" or "?"
	charRange bool   // Contains "[...]"
}

// ParsePattern converts a .gitignore line into a Pattern struct.
// Returns (nil, nil) for empty lines, comments, and whitespace-only lines.
func ParsePattern(line string) (*Pattern, error) {
	// Trim whitespace
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	p := &Pattern{raw: line}

	// Check for leading "/" (anchored to root)
	if strings.HasPrefix(line, "/") {
		p.anchored = true
		line = line[1:]
		// Update raw pattern without leading "/" for matching
		// But keep original raw for reference
	}

	// Check for trailing "/" (directory only)
	if strings.HasSuffix(line, "/") {
		p.isDir = true
		line = line[:len(line)-1]
	}

	// Check for wildcards
	p.wildcard = strings.ContainsAny(line, "*?")
	p.charRange = strings.Contains(line, "[")

	return p, nil
}

// GitignoreMatcher encapsulates gitignore pattern matching logic.
type GitignoreMatcher struct {
	patterns  []*Pattern
	verbose   bool
	warnCount int
}

// LoadGitignore reads and parses .gitignore from repository root.
// Returns empty matcher if .gitignore doesn't exist.
func LoadGitignore(rootDir string, verbose bool) (*GitignoreMatcher, error) {
	m := &GitignoreMatcher{
		patterns: make([]*Pattern, 0),
		verbose:  verbose,
	}

	gitignorePath := filepath.Join(rootDir, ".gitignore")

	// If .gitignore doesn't exist, return empty matcher
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return m, nil
	}

	file, err := os.Open(gitignorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		pattern, err := ParsePattern(line)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: .gitignore:%d: %v\n", lineNum, err)
			m.warnCount++
			continue
		}

		if pattern != nil {
			m.patterns = append(m.patterns, pattern)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .gitignore: %w", err)
	}

	return m, nil
}

// matchesPattern implements gitignore pattern matching logic.
func (m *GitignoreMatcher) matchesPattern(p *Pattern, relPath string, isDir bool) bool {
	// Directory-only pattern check
	if p.isDir && !isDir {
		return false
	}

	// Prepare pattern for matching (remove leading/trailing slashes)
	pattern := p.raw
	if strings.HasPrefix(pattern, "/") {
		pattern = pattern[1:]
	}
	if strings.HasSuffix(pattern, "/") {
		pattern = pattern[:len(pattern)-1]
	}

	// For anchored patterns, match against full path
	if p.anchored {
		matched, _ := filepath.Match(pattern, relPath)
		return matched
	}

	// For non-anchored patterns, check each path component
	components := strings.Split(relPath, string(filepath.Separator))
	for _, component := range components {
		matched, _ := filepath.Match(pattern, component)
		if matched {
			return true
		}
	}

	return false
}

// IsIgnored checks if a relative path matches any gitignore pattern.
func (m *GitignoreMatcher) IsIgnored(relPath string, isDir bool) bool {
	for _, p := range m.patterns {
		if m.matchesPattern(p, relPath, isDir) {
			if m.verbose {
				fmt.Fprintf(os.Stderr, "Excluded by .gitignore: %s\n", relPath)
			}
			return true
		}
	}
	return false
}

var (
	excludeExtensions string
	outputFilename    string
	includeSizeLimit  bool
	sizeLimit         int64
	verbose           bool
)

func init() {
	flag.StringVar(&excludeExtensions, "exclude", "", "Comma-separated list of file extensions to exclude (e.g., .jpg,.png,.gif)")
	flag.StringVar(&outputFilename, "o", "output.txt", "Output file name")
	flag.BoolVar(&includeSizeLimit, "size-limit", false, "Enable file size limit")
	flag.Int64Var(&sizeLimit, "max-size", 50*1024, "Maximum file size in bytes (default: 50KB)") // 50KB default
	flag.BoolVar(&verbose, "v", false, "Enable verbose mode (show excluded files)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode (show excluded files)")
}

func usage() {
	fmt.Println("local-gitingest: Convert a local Git repository to a single text file.")
	fmt.Println("\nUsage: local-gitingest [options]")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nThis tool must be run from the root directory of a Git repository.")
	fmt.Println("It generates a text file containing the repository's directory structure and file contents,")
	fmt.Println("excluding specified file types and those exceeding a size limit.")
	fmt.Println("This is useful for providing context to large language models or creating project snapshots.")
}

func main() {
	flag.Usage = usage // Set custom usage function
	flag.Parse()

	// 检查是否在 Git 仓库的根目录下
	if !isGitRoot() {
		fmt.Fprintln(os.Stderr, "Error: This tool must be run from the root directory of a Git repository.")
		os.Exit(1)
	}

	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Load .gitignore patterns
	gitignore, err := LoadGitignore(rootDir, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		// Continue without gitignore
		gitignore = &GitignoreMatcher{patterns: nil, verbose: verbose}
	}

	// 构建排除列表，默认排除可执行文件
	excludeList := map[string]bool{
		"": true, // 排除没有扩展名的文件，通常是可执行文件
	}
	if excludeExtensions != "" {
		for _, ext := range strings.Split(excludeExtensions, ",") {
			excludeList[strings.TrimSpace(ext)] = true
		}
	}

	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if err := writeDirectoryStructure(rootDir, excludeList, includeSizeLimit, sizeLimit, gitignore, outFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing directory structure: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated output to %s\n", outputFilename)
}

// isGitRoot 检查当前目录是否为 Git 仓库的根目录
func isGitRoot() bool {
	// 最简单的方法：检查是否存在 .git 目录
	_, err := os.Stat(".git")
	if err == nil {
		return true // .git directory exists
	}

	// 更严谨的方法：使用 git rev-parse --show-toplevel 命令 (更可靠，但稍慢)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	err = cmd.Run()
	return err == nil // If the command runs successfully, we are in a git repo (possibly a subdirectory)
}

func writeDirectoryStructure(rootDir string, excludeList map[string]bool, includeSizeLimit bool, sizeLimit int64, gitignore *GitignoreMatcher, out io.Writer) error {
	dirStructure, fileContents, err := buildDirectoryStructure(rootDir, excludeList, includeSizeLimit, sizeLimit, gitignore)
	if err != nil {
		return err
	}
	return writeOutput(out, dirStructure, fileContents)
}

func buildDirectoryStructure(rootDir string, excludeList map[string]bool, includeSizeLimit bool, sizeLimit int64, gitignore *GitignoreMatcher) (string, map[string]string, error) {
	var dirStructure strings.Builder
	fileContents := make(map[string]string)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// === Exclusion Priority Chain ===
		// 1. .gitignore rules (highest priority)
		if gitignore != nil && gitignore.IsIgnored(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 2. Hard-coded exclusions (hidden dirs, node_modules, vendor)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != "./" {
			return filepath.SkipDir
		}

		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "vendor") {
			return filepath.SkipDir
		}

		// 3. CLI -exclude flag (file extensions)
		if !d.IsDir() {
			ext := filepath.Ext(d.Name())
			if excludeList[ext] {
				return nil
			}
		}

		// 4. File size limit
		if includeSizeLimit && !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Size() > sizeLimit {
				return nil
			}
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("    ", depth)

		if d.IsDir() {
			dirStructure.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
		} else {
			dirStructure.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name())) //只写入目录结构
			content, err := os.ReadFile(path)                                 //读取文件内容
			if err != nil {
				return err
			}
			fileContents[relPath] = string(content) //将文件内容存入map
		}
		return nil
	})

	if err != nil {
		return "", nil, err
	}

	return dirStructure.String(), fileContents, nil
}

func writeOutput(out io.Writer, dirStructure string, fileContents map[string]string) error {
	io.WriteString(out, dirStructure)
	io.WriteString(out, "\n")
	for relPath, content := range fileContents {
		io.WriteString(out, fmt.Sprintf("================================================\n"))
		io.WriteString(out, fmt.Sprintf("File: %s\n", relPath))
		io.WriteString(out, fmt.Sprintf("================================================\n"))
		io.WriteString(out, content)
		io.WriteString(out, "\n\n")
	}
	return nil
}
