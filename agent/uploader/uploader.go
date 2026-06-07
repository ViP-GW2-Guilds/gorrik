package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
)

// Uploader uploads raw EVTC files to Cloudflare R2.
type Uploader struct {
	client *s3.Client
	bucket string
	dryRun bool
}

// New creates an Uploader from cfg. If dryRun is true, uploads are logged but
// not executed — useful for testing the pipeline before R2 is provisioned.
func New(cfg *config.Config, dryRun bool) (*Uploader, error) {
	if dryRun {
		return &Uploader{dryRun: true, bucket: cfg.Storage.R2Bucket}, nil
	}
	if cfg.Storage.R2AccountID == "" {
		return nil, fmt.Errorf("storage.r2_account_id is not configured")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.Storage.R2AccountID)

	r2Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       "auto",
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.Storage.R2AccessKeyID,
			cfg.Storage.R2SecretAccessKey,
			"",
		),
	})

	return &Uploader{
		client: r2Client,
		bucket: cfg.Storage.R2Bucket,
		dryRun: false,
	}, nil
}

// Upload sends the file at localPath to R2 and returns the object key.
// The key is simply the base filename; the caller constructs the full URL.
func (u *Uploader) Upload(ctx context.Context, localPath string) (key string, err error) {
	key = filepath.Base(localPath)

	if u.dryRun {
		fmt.Printf("[dry-run] would upload %s → r2://%s/%s\n", localPath, u.bucket, key)
		return key, nil
	}

	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return "", fmt.Errorf("upload %s: %w", key, err)
	}

	return key, nil
}

// FileURL constructs the R2 URL for a given object key and account ID.
func FileURL(accountID, bucket, key string) string {
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", accountID, bucket, key)
}
