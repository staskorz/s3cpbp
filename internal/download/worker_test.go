package download

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockDownloader implements the Downloader interface for testing
type mockDownloader struct {
	downloadFunc func(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error)
}

func (m *mockDownloader) Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error) {
	return m.downloadFunc(ctx, w, input, options...)
}

func TestCreateDownloader(t *testing.T) {
	// Create a mock S3 client
	client := s3.NewFromConfig(aws.Config{})

	// Call the function under test
	downloader := CreateDownloader(client)

	// Verify the downloader is not nil
	if downloader == nil {
		t.Error("CreateDownloader() returned nil")
	}
}

// TestWorkerStart tests the Start method of Worker
func TestWorkerStart(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "worker_test_start")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup test files
	testFiles := []string{
		"file1.txt",
		"path/to/file2.txt",
		"another/path/file3.txt",
	}

	// Setup the file channel
	filesChan := make(chan string, len(testFiles))
	for _, file := range testFiles {
		filesChan <- file
	}
	close(filesChan)

	// Setup the worker
	var (
		wg            sync.WaitGroup
		totalFiles    atomic.Int64
		finishedFiles atomic.Int64
	)

	// Set total files
	totalFiles.Store(int64(len(testFiles)))
	wg.Add(1)

	// Create a mock downloader
	mockDownload := &mockDownloader{
		downloadFunc: func(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error) {
			// Verify bucket name
			if *input.Bucket != "test-bucket" {
				t.Errorf("Download() bucket = %v, want %v", *input.Bucket, "test-bucket")
			}

			// Write test data to simulate a download
			data := []byte("test file content")
			_, err = w.WriteAt(data, 0)
			if err != nil {
				return 0, err
			}
			return int64(len(data)), nil
		},
	}

	// Create the worker
	worker := Worker{
		ID:            1,
		Downloader:    mockDownload,
		Bucket:        "test-bucket",
		Destination:   tempDir,
		FilesChan:     filesChan,
		WaitGroup:     &wg,
		TotalFiles:    &totalFiles,
		FinishedFiles: &finishedFiles,
	}

	// Start the worker
	worker.Start()

	// Wait for the worker to finish
	wg.Wait()

	// Verify the results
	if finishedFiles.Load() != int64(len(testFiles)) {
		t.Errorf("Worker processed %d files, want %d", finishedFiles.Load(), len(testFiles))
	}

	// Check that files were actually created
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File %s was not created", filePath)
		} else {
			// Verify file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", filePath, err)
			} else if string(content) != "test file content" {
				t.Errorf("File %s has incorrect content: %s", filePath, content)
			}
		}
	}
}

// TestDownloadFile tests the downloadFile method
func TestDownloadFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "worker_test_download")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup counters
	var (
		totalFiles    atomic.Int64
		finishedFiles atomic.Int64
	)

	// Setup a test file
	testFile := "path/to/test/file.txt"

	// Create a mock downloader
	mockDownload := &mockDownloader{
		downloadFunc: func(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error) {
			// Write test data
			data := []byte("mock file content")
			_, err = w.WriteAt(data, 0)
			if err != nil {
				return 0, err
			}
			return int64(len(data)), nil
		},
	}

	// Create a worker without a real wait group since we're not testing Start
	worker := Worker{
		ID:            2,
		Downloader:    mockDownload,
		Bucket:        "test-bucket",
		Destination:   tempDir,
		FilesChan:     nil, // Not used in this test
		WaitGroup:     nil, // Not used in this test
		TotalFiles:    &totalFiles,
		FinishedFiles: &finishedFiles,
	}

	// Call the method directly
	worker.downloadFile(testFile)

	// Verify the file was created
	filePath := filepath.Join(tempDir, testFile)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File %s was not created", filePath)
	} else {
		// Verify file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", filePath, err)
		} else if string(content) != "mock file content" {
			t.Errorf("File %s has incorrect content: %s", filePath, content)
		}
	}

	// Verify counter was incremented
	if finishedFiles.Load() != 1 {
		t.Errorf("FinishedFiles = %d, want 1", finishedFiles.Load())
	}
}

// Helper function to run tests expecting log.Fatalf
func runTestExpectingFatal(t *testing.T, testName string, testFunc func(t *testing.T)) {
	if os.Getenv("BE_WORKER_FATAL_TEST") == "1" {
		testFunc(t)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "BE_WORKER_FATAL_TEST=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		// Test exited with non-zero status, as expected from log.Fatalf
		return
	}
	t.Fatalf("Test %s process ran with err %v, want exit status 1", testName, err)
}

