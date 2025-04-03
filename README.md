# S3 Concurrent File Copy Tool

[![Build Status](https://github.com/staskorz/s3cpbp/actions/workflows/build.yml/badge.svg)](https://github.com/staskorz/s3cpbp/actions/workflows/build.yml)
[![Test Status](https://github.com/staskorz/s3cpbp/actions/workflows/test.yml/badge.svg)](https://github.com/staskorz/s3cpbp/actions/workflows/test.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/staskorz/1200dad041f4eb3300f41fef52c9fda7/raw/s3cpbp-coverage.json)](https://github.com/staskorz/s3cpbp/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/staskorz/s3cpbp)](https://goreportcard.com/report/github.com/staskorz/s3cpbp)

A Go CLI tool that allows efficient concurrent copying of files from an AWS S3 bucket to a local directory.

## Features

- Concurrent downloading of files from S3
- Configurable concurrency level
- Automatic creation of destination directories
- Real-time progress reporting
- Handles millions of files efficiently
- Starts copying as soon as files are found (doesn't wait for complete listing)

## Prerequisites

- Go 1.18 or later
- AWS credentials configured (via environment variables, AWS credentials file, etc.)

## Installation

```bash
# Clone the repository (or download the source)
git clone https://github.com/staskorz/s3cpbp.git
cd s3cpbp

# Build the binary
go build -o s3cpbp ./cmd/s3cpbp
```

## Usage

```bash
./s3cpbp --bucket BUCKET_NAME --prefix PREFIX --destination LOCAL_DIR [--concurrency NUM_WORKERS]
```

Or using the shorthand flags:

```bash
./s3cpbp -b BUCKET_NAME -p PREFIX -d LOCAL_DIR [-c NUM_WORKERS]
```

### Parameters

- `--bucket`, `-b`: AWS S3 bucket name (required)
- `--prefix`, `-p`: Prefix for S3 objects (required)
- `--destination`, `-d`: Destination directory on local machine (required)
- `--concurrency`, `-c`: Number of concurrent downloads (default: 50)

## Examples

```bash
# Download files with a specific prefix
./s3cpbp -b my-bucket -p logs/ -d ./logs

# Download files with a specific prefix and higher concurrency
./s3cpbp -b my-bucket -p logs/ -d ./logs -c 100

```

## AWS Authentication

The tool uses the default AWS SDK credential chain, which will look for credentials in the following order:

1. Environment variables (`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. EC2 Instance Profile (if running on EC2)

Make sure your AWS credentials have sufficient permissions to list and get objects from the specified S3 bucket.

## Testing

The codebase includes comprehensive tests for all packages. To run the tests:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Generate HTML coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Using Makefile
make test       # Run tests only
make coverage   # Run tests with coverage and generate report

# Using build scripts
./scripts/build.sh test      # Run tests only (Unix/macOS)
./scripts/build.sh coverage  # Run tests with coverage (Unix/macOS)
scripts/build.bat test       # Run tests only (Windows)
scripts/build.bat coverage   # Run tests with coverage (Windows)
```

The project maintains a minimum code coverage threshold of 70%. Current coverage levels:

- cmd/s3cpbp: 94.1%
- internal/config: 83.3%
- internal/download: 73.9%
- internal/s3: 80.0%

## Building the Tool

You have several options to build the s3cpbp tool:

### Using Make (Linux/macOS/Windows with WSL)

```bash
# Build for all platforms
make build

# Clean and build
make clean build

# Build for specific platform
make windows
make linux
make darwin

# Install to your GOPATH/bin
make install
```

### Using Shell Script (Linux/macOS/Windows with Git Bash)

```bash
# Make the script executable
chmod +x scripts/build.sh

# Build for all platforms
./scripts/build.sh

# Clean previous builds
./scripts/build.sh clean
```

### Using Batch File (Windows)

```cmd
# Build for Windows
scripts/build.bat

# Clean previous builds
scripts/build.bat clean
```

### Manual Go Commands

```bash
# For your current platform
go build -o s3cpbp ./cmd/s3cpbp

# For specific platform (example: Windows)
GOOS=windows GOARCH=amd64 go build -o s3cpbp.exe ./cmd/s3cpbp
```

## Release Process

The repository includes GitHub Actions workflows that will automatically:

1. Run tests and check code coverage on every push and pull request
2. Build the tool when code is pushed to the main branch
3. Build and attach binaries when a new GitHub Release is created
