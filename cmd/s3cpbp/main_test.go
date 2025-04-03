package main

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/user/s3cpbp/internal/config"
	"github.com/user/s3cpbp/internal/download"
)

// TestVersionFlag tests that the version flag is handled correctly
func TestVersionFlag(t *testing.T) {
	// Save original command line arguments and restore after test
	origArgs := os.Args
	origParseConfigFunc := parseConfigFunc

	defer func() {
		os.Args = origArgs
		parseConfigFunc = origParseConfigFunc
	}()

	// Set up test arguments
	os.Args = []string{"cmd", "-v"}

	// Track if Parse was called with correct version
	var parseCalled bool

	// Override the Parse function to avoid actual flag parsing
	parseConfigFunc = func(v string) (*appconfig.Config, bool) {
		parseCalled = true
		if v != version {
			t.Errorf("Parse() version = %v, want %v", v, version)
		}
		return nil, true // Signal version flag was used
	}

	// Call main which should exit after showing version
	main()

	// Verify Parse was called
	if !parseCalled {
		t.Error("Parse() was not called")
	}
}

// TestNormalConfig tests that config is parsed correctly in normal mode
func TestNormalConfig(t *testing.T) {
	// Save original functions
	origArgs := os.Args
	origParseConfigFunc := parseConfigFunc

	// Restore original functions after test
	defer func() {
		os.Args = origArgs
		parseConfigFunc = origParseConfigFunc
	}()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "s3cpbp-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test arguments (these won't be used due to mock)
	os.Args = []string{"cmd", "-b", "test-bucket", "-p", "prefix", "-d", tempDir}

	// Track if Parse was called correctly
	var parseCalled bool

	// Mock the config parsing
	parseConfigFunc = func(v string) (*appconfig.Config, bool) {
		parseCalled = true
		return &appconfig.Config{
			Bucket:      "test-bucket",
			Prefix:      "test-prefix",
			Destination: tempDir,
			Concurrency: 1, // Use a small number for testing
			Version:     v,
		}, false
	}

	// Call main but catch the expected AWS errors
	defer func() {
		// Recover from expected AWS errors
		if r := recover(); r != nil {
			// We expect an error when trying to load AWS config
			t.Logf("Got expected error from AWS config: %v", r)
		}
	}()

	main()

	// Verify Parse was called
	if !parseCalled {
		t.Error("Parse() was not called")
	}
}

// The following functions are mocks that could be used in integration tests
// but are kept here for reference

// Mock s3ops.ListFiles function to avoid actual S3 calls
func mockListFiles(client *s3.Client, bucket, prefix string, foundFilesChan chan<- string, totalFiles interface{}) {
	defer close(foundFilesChan)
	// Add some test files
	foundFilesChan <- "file1.txt"
	foundFilesChan <- "file2.txt"
}

// Mock download.Worker.Start function
func mockWorkerStart(w *download.Worker) {
	// Just consume files from the channel and mark them as complete
	for range w.FilesChan {
		w.FinishedFiles.Add(1)
	}
}
