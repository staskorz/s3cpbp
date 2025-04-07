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

// initializeS3Client allows for easier testing by mocking the client initialization
var initializeS3Client = func(bucket string) (*s3.Client, error) {
	// Setup initial AWS client with default configuration
	defaultAwsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	// Create a temporary S3 client to get the bucket location
	tempClient := s3.NewFromConfig(defaultAwsCfg)

	// Get the bucket's region
	region, err := s3ops.GetBucketRegion(tempClient, bucket)
	if err != nil {
		return nil, err
	}

	log.Printf("Bucket '%s' is in region '%s'", bucket, region)

	// Create a new AWS config with the correct region
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	// Create and return S3 client with the correct region
	return s3.NewFromConfig(awsCfg), nil
}

func main() {
	// Parse configuration
	cfg, showVersion := parseConfigFunc(version)
	if showVersion {
		return
	}

	// Initialize S3 client with region detection
	client, err := initializeS3Client(cfg.Bucket)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// Create downloader from client
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

	// Wait for all workers to finish
	wg.Wait()
	log.Printf("All done! Downloaded %d files from S3 bucket '%s'", finishedFiles.Load(), cfg.Bucket)
}