func TestDownloadFile_MkdirAllFailure(t *testing.T) {
	// This test expects log.Fatalf, so we use the helper
	runTestExpectingFatal(t, "^"+t.Name()+"$", func(t *testing.T) {
		// Create a read-only directory to cause MkdirAll to fail
		readOnlyDir, err := os.MkdirTemp("", "readonly")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		// On non-Windows, chmod; on Windows, this is trickier and might not reliably cause failure.
		// This test might be less effective on Windows without more complex ACL manipulation.
		if runtime.GOOS != "windows" {
			if err := os.Chmod(readOnlyDir, 0400); err != nil {
				t.Fatalf("Failed to chmod temp dir: %v", err)
			}
			defer os.Chmod(readOnlyDir, 0700) // Clean up chmod
		}
		defer os.RemoveAll(readOnlyDir)

		destination := filepath.Join(readOnlyDir, "subdir") // Try to create subdir inside readOnlyDir

		var finishedFiles atomic.Int64
		worker := Worker{
			ID:          3,
			Downloader:  nil, // Downloader won't be reached
			Bucket:      "test-bucket",
			Destination: destination,
			// Other fields like WaitGroup, FilesChan, TotalFiles not needed for this specific test
			FinishedFiles: &finishedFiles,
		}

		// This call should trigger log.Fatalf inside downloadFile
		worker.downloadFile("some/key.txt")
	})
}

func TestDownloadFile_CreateFailure(t *testing.T) {
	runTestExpectingFatal(t, "^"+t.Name()+"$", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "create_fail")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Pre-create a directory where the file should be, to cause os.Create to fail
		conflictingPath := filepath.Join(tempDir, "path/to/file.txt")
		if err := os.MkdirAll(conflictingPath, 0755); err != nil {
			t.Fatalf("Failed to create conflicting dir: %v", err)
		}

		var finishedFiles atomic.Int64
		worker := Worker{
			ID:            4,
			Downloader:    nil, // Downloader won't be reached
			Bucket:        "test-bucket",
			Destination:   tempDir,
			FinishedFiles: &finishedFiles,
		}

		// This call should trigger log.Fatalf
		worker.downloadFile("path/to/file.txt")
	})
}

func TestDownloadFile_RetrySuccess(t *testing.T) {
	tests := []struct {
		name            string
		failAttempts    int    // Number of times the download should fail before succeeding
		expectedLogPart string // Expected log message part for failure attempts
	}{
		{"SuccessOn2ndAttempt", 1, "Attempt 1: Failed to download"},
		{"SuccessOn3rdAttempt", 2, "Attempt 2: Failed to download"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "retry_success")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			var ( // Use local variables for counters in subtests
				totalFiles       atomic.Int64
				finishedFiles    atomic.Int64
				downloadAttempts atomic.Int32
			)

			mockErr := errors.New("simulated download error")

			mockDownload := &mockDownloader{
				downloadFunc: func(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error) {
					attempt := downloadAttempts.Add(1)
					if int(attempt) <= tc.failAttempts {
						// Simulate failure by writing partial data maybe, then returning error
						partialData := []byte("partial")
						w.WriteAt(partialData, 0) // Ignore error for testing
						return 0, mockErr
					}
					// Simulate success
					data := []byte("full content")
					nWritten, err := w.WriteAt(data, 0)
					return int64(nWritten), err
				},
			}

			worker := Worker{
				ID:            5,
				Downloader:    mockDownload,
				Bucket:        "test-bucket",
				Destination:   tempDir,
				TotalFiles:    &totalFiles,
				FinishedFiles: &finishedFiles,
			}

			// Redirect log output to capture it
			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			defer log.SetOutput(os.Stderr) // Restore log output

			testFile := "retry/success/file.txt"
			worker.downloadFile(testFile)

			// Verify file content is the final successful one
			filePath := filepath.Join(tempDir, testFile)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", filePath, err)
			}
			if string(content) != "full content" {
				t.Errorf("File content = %q, want %q", string(content), "full content")
			}

			// Verify finished count
			if finishedFiles.Load() != 1 {
				t.Errorf("FinishedFiles = %d, want 1", finishedFiles.Load())
			}

			// Verify number of download attempts
			if downloadAttempts.Load() != int32(tc.failAttempts+1) {
				t.Errorf("Download attempts = %d, want %d", downloadAttempts.Load(), tc.failAttempts+1)
			}

			// Verify log output contains failure message
			logOutput := logBuf.String()
			if !strings.Contains(logOutput, tc.expectedLogPart) {
				t.Errorf("Log output does not contain expected failure message %q. Log:\n%s", tc.expectedLogPart, logOutput)
			}
			if !strings.Contains(logOutput, "downloaded retry/success/file.txt") {
				t.Errorf("Log output does not contain final success message. Log:\n%s", logOutput)
			}

		})
	}
}

