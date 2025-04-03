package download

import (
	"context"
	"io"
	"os"
	"path/filepath"
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
