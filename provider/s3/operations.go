package s3provider

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/yunerou/niarb/shared/aerror"
)

func (c *client) Upload(ctx context.Context, key string, body io.Reader, contentType string) aerror.AError {
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
	}
	return nil
}

func (c *client) Download(ctx context.Context, key string) (io.ReadCloser, aerror.AError) {
	out, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, c.mapError(ctx, err)
	}
	return out.Body, nil
}

func (c *client) Delete(ctx context.Context, key string) aerror.AError {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return c.mapError(ctx, err)
	}
	return nil
}

func (c *client) List(ctx context.Context, prefix string) ([]string, aerror.AError) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(c.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
		}
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
	}
	return keys, nil
}

func (c *client) GetFileMetadata(ctx context.Context, s3FilePath string) (map[string]any, aerror.AError) {
	out, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3FilePath),
	})
	if err != nil {
		return nil, c.mapError(ctx, err)
	}

	metadata := make(map[string]any)
	if out.ContentType != nil {
		metadata["content_type"] = *out.ContentType
	}
	if out.ContentLength != nil {
		metadata["content_length"] = *out.ContentLength
	}
	if out.LastModified != nil {
		metadata["last_modified"] = *out.LastModified
	}
	if out.ETag != nil {
		metadata["etag"] = *out.ETag
	}
	for k, v := range out.Metadata {
		metadata[k] = v
	}
	return metadata, nil
}

func (c *client) mapError(ctx context.Context, err error) aerror.ASingleError {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return aerror.New(ctx, aerror.RecordNotFound, err)
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return aerror.New(ctx, aerror.RecordNotFound, err)
	}
	return aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
}