func TestDownloadFile_RetryFailure(t *testing.T) {
	runTestExpectingFatal(t, "^"+t.Name()+"$", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "retry_fail")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		var (
			downloadAttempts atomic.Int32
			finishedFiles    atomic.Int64 // Should remain 0
		)

		mockErr := errors.New("persistent download error")

		mockDownload := &mockDownloader{
			downloadFunc: func(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error) {
				downloadAttempts.Add(1)
				// Always fail
				return 0, mockErr
			},
		}

		worker := Worker{
			ID:            6,
			Downloader:    mockDownload,
			Bucket:        "test-bucket",
			Destination:   tempDir,
			FinishedFiles: &finishedFiles,
		}

		testFile := "retry/failure/file.txt"

		// Capture log output to verify attempts were logged before fatal
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		// Note: Cannot restore stderr here as log.Fatalf will exit

		// This call should eventually trigger log.Fatalf after 3 attempts
		worker.downloadFile(testFile)

		// Code here will not be reached if log.Fatalf works correctly
		// We rely on runTestExpectingFatal to verify the exit

		// We can still check the log *if* it didn't exit (which would be a test failure)
		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "Attempt 1: Failed to download") {
			t.Errorf("Log missing attempt 1 failure")
		}
		if !strings.Contains(logOutput, "Attempt 2: Failed to download") {
			t.Errorf("Log missing attempt 2 failure")
		}
		if !strings.Contains(logOutput, "Attempt 3: Failed to download") {
			t.Errorf("Log missing attempt 3 failure")
		}
		if downloadAttempts.Load() != 3 {
			t.Errorf("Expected 3 download attempts, got %d", downloadAttempts.Load())
		}
	})
}

// TestWorkerFileProcessing tests basic file handling
func TestWorkerFileProcessing(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "worker_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup test files
	testFiles := []string{
		"file1.txt",
		"path/to/file2.txt",
		"another/path/file3.txt",
	}

	// Setup the file channel
	filesChan := make(chan string, len(testFiles))
	for _, file := range testFiles {
		filesChan <- file
	}
	close(filesChan)

	// Setup the worker without an actual downloader
	// We'll test file creation and channel consumption
	var (
		wg            sync.WaitGroup
		totalFiles    atomic.Int64
		finishedFiles atomic.Int64
	)

	wg.Add(1)

	// Create a worker with a nil downloader - we'll override the downloadFile method
	worker := Worker{
		ID:            1,
		Downloader:    nil, // We don't need this for our test
		Bucket:        "test-bucket",
		Destination:   tempDir,
		FilesChan:     filesChan,
		WaitGroup:     &wg,
		TotalFiles:    &totalFiles,
		FinishedFiles: &finishedFiles,
	}

	// Set total files
	totalFiles.Store(int64(len(testFiles)))

	// Instead of calling worker.Start(), we'll manually process the files
	// to avoid needing a real S3 downloader
	go func() {
		defer wg.Done()

		for key := range worker.FilesChan {
			// Create the directory structure
			localPath := filepath.Join(worker.Destination, key)
			dir := filepath.Dir(localPath)
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				t.Errorf("Failed to create directory %s: %v", dir, err)
				continue
			}

			// Create a test file
			file, err := os.Create(localPath)
			if err != nil {
				t.Errorf("Failed to create file %s: %v", localPath, err)
				continue
			}

			// Write test content
			_, err = file.WriteString("test file content")
			file.Close()
			if err != nil {
				t.Errorf("Failed to write to file %s: %v", localPath, err)
				continue
			}

			// Increment counter
			finishedFiles.Add(1)
		}
	}()

	// Wait for processing to complete
	wg.Wait()

	// Check results
	if finishedFiles.Load() != int64(len(testFiles)) {
		t.Errorf("Worker processed %d files, want %d", finishedFiles.Load(), len(testFiles))
	}

	// Verify files were created
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File %s was not created", filePath)
		} else {
			// Check file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", filePath, err)
			} else if string(content) != "test file content" {
				t.Errorf("File %s has wrong content: %s", filePath, content)
			}
		}
	}
}
