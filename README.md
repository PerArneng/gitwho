# GitWho

[![Go Report Card](https://goreportcard.com/badge/github.com/PerArneng/gitwho)](https://goreportcard.com/report/github.com/PerArneng/gitwho)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/PerArneng/gitwho)](https://github.com/PerArneng/gitwho/releases/latest)

A powerful CLI tool that analyzes git history to identify contributors responsible for files or directories in a git repository. GitWho provides detailed statistics about each contributor's impact, including commits, lines added, and lines deleted.

## Features

- Analyze contribution statistics for a specific file or directory
- Recursive analysis of all files within a directory
- Filter statistics by time range (day, week, month, year)
- Sort contributors by their total impact (additions + deletions)
- Well-formatted tabular output for easy reading
- Validates that you're in a git repository and the specified path exists

## Installation

### Using Go

```bash
go install github.com/PerArneng/gitwho@latest
```

### From Releases

Download the appropriate binary for your platform from the [releases page](https://github.com/PerArneng/gitwho/releases).

## Usage

### Basic Usage

```bash
# Analyze a specific file
gitwho path/to/file.go

# Analyze a directory (recursive)
gitwho path/to/directory
```

### Time Range Filter

Filter statistics to only include changes within a specific time range:

```bash
# Get statistics for the last day
gitwho -l day path/to/file.go

# Get statistics for the last week
gitwho -l week path/to/directory

# Get statistics for the last month
gitwho --last month path/to/file.go

# Get statistics for the last year
gitwho --last year path/to/directory
```

## Example Output

```
Contributor Statistics for main.go

NAME                           EMAIL                                COMMITS      ADDED    DELETED      TOTAL
John Doe                       john.doe@example.com                      12        450        120        570
Jane Smith                     jane.smith@example.com                     8        220         85        305
Alex Johnson                   alex@example.com                           3         45         12         57
```

## Requirements

- Git must be installed on your system
- Go 1.18 or later (for building from source)

## Building From Source

```bash
# Clone the repository
git clone https://github.com/PerArneng/gitwho.git
cd gitwho

# Build the binary
go build -o gitwho main.go

# Install to your GOPATH/bin
go install
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
