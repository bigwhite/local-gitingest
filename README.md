# local-gitingest

`local-gitingest` is a command-line tool written in Go that converts a local Git repository into a single text file.  This text file includes the repository's directory structure and the contents of its source files (excluding specified file types and those exceeding a size limit). This format is suitable for providing context to large language models (LLMs) or for creating quick project snapshots.  It's inspired by the online tool [gitingest.com](https://gitingest.com/).

## Features

*   **Directory Structure:**  Generates a hierarchical representation of your project's directory structure.
*   **File Content Inclusion:** Includes the content of text-based source files.
*   **Exclusion Filters:**
    *   **Default Exclusions:** Automatically excludes executable files (files without extensions) and common directories like `.git`, `node_modules`, and `vendor`.
    *   **Extension-based Exclusion:**  Allows you to specify file extensions to exclude (e.g., `.jpg`, `.png`, `.log`).
*   **File Size Limit:**  Optionally limits the size of files included in the output.
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
local-gitingest [options]
```

**Options:**

*   `-exclude <extensions>`:  A comma-separated list of file extensions to exclude (e.g., `.jpg,.png,.gif`).  Do *not* include a space after the comma.
*   `-o <filename>`:  Specifies the output file name (default: `output.txt`).
*   `-size-limit`: Enables a file size limit.
*   `-max-size <bytes>`: Sets the maximum file size in bytes (default: 50KB, which is 51200 bytes).  This option is only used if `-size-limit` is also provided.

**Important:**  `local-gitingest` *must* be run from the root directory of a Git repository.

## Examples

*   **Basic Usage (default exclusions):**

    ```bash
    ./local-gitingest
    ```
    This will create a file named `output.txt` containing the repository structure and file contents, excluding executables and common build/dependency directories.

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

## Why use `local-gitingest`?

*   **LLM Context:**  Provide a concise representation of your codebase to large language models for tasks like code completion, documentation generation, or code analysis.
*   **Project Snapshots:**  Quickly create a text-based snapshot of your project at a specific point in time.
* **Offline usage:** Unlike gitingest.com, local-gitingest can be used offline.

