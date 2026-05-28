package s3provider

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/yunerou/niarb/shared/aerror"
)

func (c *client) PresignGetObject(
	ctx context.Context,
	s3FilePath string,
	lifetimeSecs int64,
) (string, aerror.AError) {
	out, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3FilePath),
	}, s3.WithPresignExpires(time.Duration(lifetimeSecs)*time.Second))
	if err != nil {
		return "", aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
	}
	return out.URL, nil
}

func (c *client) PresignPostObject(
	ctx context.Context,
	s3FilePath string,
	lifetimeSecs int64,
	opts *PresignPostObjectOptions,
) (string, map[string]string, aerror.AError) {

	// build conditions
	conditions := []interface{}{}

	if opts.MaxContentLength > 0 {
		conditions = append(
			conditions,
			[]interface{}{
				"content-length-range",
				opts.MinContentLength,
				opts.MaxContentLength,
			},
		)
	}

	if opts.ContentTypePrefix != nil {
		conditions = append(
			conditions,
			[]interface{}{"starts-with", "$Content-Type", opts.ContentTypePrefix},
		)
	}

	if opts.ContentType != nil {
		conditions = append(conditions, []interface{}{"eq", "$Content-Type", opts.ContentType})
	}

	if opts.Metadata != nil {
		for k, v := range opts.Metadata {
			// Metadata in post always start with "x-amz-meta-" prefix condition
			conditions = append(conditions, []interface{}{"eq", "$x-amz-meta-" + k, v})
		}
	}

	request, err := c.presignClient.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3FilePath),
	}, func(opts *s3.PresignPostOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
		opts.Conditions = conditions
	})
	if err != nil {
		slog.ErrorContext(ctx, "PresignPutObject error #1",
			slog.String("Bucket", c.bucket),
			slog.String("s3ObjectKey", s3FilePath),
			slog.Any("err", err),
		)
		return "", nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
	}

	return request.URL, request.Values, nil
}
