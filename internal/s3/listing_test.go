package s3

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// mockS3Client implements S3ListObjectsAPI interface for testing
type mockS3Client struct {
	// Pages to return, in order
	pages []*s3.ListObjectsV2Output
	// Current page index
	currentPage int
}

// ListObjectsV2 implements the S3ListObjectsAPI interface
func (m *mockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.currentPage >= len(m.pages) {
		return &s3.ListObjectsV2Output{Contents: []types.Object{}}, nil
	}

	response := m.pages[m.currentPage]
	m.currentPage++
	return response, nil
}

// mockPaginator mocks the behavior of ListObjectsV2Paginator
type mockPaginator struct {
	pages     []*s3.ListObjectsV2Output
	pageIndex int
}

func (m *mockPaginator) HasMorePages() bool {
	return m.pageIndex < len(m.pages)
}

func (m *mockPaginator) NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if !m.HasMorePages() {
		return nil, fmt.Errorf("no more pages")
	}
	response := m.pages[m.pageIndex]
	m.pageIndex++
	return response, nil
}

// mockListFilesImpl is a function that replaces the standard ListFiles implementation for testing
func mockListFilesImpl(pages []*s3.ListObjectsV2Output, foundFilesChan chan<- string, totalFiles *atomic.Int64) {
	defer close(foundFilesChan)

	for _, page := range pages {
		for _, obj := range page.Contents {
			totalFiles.Add(1)
			foundFilesChan <- *obj.Key
		}
	}
}

// TestListFiles tests the ListFiles function
func TestListFiles(t *testing.T) {
	tests := []struct {
		name          string
		bucket        string
		prefix        string
		pages         []*s3.ListObjectsV2Output
		expectedFiles []string
		expectedTotal int64
	}{
		{
			name:   "single page of results",
			bucket: "test-bucket",
			prefix: "test-prefix",
			pages: []*s3.ListObjectsV2Output{
				{
					Contents: []types.Object{
						{Key: aws.String("test-prefix/file1.txt")},
						{Key: aws.String("test-prefix/file2.txt")},
						{Key: aws.String("test-prefix/file3.txt")},
					},
					IsTruncated: aws.Bool(false),
				},
			},
			expectedFiles: []string{
				"test-prefix/file1.txt",
				"test-prefix/file2.txt",
				"test-prefix/file3.txt",
			},
			expectedTotal: 3,
		},
		{
			name:   "multiple pages of results",
			bucket: "test-bucket",
			prefix: "test-prefix",
			pages: []*s3.ListObjectsV2Output{
				{
					Contents: []types.Object{
						{Key: aws.String("test-prefix/file1.txt")},
						{Key: aws.String("test-prefix/file2.txt")},
					},
					IsTruncated:           aws.Bool(true),
					NextContinuationToken: aws.String("token"),
				},
				{
					Contents: []types.Object{
						{Key: aws.String("test-prefix/file3.txt")},
						{Key: aws.String("test-prefix/file4.txt")},
					},
					IsTruncated: aws.Bool(false),
				},
			},
			expectedFiles: []string{
				"test-prefix/file1.txt",
				"test-prefix/file2.txt",
				"test-prefix/file3.txt",
				"test-prefix/file4.txt",
			},
			expectedTotal: 4,
		},
		{
			name:   "empty result",
			bucket: "test-bucket",
			prefix: "test-prefix",
			pages: []*s3.ListObjectsV2Output{
				{
					Contents:    []types.Object{},
					IsTruncated: aws.Bool(false),
				},
			},
			expectedFiles: []string{},
			expectedTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a channel to receive files
			filesChan := make(chan string, len(tt.expectedFiles)+1)

			// Create a counter for total files
			var totalFiles atomic.Int64

			// Create a mock S3 client
			mockClient := &mockS3Client{
				pages: tt.pages,
			}

			// Call the actual ListFiles function with our mock client
			go ListFiles(mockClient, tt.bucket, tt.prefix, filesChan, &totalFiles)

			// Collect all files from the channel
			var files []string
			for file := range filesChan {
				files = append(files, file)
			}

			// Verify the results
			if len(files) != len(tt.expectedFiles) {
				t.Errorf("ListFiles() received %d files, want %d", len(files), len(tt.expectedFiles))
			}

			// Compare the files without caring about order
			fileMap := make(map[string]bool)
			for _, file := range files {
				fileMap[file] = true
			}

			for _, expectedFile := range tt.expectedFiles {
				if !fileMap[expectedFile] {
					t.Errorf("ListFiles() did not receive file %s", expectedFile)
				}
			}

			// Verify the total count
			if count := totalFiles.Load(); count != tt.expectedTotal {
				t.Errorf("ListFiles() total count = %d, want %d", count, tt.expectedTotal)
			}
		})
	}
}
