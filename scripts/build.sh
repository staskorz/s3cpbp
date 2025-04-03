#!/bin/bash
set -e

# Build script for s3cpbp - S3 Copy Bucket with Prefix

VERSION="0.1.0"
BINARY_NAME="s3cpbp"
OUTPUT_DIR="./bin"
MIN_COVERAGE=70

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Run tests and check coverage
run_tests() {
    echo "Running tests..."
    go test ./... -v
}

# Run tests with coverage
check_coverage() {
    echo "Running tests with coverage..."
    go test ./... -coverprofile=coverage.out
    go tool cover -func=coverage.out
    
    # Check if coverage meets minimum threshold
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    echo "Total coverage: $COVERAGE%"
    
    if (( $(echo "$COVERAGE < $MIN_COVERAGE" | bc -l) )); then
        echo "Code coverage is below $MIN_COVERAGE%"
        exit 1
    fi
    
    echo "Coverage threshold met."
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    echo "Coverage report generated at coverage.html"
}

# Build for different platforms
build() {
    GOOS=$1
    GOARCH=$2
    OUTPUT_NAME=$3
    echo "Building for $GOOS/$GOARCH..."
    
    # Set the output binary name with extension for Windows
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_PATH="$OUTPUT_DIR/${OUTPUT_NAME}.exe"
    else
        OUTPUT_PATH="$OUTPUT_DIR/${OUTPUT_NAME}"
    fi
    
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w -X main.version=$VERSION" -o "$OUTPUT_PATH" ./cmd/s3cpbp
    
    echo "Built $OUTPUT_PATH"
}

# Clean up existing binaries
clean() {
    echo "Cleaning up previous builds..."
    rm -rf "$OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"
    rm -f coverage.out coverage.html
}

# Default action is to build for all platforms
if [ "$1" = "clean" ]; then
    clean
    exit 0
fi

if [ "$1" = "test" ]; then
    run_tests
    exit 0
fi

if [ "$1" = "coverage" ]; then
    check_coverage
    exit 0
fi

echo "Building s3cpbp v$VERSION..."

# Run tests before building
run_tests

# Build for common platforms
build "linux" "amd64" "${BINARY_NAME}_linux_amd64"
build "windows" "amd64" "${BINARY_NAME}_windows_amd64"
build "darwin" "amd64" "${BINARY_NAME}_darwin_amd64"
build "darwin" "arm64" "${BINARY_NAME}_darwin_arm64"

echo "All builds completed!"
echo "Binaries are available in $OUTPUT_DIR directory" 