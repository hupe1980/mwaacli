package s3

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Client struct {
	client *s3.Client
}

// NewClient creates a new Client with the provided AWS configuration.
func NewClient(cfg aws.Config) *Client {
	return &Client{
		client: s3.NewFromConfig(cfg),
	}
}

// DownloadAndUnzip downloads a file from S3 and unzips it to the specified directory.
func (s *Client) DownloadAndUnzip(ctx context.Context, s3Path, destDir string) error {
	bucket, key, err := parseS3Path(s3Path)
	if err != nil {
		return fmt.Errorf("invalid S3 path: %w", err)
	}

	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download file from S3: %w", err)
	}
	defer output.Body.Close()

	// Read the object into memory
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, output.Body); err != nil {
		return fmt.Errorf("failed to read S3 object: %w", err)
	}

	// Unzip the file
	if err := unzip(buf.Bytes(), destDir); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}

	return nil
}

// SyncDirectory synchronizes files from an S3 bucket to a local directory.
func (s *Client) SyncDirectory(ctx context.Context, s3Path, localDir string) error {
	bucket, prefix, err := parseS3Path(s3Path)
	if err != nil {
		return fmt.Errorf("invalid S3 path: %w", err)
	}

	listOutput, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return fmt.Errorf("failed to list objects in S3 bucket: %w", err)
	}

	// Create a map of S3 objects for comparison
	s3Objects := make(map[string]types.Object)

	for _, obj := range listOutput.Contents {
		relativePath := strings.TrimPrefix(*obj.Key, prefix)
		s3Objects[relativePath] = obj
	}

	// Ensure the local directory exists
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Download files from S3
	for relativePath, obj := range s3Objects {
		localFilePath := filepath.Join(localDir, relativePath)

		// Create parent directories if necessary
		if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create directories for %s: %w", localFilePath, err)
		}

		// Download the file
		getOutput, err := s.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", *obj.Key, err)
		}
		defer getOutput.Body.Close()

		// Write the file to the local directory
		localFile, err := os.Create(localFilePath)
		if err != nil {
			return fmt.Errorf("failed to create local file %s: %w", localFilePath, err)
		}
		defer localFile.Close()

		if _, err := io.Copy(localFile, getOutput.Body); err != nil {
			return fmt.Errorf("failed to write to local file %s: %w", localFilePath, err)
		}

		// Set the file's modification time to match the S3 object's LastModified
		if err := os.Chtimes(localFilePath, *obj.LastModified, *obj.LastModified); err != nil {
			return fmt.Errorf("failed to set timestamp for %s: %w", localFilePath, err)
		}
	}

	// Delete local files not present in the S3 bucket
	err = filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the relative path of the local file
		relativePath, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		// Check if the file exists in the S3 bucket
		if _, exists := s3Objects[relativePath]; !exists {
			// Delete the file if it doesn't exist in the S3 bucket
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete local file %s: %w", path, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to clean up local files: %w", err)
	}

	return nil
}

// parseS3Path parses an S3 path into bucket and key/prefix.
func parseS3Path(s3Path string) (bucket, key string, err error) {
	if !strings.HasPrefix(s3Path, "s3://") {
		return "", "", fmt.Errorf("invalid S3 path: %s", s3Path)
	}

	parts := strings.SplitN(s3Path[5:], "/", 2)

	bucket = parts[0]

	if len(parts) > 1 {
		key = parts[1]
	}

	return bucket, key, nil
}

// unzip extracts a zip archive from a byte slice to a destination directory.
func unzip(data []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	for _, file := range reader.File {
		filePath := filepath.Join(dest, filepath.Clean(file.Name))

		// Ensure the file path is within the destination directory
		if !strings.HasPrefix(filePath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		if file.FileInfo().IsDir() {
			// Create directories
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}

			continue
		}

		// Create the file
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directories for %s: %w", filePath, err)
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		defer outFile.Close()

		// Write the file content
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip file %s: %w", file.Name, err)
		}
		defer rc.Close()

		// Limit the size of data being copied to prevent decompression bomb attacks
		const maxFileSize = 100 * 1024 * 1024 // 100 MB
		if _, err := io.Copy(outFile, io.LimitReader(rc, maxFileSize)); err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filePath, err)
		}
	}

	return nil
}
