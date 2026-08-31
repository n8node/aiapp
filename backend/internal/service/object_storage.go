package service

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n8node/aiapp/internal/model"
)

var (
	ErrStorageNotConfigured = fmt.Errorf("storage not configured")
)

type ObjectStorage struct {
	settings *StorageSettingsService
}

func NewObjectStorage(settings *StorageSettingsService) *ObjectStorage {
	return &ObjectStorage{settings: settings}
}

func (o *ObjectStorage) client(ctx context.Context) (*s3.Client, model.StorageSettings, error) {
	st, err := o.settings.GetEffective(ctx)
	if err != nil {
		return nil, model.StorageSettings{}, err
	}
	if !IsStorageEnabled(st) {
		return nil, st, ErrStorageNotConfigured
	}
	c, err := newS3Client(st)
	if err != nil {
		return nil, st, err
	}
	return c, st, nil
}

type PresignedUpload struct {
	URL     string
	Headers map[string]string
}

func (o *ObjectStorage) PresignPut(ctx context.Context, s3Key, contentType string, expires time.Duration) (*PresignedUpload, error) {
	client, st, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	presign := s3.NewPresignClient(client)
	out, err := presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(st.Bucket),
		Key:         aws.String(s3Key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"Content-Type": contentType}
	for k, vals := range out.SignedHeader {
		if len(vals) > 0 {
			headers[k] = vals[0]
		}
	}
	return &PresignedUpload{URL: out.URL, Headers: headers}, nil
}

func (o *ObjectStorage) PresignGet(ctx context.Context, s3Key, filename string, inline bool, expires time.Duration) (string, error) {
	client, st, err := o.client(ctx)
	if err != nil {
		return "", err
	}
	if expires <= 0 {
		expires = 5 * time.Minute
	}
	presign := s3.NewPresignClient(client)
	input := &s3.GetObjectInput{
		Bucket: aws.String(st.Bucket),
		Key:    aws.String(s3Key),
	}
	if inline {
		input.ResponseContentDisposition = aws.String("inline")
	} else if filename != "" {
		input.ResponseContentDisposition = aws.String(`attachment; filename="` + sanitizeDownloadName(filename) + `"`)
	}
	out, err := presign.PresignGetObject(ctx, input, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

type HeadObjectResult struct {
	Size        int64
	ContentType string
}

func (o *ObjectStorage) HeadObject(ctx context.Context, s3Key string) (*HeadObjectResult, error) {
	client, st, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	out, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(st.Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return nil, err
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return &HeadObjectResult{Size: size, ContentType: ct}, nil
}

func (o *ObjectStorage) ReadHead(ctx context.Context, s3Key string, n int64) ([]byte, error) {
	if n <= 0 {
		n = 512
	}
	client, st, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(st.Bucket),
		Key:    aws.String(s3Key),
		Range:  aws.String(fmt.Sprintf("bytes=0-%d", n-1)),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(io.LimitReader(out.Body, n))
}

func (o *ObjectStorage) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	client, st, err := o.client(ctx)
	if err != nil {
		return err
	}
	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(st.Bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(encodeCopySource(st.Bucket, srcKey)),
	})
	return err
}

func (o *ObjectStorage) DeleteObject(ctx context.Context, s3Key string) error {
	client, st, err := o.client(ctx)
	if err != nil {
		return err
	}
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(st.Bucket),
		Key:    aws.String(s3Key),
	})
	return err
}

func encodeCopySource(bucket, key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return url.PathEscape(bucket) + "/" + strings.Join(parts, "/")
}

func sanitizeDownloadName(name string) string {
	name = strings.ReplaceAll(name, `"`, "'")
	name = strings.ReplaceAll(name, "\n", " ")
	name = strings.ReplaceAll(name, "\r", " ")
	if len(name) > 180 {
		return name[:180]
	}
	return name
}
