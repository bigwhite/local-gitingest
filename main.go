package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	excludeExtensions string
	outputFilename    string
	includeSizeLimit  bool
	sizeLimit         int64
)

func init() {
	flag.StringVar(&excludeExtensions, "exclude", "", "Comma-separated list of file extensions to exclude (e.g., .jpg,.png,.gif)")
	flag.StringVar(&outputFilename, "o", "output.txt", "Output file name")
	flag.BoolVar(&includeSizeLimit, "size-limit", false, "Enable file size limit")
	flag.Int64Var(&sizeLimit, "max-size", 50*1024, "Maximum file size in bytes (default: 50KB)") // 50KB default
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

	if err := writeDirectoryStructure(rootDir, excludeList, includeSizeLimit, sizeLimit, outFile); err != nil {
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

func writeDirectoryStructure(rootDir string, excludeList map[string]bool, includeSizeLimit bool, sizeLimit int64, outFile *os.File) error {
	var dirStructure strings.Builder
	var fileContents strings.Builder

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor") {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("    ", depth)

		if d.IsDir() {
			dirStructure.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
		} else {
			ext := filepath.Ext(d.Name())
			if excludeList[ext] {
				return nil
			}

			if includeSizeLimit {
				info, err := d.Info()
				if err != nil {
					return err
				}
				if info.Size() > sizeLimit {
					return nil
				}
			}
			dirStructure.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fileContents.WriteString(fmt.Sprintf("================================================\n"))
			fileContents.WriteString(fmt.Sprintf("File: %s\n", relPath))
			fileContents.WriteString(fmt.Sprintf("================================================\n"))
			fileContents.WriteString(string(content))
			fileContents.WriteString("\n\n")
		}
		return nil
	})

	if err != nil {
		return err
	}
	outFile.WriteString(dirStructure.String())
	outFile.WriteString("\n")
	outFile.WriteString(fileContents.String())
	return nil
}
