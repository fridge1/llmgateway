package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

// TOSConfig holds TOS client configuration.
type TOSConfig struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	URLPrefix string
}

// TOSClient wraps the Volcengine TOS SDK client.
type TOSClient struct {
	client    *tos.ClientV2
	bucket    string
	urlPrefix string
}

// NewTOSClient creates a new TOS client.
func NewTOSClient(cfg TOSConfig) (*TOSClient, error) {
	client, err := tos.NewClientV2(cfg.Endpoint, tos.WithRegion(cfg.Region), tos.WithCredentials(tos.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey)))
	if err != nil {
		return nil, fmt.Errorf("failed to create TOS client: %w", err)
	}

	// 自动创建存储桶（如果不存在）
	if err := ensureBucketExists(client, cfg.Bucket, cfg.Region); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return &TOSClient{
		client:    client,
		bucket:    cfg.Bucket,
		urlPrefix: cfg.URLPrefix,
	}, nil
}

// ensureBucketExists checks if bucket exists, creates it if not, and sets public read ACL.
func ensureBucketExists(client *tos.ClientV2, bucket, region string) error {
	ctx := context.Background()

	// 检查存储桶是否存在
	_, err := client.HeadBucket(ctx, &tos.HeadBucketInput{Bucket: bucket})
	if err == nil {
		// 存储桶已存在，检查并设置 ACL
		return ensureBucketACL(client, bucket)
	}

	// 如果是其他错误（非 404），返回错误
	if !isBucketNotFoundError(err) {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	// 存储桶不存在，创建它
	_, err = client.CreateBucketV2(ctx, &tos.CreateBucketV2Input{
		Bucket: bucket,
		ACL:    tos.ACLPublicRead,
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// ensureBucketACL ensures the bucket has public read ACL.
func ensureBucketACL(client *tos.ClientV2, bucket string) error {
	ctx := context.Background()

	// 设置存储桶为公共读
	_, err := client.PutBucketACL(ctx, &tos.PutBucketACLInput{
		Bucket:    bucket,
		ACLType:   tos.ACLPublicRead,
	})
	if err != nil {
		return fmt.Errorf("failed to set bucket ACL: %w", err)
	}

	return nil
}

// isBucketNotFoundError checks if the error is a bucket not found error.
func isBucketNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// TOS SDK 返回的 404 错误
	errStr := err.Error()
	return strings.Contains(errStr, "StatusCode=404") ||
		strings.Contains(errStr, "NoSuchBucket") ||
		strings.Contains(errStr, "NotFound")
}

// UploadImage uploads image data to TOS and returns the public URL.
func (c *TOSClient) UploadImage(ctx context.Context, data []byte, userID string) (string, error) {
	filename := c.generateFilename(userID)
	key := fmt.Sprintf("images/%s/%s", userID, filename)

	input := &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:      c.bucket,
			Key:         key,
			ACL:         tos.ACLPublicRead, // 设置对象为公共读
			ContentType: "image/png",
		},
		Content: bytes.NewReader(data),
	}

	_, err := c.client.PutObjectV2(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to TOS: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.urlPrefix, key)
	return url, nil
}

// DownloadImageFromURL downloads an image from a URL and returns the data.
func (c *TOSClient) DownloadImageFromURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 验证文件大小（最大 10MB）
	if len(data) > 10*1024*1024 {
		return nil, fmt.Errorf("image too large: %d bytes", len(data))
	}

	return data, nil
}

// DetermineImageSize determines the billing tier based on image dimensions.
// Returns "1K", "2K", or "4K".
func (c *TOSClient) DetermineImageSize(width, height int) string {
	if width > 1792 || height > 1792 {
		return "4K"
	}
	if width > 1024 || height > 1024 {
		return "2K"
	}
	return "1K"
}

// IsTOSURL reports whether the given URL belongs to this TOS bucket.
func (c *TOSClient) IsTOSURL(url string) bool {
	return c.urlPrefix != "" && strings.HasPrefix(url, c.urlPrefix)
}

// KeyFromURL extracts the object key from a public URL stored on this bucket.
// Returns empty string if the URL doesn't belong to this bucket.
func (c *TOSClient) KeyFromURL(url string) string {
	if !c.IsTOSURL(url) {
		return ""
	}
	return strings.TrimPrefix(url, c.urlPrefix+"/")
}

// DeleteObject removes a single object from the bucket. Treats not-found as success.
func (c *TOSClient) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	_, err := c.client.DeleteObjectV2(ctx, &tos.DeleteObjectV2Input{
		Bucket: c.bucket,
		Key:    key,
	})
	if err != nil && !isObjectNotFoundError(err) {
		return fmt.Errorf("failed to delete TOS object %q: %w", key, err)
	}
	return nil
}

// isObjectNotFoundError checks if the error indicates the object does not exist.
func isObjectNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "StatusCode=404") ||
		strings.Contains(errStr, "NoSuchKey") ||
		strings.Contains(errStr, "NotFound")
}

// UploadFile uploads arbitrary file data to TOS with private ACL and returns the object key.
func (c *TOSClient) UploadFile(ctx context.Context, data []byte, key string, contentType string) (string, error) {
	input := &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:      c.bucket,
			Key:         key,
			ACL:         tos.ACLPrivate,
			ContentType: contentType,
		},
		Content: bytes.NewReader(data),
	}

	_, err := c.client.PutObjectV2(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to TOS: %w", err)
	}
	return key, nil
}

// PreSignedURL generates a pre-signed GET URL for the given key, valid for the specified duration.
func (c *TOSClient) PreSignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	output, err := c.client.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: http.MethodGet,
		Bucket:     c.bucket,
		Key:        key,
		Expires:    int64(expires.Seconds()),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate pre-signed URL: %w", err)
	}
	return output.SignedUrl, nil
}

// generateFilename generates a unique filename with timestamp and random string.
func (c *TOSClient) generateFilename(userID string) string {
	timestamp := time.Now().Unix()
	random := randomString(8)
	return fmt.Sprintf("%d_%s.png", timestamp, random)
}

// randomString generates a random alphanumeric string of given length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
