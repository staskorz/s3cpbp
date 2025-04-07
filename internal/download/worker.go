package download

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Downloader defines an interface for the S3 download functionality
type Downloader interface {
	Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error)
}

// Worker represents a download worker
type Worker struct {
	ID            int
	Downloader    Downloader
	Bucket        string
	Destination   string
	FilesChan     <-chan string
	WaitGroup     *sync.WaitGroup
	TotalFiles    *atomic.Int64
	FinishedFiles *atomic.Int64
}

// Start starts the download worker
func (w *Worker) Start() {
	defer w.WaitGroup.Done()

	for key := range w.FilesChan {
		w.downloadFile(key)
	}
}

// downloadFile downloads a single file from S3
func (w *Worker) downloadFile(key string) {
	localPath := filepath.Join(w.Destination, key)
	var err error

	// Create directories if they don't exist (only attempt once)
	dir := filepath.Dir(localPath)
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Fatalf("Worker %d: Failed to create directory %s: %v", w.ID, dir, err)
	}

	// Create the file (only attempt once)
	file, err := os.Create(localPath)
	if err != nil {
		log.Fatalf("Worker %d: Failed to create file %s: %v", w.ID, localPath, err)
	}
	defer file.Close()

	// Retry logic for download only
	for attempt := 1; attempt <= 3; attempt++ {
		// Download the file using S3 Manager
		_, err = w.Downloader.Download(context.TODO(), file, &s3.GetObjectInput{
			Bucket: aws.String(w.Bucket),
			Key:    aws.String(key),
		})

		if err == nil {
			// Success!
			break // Exit retry loop
		}

		// Log failure and prepare for next attempt (if any)
		log.Printf("Worker %d: Attempt %d: Failed to download %s: %v", w.ID, attempt, key, err)
		if attempt == 3 {
			// Clean up the potentially partially downloaded file on final failure
			// We need to close the file first before removing it
			file.Close()
			os.Remove(localPath)
			log.Fatalf("Worker %d: Failed to download %s after 3 attempts: %v", w.ID, key, err)
		}
		// Optional: Add a small delay before retrying
		// time.Sleep(1 * time.Second)

		// Reset file pointer to the beginning for the next download attempt
		_, seekErr := file.Seek(0, io.SeekStart)
		if seekErr != nil {
			log.Printf("Worker %d: Failed to seek file %s before retry: %v", w.ID, localPath, seekErr)
			file.Close() // Close before removing
			os.Remove(localPath)
			log.Fatalf("Worker %d: Unrecoverable error seeking file %s: %v", w.ID, localPath, seekErr)
		}
		err = file.Truncate(0) // Truncate the file to overwrite potentially partial download
		if err != nil {
			log.Printf("Worker %d: Failed to truncate file %s before retry: %v", w.ID, localPath, err)
			file.Close() // Close before removing
			os.Remove(localPath)
			log.Fatalf("Worker %d: Unrecoverable error truncating file %s: %v", w.ID, localPath, err)
		}
	}

	// If we reach here, the download was successful within the retry loop.
	// The redundant check for err != nil after the loop is removed as log.Fatalf would have exited.

	// Increment counter and log progress only on success
	finished := w.FinishedFiles.Add(1)
	total := w.TotalFiles.Load()

	log.Printf("Worker %d (%d/%d), downloaded %s", w.ID, finished, total, key)
}

// CreateDownloader creates a new S3 downloader
func CreateDownloader(client *s3.Client) *manager.Downloader {
	return manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = 5 * 1024 * 1024 // 5MB per part
		d.Concurrency = 3            // 3 go routines per file download
	})
}
