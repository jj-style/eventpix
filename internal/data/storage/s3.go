package storage

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Store struct {
	s3     *minio.Client
	bucket string
}

func (s *s3Store) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	file, err := s.s3.GetObject(ctx, s.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		if err.Error() == "status code: 404 Not Found" {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return file, nil
}

func (s *s3Store) Store(ctx context.Context, name string, file io.Reader) (string, error) {
	id := uuid.NewString()
	_, err := s.s3.PutObject(ctx, s.bucket, id, file, -1, minio.PutObjectOptions{})
	return id, err
}

type S3Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	Endpoint  string
}

func NewS3Store(cfg *S3Config) Storage {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: true,
	})
	if err != nil {
		panic(err)
	}
	return &s3Store{s3: minioClient, bucket: cfg.Bucket}
}
