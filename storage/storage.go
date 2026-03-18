package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type FileType string

const (
	FileTypeImage    FileType = "image"
	FileTypeVideo    FileType = "video"
	FileTypeAudio    FileType = "audio"
	FileTypeDocument FileType = "document"
)

// UploadedFile holds metadata about an uploaded file
type UploadedFile struct {
	URL      string
	Name     string
	Size     int64
	MimeType string
	FileType FileType
}

// Client is the interface for file storage operations
type Client interface {
	Upload(ctx context.Context, folder string, file *multipart.FileHeader) (*UploadedFile, error)
	Delete(ctx context.Context, url string) error
}

// NewClient returns a local or S3 client based on STORAGE_DRIVER env var.
// Defaults to "local" if not set.
func NewClient(ctx context.Context) (Client, error) {
	driver := strings.TrimSpace(os.Getenv("STORAGE_DRIVER"))
	if driver == "" {
		driver = "local"
	}

	switch driver {
	case "local":
		return newLocalClient(), nil
	case "s3":
		return newS3Client(ctx)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %q (use local or s3)", driver)
	}
}

// =============================================================================
// Local driver
// =============================================================================

type localClient struct {
	uploadDir string
	baseURL   string
}

func newLocalClient() *localClient {
	uploadDir := "uploads"
	baseURL := os.Getenv("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &localClient{uploadDir: uploadDir, baseURL: strings.TrimSpace(baseURL)}
}

func (c *localClient) Upload(_ context.Context, folder string, file *multipart.FileHeader) (*UploadedFile, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	mimeType := file.Header.Get("Content-Type")
	fileType := detectFileType(mimeType)

	ext := filepath.Ext(file.Filename)
	datePath := time.Now().Format("2006/01/02")
	relPath := filepath.Join(folder, datePath, uuid.New().String()+ext)
	absPath := filepath.Join(c.uploadDir, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	urlPath := strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
	url := fmt.Sprintf("%s/uploads/%s", strings.TrimRight(c.baseURL, "/"), urlPath)

	return &UploadedFile{
		URL:      url,
		Name:     file.Filename,
		Size:     file.Size,
		MimeType: mimeType,
		FileType: fileType,
	}, nil
}

func (c *localClient) Delete(_ context.Context, url string) error {
	prefix := strings.TrimRight(c.baseURL, "/") + "/uploads/"
	relPath := strings.TrimPrefix(url, prefix)
	absPath := filepath.Join(c.uploadDir, relPath)
	return os.Remove(absPath)
}

// =============================================================================
// S3 driver
// =============================================================================

type s3Client struct {
	client  *s3.Client
	bucket  string
	baseURL string
}

// newS3Client creates an S3-compatible storage client.
//
// Required env vars:
//
//	STORAGE_ENDPOINT   — e.g. https://s3.amazonaws.com or R2/MinIO endpoint
//	STORAGE_REGION     — e.g. us-east-1
//	STORAGE_BUCKET     — bucket name
//	STORAGE_KEY_ID     — access key ID
//	STORAGE_SECRET     — secret access key
//	STORAGE_BASE_URL   — public base URL e.g. https://cdn.yourdomain.com
func newS3Client(ctx context.Context) (*s3Client, error) {
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	region := os.Getenv("STORAGE_REGION")
	bucket := os.Getenv("STORAGE_BUCKET")
	keyID := os.Getenv("STORAGE_KEY_ID")
	secret := os.Getenv("STORAGE_SECRET")
	baseURL := os.Getenv("STORAGE_BASE_URL")

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(keyID, secret, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})

	return &s3Client{client: client, bucket: bucket, baseURL: baseURL}, nil
}

func (c *s3Client) Upload(ctx context.Context, folder string, file *multipart.FileHeader) (*UploadedFile, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	mimeType := file.Header.Get("Content-Type")
	fileType := detectFileType(mimeType)

	ext := filepath.Ext(file.Filename)
	key := fmt.Sprintf("%s/%s/%s%s",
		folder,
		time.Now().Format("2006/01/02"),
		uuid.New().String(),
		ext,
	)

	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        src,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	url := fmt.Sprintf("%s/%s", strings.TrimRight(c.baseURL, "/"), key)

	return &UploadedFile{
		URL:      url,
		Name:     file.Filename,
		Size:     file.Size,
		MimeType: mimeType,
		FileType: fileType,
	}, nil
}

func (c *s3Client) Delete(ctx context.Context, url string) error {
	key := strings.TrimPrefix(url, strings.TrimRight(c.baseURL, "/")+"/")
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

// =============================================================================
// Shared helpers
// =============================================================================

func detectFileType(mimeType string) FileType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return FileTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return FileTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return FileTypeAudio
	default:
		return FileTypeDocument
	}
}
