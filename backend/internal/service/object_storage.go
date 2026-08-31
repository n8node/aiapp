package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/n8node/aiapp/internal/model"
)

var (
	ErrStorageNotConfigured = fmt.Errorf("storage not configured")
	ErrObjectTooLarge       = errors.New("object too large")
)

type ObjectStorage struct {
	settings *StorageSettingsService
	corsMu   sync.Mutex
	corsOK   bool
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
	return &PresignedUpload{
		URL:     out.URL,
		Headers: browserUploadHeaders(contentType, out.SignedHeader),
	}, nil
}

func (o *ObjectStorage) EnsureBucketCORS(ctx context.Context) error {
	if o.settings == nil || o.settings.cfg == nil {
		return nil
	}
	o.corsMu.Lock()
	defer o.corsMu.Unlock()
	if o.corsOK {
		return nil
	}
	client, st, err := o.client(ctx)
	if err != nil {
		return err
	}
	origins := buildCORSOrigins(o.settings.cfg.CORSOrigins, o.settings.cfg.PublicAppURL)
	if err := putBucketCORS(ctx, client, st.Bucket, origins); err != nil {
		return err
	}
	o.corsOK = true
	return nil
}

func browserUploadHeaders(contentType string, signed http.Header) map[string]string {
	headers := map[string]string{}
	for k, vals := range signed {
		if len(vals) == 0 {
			continue
		}
		lk := strings.ToLower(k)
		switch {
		case lk == "host", lk == "content-length", lk == "content-type", lk == "connection",
			lk == "date", lk == "authorization", strings.HasPrefix(lk, "x-amz-checksum"),
			lk == "x-amz-sdk-checksum-algorithm", strings.HasPrefix(lk, "amz-sdk-"):
			continue
		}
		headers[k] = vals[0]
	}
	headers["Content-Type"] = contentType
	return headers
}

func (o *ObjectStorage) HashAndSize(ctx context.Context, s3Key string, max int64) (string, int64, error) {
	if max <= 0 {
		max = filesniff.MaxSizeBytes("", "a.pdf")
	}
	stream, err := o.OpenObject(ctx, s3Key, "")
	if err != nil {
		return "", 0, err
	}
	defer stream.Body.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(stream.Body, max+1))
	if err != nil {
		return "", 0, err
	}
	if n > max {
		return "", n, ErrObjectTooLarge
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func (o *ObjectStorage) DownloadToFile(ctx context.Context, s3Key, dest string, max int64) error {
	stream, err := o.OpenObject(ctx, s3Key, "")
	if err != nil {
		return err
	}
	defer stream.Body.Close()
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(stream.Body, max+1))
	if err != nil {
		return err
	}
	if n > max {
		return ErrObjectTooLarge
	}
	return nil
}

func (o *ObjectStorage) OpenObject(ctx context.Context, s3Key, rangeHeader string) (*ObjectStream, error) {
	client, st, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	if rangeHeader != "" && !validByteRange(rangeHeader) {
		rangeHeader = ""
	}
	in := &s3.GetObjectInput{
		Bucket: aws.String(st.Bucket),
		Key:    aws.String(s3Key),
	}
	if rangeHeader != "" {
		in.Range = aws.String(rangeHeader)
	}
	out, err := client.GetObject(ctx, in)
	if err != nil {
		return nil, err
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	cr := ""
	if out.ContentRange != nil {
		cr = *out.ContentRange
	}
	return &ObjectStream{
		Body:         out.Body,
		Size:         size,
		ContentRange: cr,
		Partial:      cr != "",
	}, nil
}

type ObjectStream struct {
	Body         io.ReadCloser
	Size         int64
	ContentRange string
	Partial      bool
}

func validByteRange(h string) bool {
	h = strings.TrimSpace(h)
	if h == "" {
		return false
	}
	if len(h) > 128 || strings.Contains(h, ",") {
		return false
	}
	return strings.HasPrefix(strings.ToLower(h), "bytes=")
}

func allowInlineDisposition(mimeType, name string) bool {
	m := strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(m, "image/"), strings.HasPrefix(m, "audio/"), strings.HasPrefix(m, "video/"), m == "application/pdf":
		return true
	}
	switch filesniff.Extension(name) {
	case "jpg", "jpeg", "png", "gif", "webp", "tif", "tiff", "mp4", "webm", "mp3", "wav", "m4a", "pdf":
		return true
	}
	return false
}

func ContentDispositionHeader(inline bool, mimeType, name string) string {
	kind := "attachment"
	if inline && allowInlineDisposition(mimeType, name) {
		kind = "inline"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "file"
	}
	ascii := sanitizeDownloadName(name)
	ascii = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, ascii)
	if ascii == "" {
		ascii = "file"
	}
	disp := kind + `; filename="` + ascii + `"`
	escaped := url.PathEscape(name)
	if escaped != ascii {
		disp += "; filename*=UTF-8''" + escaped
	}
	return disp
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

func (o *ObjectStorage) ReadObjectLimited(ctx context.Context, s3Key string, max int64) ([]byte, error) {
	if max <= 0 {
		max = filesniff.MaxSizeBytes("application/zip", "archive.zip")
	}
	stream, err := o.OpenObject(ctx, s3Key, "")
	if err != nil {
		return nil, err
	}
	defer stream.Body.Close()
	data, err := io.ReadAll(io.LimitReader(stream.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("object too large")
	}
	return data, nil
}

func (o *ObjectStorage) PutObject(ctx context.Context, s3Key, contentType string, body io.Reader, size int64) error {
	client, st, err := o.client(ctx)
	if err != nil {
		return err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	in := &s3.PutObjectInput{
		Bucket:      aws.String(st.Bucket),
		Key:         aws.String(s3Key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if size > 0 {
		in.ContentLength = aws.Int64(size)
	}
	_, err = client.PutObject(ctx, in)
	return err
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
	return deleteS3Object(ctx, client, st.Bucket, s3Key)
}

func (o *ObjectStorage) DeleteObjects(ctx context.Context, keys []string) error {
	keys = uniqueS3Keys(keys)
	if len(keys) == 0 {
		return nil
	}
	client, st, err := o.client(ctx)
	if err != nil {
		return err
	}
	for _, chunk := range chunkStrings(keys, 1000) {
		objs := make([]types.ObjectIdentifier, 0, len(chunk))
		for _, k := range chunk {
			objs = append(objs, types.ObjectIdentifier{Key: aws.String(k)})
		}
		out, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(st.Bucket),
			Delete: &types.Delete{Objects: objs, Quiet: aws.Bool(false)},
		})
		if err != nil {
			for _, k := range chunk {
				if derr := deleteS3Object(ctx, client, st.Bucket, k); derr != nil {
					return derr
				}
			}
			continue
		}
		for _, e := range out.Errors {
			code := ""
			if e.Code != nil {
				code = *e.Code
			}
			if code == "NoSuchKey" || code == "NotFound" {
				continue
			}
			key := ""
			if e.Key != nil {
				key = *e.Key
			}
			if key == "" {
				return fmt.Errorf("s3 delete failed")
			}
			if derr := deleteS3Object(ctx, client, st.Bucket, key); derr != nil {
				return derr
			}
		}
	}
	return nil
}

func deleteS3Object(ctx context.Context, client *s3.Client, bucket, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func uniqueS3Keys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func chunkStrings(items []string, size int) [][]string {
	if size <= 0 || len(items) == 0 {
		return nil
	}
	out := make([][]string, 0, (len(items)+size-1)/size)
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		out = append(out, items[i:end])
	}
	return out
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
