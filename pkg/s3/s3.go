package s3

import (
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
	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/util"
)

type Client struct {
	client *s3.Client
}

// NewClient creates a new Client with the provided AWS configuration.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		client: s3.NewFromConfig(cfg.AWSConfig),
	}
}

// DownloadFileInput defines the input parameters for the DownloadFile method.
type DownloadFileInput struct {
	Bucket    *string // S3 bucket name
	Key       *string // S3 object key (e.g., "requirements.txt")
	LocalPath *string // Local file path to overwrite (e.g., "requirements.txt")
	Version   *string // Optional S3 object version
}

// DownloadFile downloads the remote file from S3 and overwrites or creates the local file.
func (s *Client) DownloadFile(ctx context.Context, input *DownloadFileInput) error {
	if input.Bucket == nil || input.Key == nil || input.LocalPath == nil {
		return fmt.Errorf("bucket, key, and localPath are required")
	}

	// Prepare the GetObjectInput
	getObjectInput := &s3.GetObjectInput{
		Bucket:    input.Bucket,
		Key:       input.Key,
		VersionId: input.Version, // Optional version
	}

	// Get the object from S3
	output, err := s.client.GetObject(ctx, getObjectInput)
	if err != nil {
		return fmt.Errorf("failed to download file from S3: %w", err)
	}
	defer output.Body.Close()

	// Create or overwrite the local file
	localFile, err := os.Create(aws.ToString(input.LocalPath))
	if err != nil {
		return fmt.Errorf("failed to create local file '%s': %w", aws.ToString(input.LocalPath), err)
	}
	defer localFile.Close()

	// Write the S3 object content to the local file
	if _, err := io.Copy(localFile, output.Body); err != nil {
		return fmt.Errorf("failed to write to local file '%s': %w", aws.ToString(input.LocalPath), err)
	}

	return nil
}

// DownloadAndUnzipInput defines the input parameters for the DownloadAndUnzip method.
type DownloadAndUnzipInput struct {
	Bucket  *string // S3 bucket name
	Key     *string // S3 object key
	Version *string // Optional S3 object version
	DestDir *string // Destination directory for unzipping
}

// DownloadAndUnzip downloads a file from S3 and unzips it to the specified directory.
func (s *Client) DownloadAndUnzip(ctx context.Context, input *DownloadAndUnzipInput) error {
	if input.Bucket == nil || input.Key == nil || input.DestDir == nil {
		return fmt.Errorf("bucket, key, and destDir are required")
	}

	// Prepare the GetObjectInput
	getObjectInput := &s3.GetObjectInput{
		Bucket:    input.Bucket,
		Key:       input.Key,
		VersionId: input.Version, // Directly assign the version pointer
	}

	// Get the object from S3
	output, err := s.client.GetObject(ctx, getObjectInput)
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
	if err := util.Unzip(buf.Bytes(), aws.ToString(input.DestDir)); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}

	return nil
}

// SyncDirectoryInput defines the input parameters for the SyncDirectory method.
type SyncDirectoryInput struct {
	Bucket   *string // S3 bucket name
	Prefix   *string // S3 prefix for the directory
	LocalDir *string // Local directory to sync files to
}

// SyncDirectory synchronizes files from an S3 bucket to a local directory.
func (s *Client) SyncDirectory(ctx context.Context, input *SyncDirectoryInput) error {
	if input.Bucket == nil || input.Prefix == nil || input.LocalDir == nil {
		return fmt.Errorf("bucket, prefix, and localDir are required")
	}

	// List objects in the S3 bucket
	listOutput, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: input.Bucket,
		Prefix: input.Prefix,
	})
	if err != nil {
		return fmt.Errorf("failed to list objects in S3 bucket: %w", err)
	}

	// Create a map of S3 objects for comparison
	s3Objects := make(map[string]types.Object)

	for _, obj := range listOutput.Contents {
		relativePath := strings.TrimPrefix(aws.ToString(obj.Key), aws.ToString(input.Prefix))
		s3Objects[relativePath] = obj
	}

	// Ensure the local directory exists
	if err := os.MkdirAll(*input.LocalDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Download files from S3
	for relativePath, obj := range s3Objects {
		localFilePath := filepath.Join(*input.LocalDir, relativePath)

		// Create parent directories if necessary
		if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create directories for %s: %w", localFilePath, err)
		}

		// Download the file
		getOutput, err := s.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: input.Bucket,
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
		if err := os.Chtimes(localFilePath, aws.ToTime(obj.LastModified), aws.ToTime(obj.LastModified)); err != nil {
			return fmt.Errorf("failed to set timestamp for %s: %w", localFilePath, err)
		}
	}

	// Delete local files not present in the S3 bucket
	err = filepath.Walk(aws.ToString(input.LocalDir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the relative path of the local file
		relativePath, err := filepath.Rel(aws.ToString(input.LocalDir), path)
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
