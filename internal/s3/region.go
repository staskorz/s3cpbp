package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3GetBucketLocationAPI defines the interface for the GetBucketLocation operation
type S3GetBucketLocationAPI interface {
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
}

// GetBucketRegion determines the region where the bucket is located
func GetBucketRegion(client S3GetBucketLocationAPI, bucket string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	}

	result, err := client.GetBucketLocation(context.TODO(), input)
	if err != nil {
		return "", err
	}

	// Convert the BucketLocationConstraint to a string region
	// If the bucket is in US Standard (LocationConstraint is nil), use "us-east-1"
	var region string
	if result.LocationConstraint == "" {
		region = "us-east-1"
	} else {
		region = string(result.LocationConstraint)
	}

	return region, nil
}
