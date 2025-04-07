package config

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// Config holds the application configuration
type Config struct {
	Bucket      string
	Prefix      string
	Destination string
	Concurrency int
	Version     string
}

// Parse parses command line flags and returns application configuration
func Parse(version string) (*Config, bool) {
	var (
		bucket      string
		prefix      string
		destination string
		concurrency int
		showVersion bool
	)

	// Parse command line flags
	flag.StringVar(&bucket, "bucket", "", "AWS S3 bucket name")
	flag.StringVar(&bucket, "b", "", "AWS S3 bucket name (shorthand)")

	flag.StringVar(&prefix, "prefix", "", "Prefix for S3 objects")
	flag.StringVar(&prefix, "p", "", "Prefix for S3 objects (shorthand)")

	flag.StringVar(&destination, "destination", "", "Destination directory on local machine")
	flag.StringVar(&destination, "d", "", "Destination directory on local machine (shorthand)")

	flag.IntVar(&concurrency, "concurrency", 50, "Number of concurrent downloads")
	flag.IntVar(&concurrency, "c", 50, "Number of concurrent downloads (shorthand)")

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")

	flag.Parse()

	// Show version and exit if requested
	if showVersion {
		fmt.Printf("s3cpbp version %s\n", version)
		return nil, true
	}

	// Validate required parameters
	if bucket == "" {
		log.Fatal("Bucket name is required")
	}

	if prefix == "" {
		log.Fatal("Prefix is required")
	}

	if destination == "" {
		log.Fatal("Destination directory is required")
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destination, os.ModePerm); err != nil {
		log.Fatalf("Failed to create destination directory: %v", err)
	}

	return &Config{
		Bucket:      bucket,
		Prefix:      prefix,
		Destination: destination,
		Concurrency: concurrency,
		Version:     version,
	}, false
}
