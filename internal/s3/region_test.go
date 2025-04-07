package s3

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MockS3GetBucketLocationClient is a mock implementation of S3GetBucketLocationAPI
type MockS3GetBucketLocationClient struct {
	GetBucketLocationFunc func(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
}

func (m *MockS3GetBucketLocationClient) GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return m.GetBucketLocationFunc(ctx, params, optFns...)
}

func TestGetBucketRegion(t *testing.T) {
	tests := []struct {
		name           string
		bucket         string
		mockResponse   *s3.GetBucketLocationOutput
		mockError      error
		expectedRegion string
		expectError    bool
	}{
		{
			name:   "US Standard Region",
			bucket: "us-east-bucket",
			mockResponse: &s3.GetBucketLocationOutput{
				LocationConstraint: "",
			},
			mockError:      nil,
			expectedRegion: "us-east-1",
			expectError:    false,
		},
		{
			name:   "EU Region",
			bucket: "eu-bucket",
			mockResponse: &s3.GetBucketLocationOutput{
				LocationConstraint: types.BucketLocationConstraintEuWest1,
			},
			mockError:      nil,
			expectedRegion: "eu-west-1",
			expectError:    false,
		},
		{
			name:           "Error Response",
			bucket:         "error-bucket",
			mockResponse:   nil,
			mockError:      errors.New("bucket not found"),
			expectedRegion: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockS3GetBucketLocationClient{
				GetBucketLocationFunc: func(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
					if *params.Bucket != tt.bucket {
						t.Errorf("GetBucketLocation() called with wrong bucket name: got %v, want %v", *params.Bucket, tt.bucket)
					}
					return tt.mockResponse, tt.mockError
				},
			}

			region, err := GetBucketRegion(mockClient, tt.bucket)

			// Check error
			if (err != nil) != tt.expectError {
				t.Errorf("GetBucketRegion() error = %v, expectError %v", err, tt.expectError)
				return
			}

			// Check region when we don't expect an error
			if !tt.expectError && region != tt.expectedRegion {
				t.Errorf("GetBucketRegion() got = %v, want %v", region, tt.expectedRegion)
			}
		})
	}
}
