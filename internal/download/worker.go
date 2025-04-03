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

	// Create directories if they don't exist
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Printf("Worker %d: Failed to create directory %s: %v", w.ID, dir, err)
		return
	}

	// Create the file
	file, err := os.Create(localPath)
	if err != nil {
		log.Printf("Worker %d: Failed to create file %s: %v", w.ID, localPath, err)
		return
	}
	defer file.Close()

	// Download the file using S3 Manager
	_, err = w.Downloader.Download(context.TODO(), file, &s3.GetObjectInput{
		Bucket: aws.String(w.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Worker %d: Failed to download %s: %v", w.ID, key, err)
		return
	}

	// Increment counter and log progress
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
