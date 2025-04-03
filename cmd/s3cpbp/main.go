package main

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/user/s3cpbp/internal/config"
	"github.com/user/s3cpbp/internal/download"
	s3ops "github.com/user/s3cpbp/internal/s3"
)

// Set during build by -ldflags
var version = "dev"

// parseConfigFunc allows for easier testing by mocking config parsing
var parseConfigFunc = appconfig.Parse

func main() {
	// Parse configuration
	cfg, showVersion := parseConfigFunc(version)
	if showVersion {
		return
	}

	// Setup AWS client
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Setup S3 client
	client := s3.NewFromConfig(awsCfg)
	downloader := download.CreateDownloader(client)

	// Setup counters
	var (
		totalFiles    atomic.Int64
		finishedFiles atomic.Int64
		wg            sync.WaitGroup
	)

	// Channel to communicate files to be downloaded
	foundFilesChan := make(chan string, 1000)

	// Start listing files
	go s3ops.ListFiles(client, cfg.Bucket, cfg.Prefix, foundFilesChan, &totalFiles)

	// Start worker pool for downloading
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		worker := download.Worker{
			ID:            i,
			Downloader:    downloader,
			Bucket:        cfg.Bucket,
			Destination:   cfg.Destination,
			FilesChan:     foundFilesChan,
			WaitGroup:     &wg,
			TotalFiles:    &totalFiles,
			FinishedFiles: &finishedFiles,
		}
		go worker.Start()
	}

	// Wait for all downloads to complete
	wg.Wait()
	log.Printf("All done! Downloaded %d files from S3 bucket '%s'", finishedFiles.Load(), cfg.Bucket)
}
