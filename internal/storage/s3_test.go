package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

func TestS3Upload(t *testing.T) {
	ctx := context.Background()

	minioContainer, err := minio.Run(ctx, "minio/minio:latest")
	require.NoError(t, err)
	defer minioContainer.Terminate(ctx)

	endpoint, err := minioContainer.ConnectionString(ctx)
	require.NoError(t, err)

	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "http://" + endpoint
	}

	s3Storage, err := NewS3Storage(ctx, endpoint, "minioadmin", "minioadmin")
	require.NoError(t, err)

	bucketName := "test-bucket"
	_, err = s3Storage.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err)

	content := "hello world"
	key := "test.txt"
	url, err := s3Storage.UploadFile(ctx, bucketName, key, strings.NewReader(content))
	require.NoError(t, err)
	require.Contains(t, url, bucketName)
	require.Contains(t, url, key)
}
