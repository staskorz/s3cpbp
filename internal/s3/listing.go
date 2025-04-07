package s3

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3ListObjectsAPI defines the interface for the ListObjectsV2 operation
type S3ListObjectsAPI interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// ListFiles lists files from S3 bucket with the given prefix
// and sends them to the provided channel
func ListFiles(client S3ListObjectsAPI, bucket, prefix string, foundFilesChan chan<- string, totalFiles *atomic.Int64) {
	defer close(foundFilesChan)

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Printf("Error listing objects: %v", err)
			return
		}

		for _, obj := range page.Contents {
			totalFiles.Add(1)
			foundFilesChan <- *obj.Key
		}
	}
}
