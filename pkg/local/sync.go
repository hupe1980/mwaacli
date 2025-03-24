package local

import (
	"context"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/s3"
)

type Syncer struct {
	s3Client *s3.Client
}

func NewSyncer(cfg *config.Config) *Syncer {
	return &Syncer{
		s3Client: s3.NewClient(cfg),
	}
}

type SyncRequirementsTXTInput struct {
	Bucket  *string // S3 bucket name
	Key     *string // S3 object key (e.g., "requirements.txt")
	Version *string // Optional S3 object version
}

func (s *Syncer) SyncRequirementsTXT(ctx context.Context, input *SyncRequirementsTXTInput) error {
	localPath := filepath.Join(DefaultClonePath, "requirements", "requirements.txt")

	return s.s3Client.DownloadFile(ctx, &s3.DownloadFileInput{
		Bucket:    input.Bucket,
		Key:       input.Key,
		Version:   input.Version,
		LocalPath: aws.String(localPath),
	})
}

type SyncStartupScriptInput struct {
	Bucket  *string // S3 bucket name
	Key     *string // S3 object key (e.g., "startup.sh")
	Version *string // Optional S3 object version
}

func (s *Syncer) SyncStartupScript(ctx context.Context, input *SyncStartupScriptInput) error {
	localPath := filepath.Join(DefaultClonePath, "startup_script", "startup.sh")

	return s.s3Client.DownloadFile(ctx, &s3.DownloadFileInput{
		Bucket:    input.Bucket,
		Key:       input.Key,
		Version:   input.Version,
		LocalPath: aws.String(localPath),
	})
}
