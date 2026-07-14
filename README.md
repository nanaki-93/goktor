[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![CLI](https://img.shields.io/badge/CLI-Cobra-6f42c1)](https://github.com/spf13/cobra)
[![Git](https://img.shields.io/badge/Git-go--git-f05032)](https://github.com/go-git/go-git)
[![Tests](https://img.shields.io/badge/tests-go%20test%20.%2F...-blue)](#development)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

# Goktor

Goktor is a Go CLI for inspecting directories, comparing structured CSV exports, and managing groups of Git repositories.
Goktor is a CLI tool written to manage generic daily operations.

## Features

- List files in a directory with formatted sizes.
- Scan directories recursively and print large folders sorted by size.
- Compare two delimited files by key, content, and content type.
- Normalize JSON and XML content before diffing.
- Update `origin` remotes across multiple repositories.
- Delete merged remote `feature/`, `bugfix/`, and `hotfix/` branches using release-branch ancestry.

## Requirements

- Go `1.24` or newer.
- Git repository access for `mr-repo` commands.

## Installation

Build the CLI from source:

```sh
go build -o goktor .
```

Run it locally:

```sh
./goktor --help
```

Optionally move the binary into a directory on your `PATH`:

```sh
mv ./goktor /usr/local/bin/goktor
```

## Usage

Enable verbose logs with the global `--verbose` flag:

```sh
goktor --verbose <command>
```

### List Files

Print files directly inside a directory:

```sh
goktor file-list --dir ./path/to/scan
```

If `--dir` is omitted, Goktor scans the current working directory.

### List Folders

Scan folders recursively and print directories larger than the built-in size threshold:

```sh
goktor folder-list --dir ./path/to/scan
```

The output is sorted by directory size in descending order.

### Diff Files

Compare two delimited files:

```sh
goktor diff \
  --left ./left.tsv \
  --right ./right.tsv \
  --delimiter $'\t' \
  --header
```

Input rows must contain at least three columns:

1. Key
2. Content
3. Type

Supported structured types are `json` and `xml`. Other types are compared as plain strings. The command writes timestamped `OK` and `KO` result files next to the input paths.

### Manage Multiple Repositories

`mr-repo` commands are Git operations. Run them from the intended parent directory or repository and use `--dry-run` where available before making destructive changes.

Update `origin` remotes for all immediate child repositories of the current directory:

```sh
goktor mr-repo update-remote git@github.com:new-org
```

Goktor keeps each repository name from the existing remote and builds a new remote URL from the provided base. It verifies the new remote with `fetch` and rolls back on failure unless `--force` is set:

```sh
goktor mr-repo update-remote git@github.com:new-org --force
```

Delete remote branches that are merged into `origin/release/*` branches on or before a cutoff date:

```sh
goktor mr-repo delete-merged 2026-01-31 --dry-run
```

Without `--dry-run`, this command deletes matching remote `feature/`, `bugfix/`, and `hotfix/` branches from `origin`:

```sh
goktor mr-repo delete-merged 2026-01-31
```

## Command Reference

```text
goktor
├── file-list      List files and their sizes
├── folder-list    List directories and their sizes
├── diff           Compare two delimited files
└── mr-repo        Manage Git repositories
    ├── update-remote <new-remote>
    └── delete-merged <YYYY-MM-DD>
```

## Development

Run the test suite:

```sh
go test ./...
```

Build the binary:

```sh
go build -o goktor .
```

## Project Structure

```text
cmd/          Cobra commands and CLI wiring
cmd/mr_repo/  Multi-repository Git commands
model/        Data models for file and diff operations
service/      File-system, diff, logging, and Git services
main.go       CLI entrypoint
```

## License

Goktor is released under the MIT License. See `LICENSE` for details.
