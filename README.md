# local-gitingest

`local-gitingest` is a command-line tool written in Go that converts a local Git repository into a single text file.  This text file includes the repository's directory structure and the contents of its source files (excluding specified file types and those exceeding a size limit). This format is suitable for providing context to large language models (LLMs) or for creating quick project snapshots.  It's inspired by the online tool [gitingest.com](https://gitingest.com/).

## Features

*   **Directory Structure:**  Generates a hierarchical representation of your project's directory structure.
*   **File Content Inclusion:** Includes the content of text-based source files.
*   **Exclusion Filters:**
    *   **Default Exclusions:** Automatically excludes executable files (files without extensions) and common directories like `.git`, `node_modules`, and `vendor`.
    *   **`.gitignore` Support:**  Automatically respects `.gitignore` patterns from your repository root.
    *   **Extension-based Exclusion:**  Allows you to specify file extensions to exclude (e.g., `.jpg`, `.png`, `.log`).
    *   **Directory-based Exclusion:**  Allows you to exclude specific directories at the root level.
*   **Target Subdirectory:**  Optionally ingest only a specific subdirectory instead of the entire repository.
*   **File Size Limit:**  Optionally limits the size of files included in the output.
*   **Verbose Mode:**  Optional verbose output (`-v`) shows which files are being excluded.
*   **Git Repository Root Check:**  Ensures the tool is run from the root directory of a Git repository.

## Installation

1.  **Make sure you have Go installed (version 1.16 or later).**
2.  **Use go install**:
    ```
    $go install github.com/bigwhite/local-gitingest@latest
    ```

    or 
    
    **Build the executable from source**:
    ```bash
    $git clone https://github.com/bigwhite/local-gitingest.git
	$cd local-gitingest
    $make
    ```

## Usage

```bash
local-gitingest [options] [target-subdirectory]
```

**Options:**

*   `-d <directory>`:  Target subdirectory to ingest. Only this directory (recursively) will be included in the output. If not specified, the entire repository is processed.
*   `-exclude-dir <dir>`:  Directories to exclude at the root level (can be specified multiple times). For example, `-exclude-dir=vendor -exclude-dir=node_modules`.
*   `-exclude <extensions>`:  A comma-separated list of file extensions to exclude (e.g., `.jpg,.png,.gif`).  Do *not* include a space after the comma.
*   `-o <filename>`:  Specifies the output file name (default: `output.txt`).
*   `-size-limit`: Enables a file size limit.
*   `-max-size <bytes>`: Sets the maximum file size in bytes (default: 50KB, which is 51200 bytes).  This option is only used if `-size-limit` is also provided.
*   `-v, -verbose`: Enables verbose mode, showing files excluded by filters.

**Arguments:**

*   `target-subdirectory`:  Alternative to `-d` flag. Specifies the subdirectory to ingest.

**Important:**  `local-gitingest` *must* be run from the root directory of a Git repository.

## Examples

*   **Basic Usage (default exclusions):**

    ```bash
    ./local-gitingest
    ```
    This will create a file named `output.txt` containing the repository structure and file contents, excluding executables and common build/dependency directories.

*   **Target a specific subdirectory:**

    ```bash
    ./local-gitingest integration-tests
    ```
    or
    ```bash
    ./local-gitingest -d integration-tests
    ```
    This will only ingest files from the `integration-tests` directory.

*   **Exclude specific directories:**

    ```bash
    ./local-gitingest -exclude-dir=vendor -exclude-dir=node_modules
    ```
    This will exclude the `vendor` and `node_modules` directories from the output.

*   **Combine target subdirectory with extension filter:**

    ```bash
    ./local-gitingest -d cmd/server -exclude go,sum
    ```
    This will only ingest files from the `cmd/server` directory, excluding `.go` and `.sum` files.

*   **Exclude specific file types:**

    ```bash
    ./local-gitingest -exclude .log,.tmp,.bak
    ```
    This will exclude files with the extensions `.log`, `.tmp`, and `.bak`.

*   **Specify output file and enable size limit:**

    ```bash
    ./local-gitingest -o my_repo.txt -size-limit -max-size 102400
    ```
    This will create a file named `my_repo.txt`, and only files smaller than 100KB (102400 bytes) will be included.

* **Exclude specific file types and enable size limit:**
   ```bash
    ./local-gitingest -exclude .log,.tmp,.bak -o my_repo.txt -size-limit -max-size 102400
    ```
    This will create a file named `my_repo.txt`, exclude files with the extensions `.log`, `.tmp`, and `.bak`. and only files smaller than 100KB (102400 bytes) will be included.

*   **Verbose mode (shows excluded files):**

    ```bash
    ./local-gitingest -v
    ```
    This will show which files are being excluded by `.gitignore` patterns.

*   **`.gitignore` support:**
    The tool automatically respects your repository's `.gitignore` file. Patterns like `*.log`, `build/`, `*.tmp` will exclude matching files and directories from the output.

## Why use `local-gitingest`?

*   **LLM Context:**  Provide a concise representation of your codebase to large language models for tasks like code completion, documentation generation, or code analysis.
*   **Project Snapshots:**  Quickly create a text-based snapshot of your project at a specific point in time.
* **Offline usage:** Unlike gitingest.com, local-gitingest can be used offline.

