package storage

import (
	"bytes"
	"context"
	"io"

	"github.com/rhnvrm/simples3"
)

type s3Store struct {
	s3     *simples3.S3
	bucket string
}

func (s *s3Store) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	file, err := s.s3.FileDownload(simples3.DownloadInput{
		Bucket:    s.bucket,
		ObjectKey: name,
	})
	if err != nil {
		if err.Error() == "status code: 404 Not Found" {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return file, nil
}

func (s *s3Store) Store(ctx context.Context, name string, file io.Reader) error {
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	_, err = s.s3.FileUpload(simples3.UploadInput{
		Bucket:      s.bucket,
		ObjectKey:   name,
		ContentType: "text/plain",
		FileName:    name,
		Body:        bytes.NewReader(data),
	})
	return err
}

type S3Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	Endpoint  string
}

func NewS3Store(cfg *S3Config) Storage {
	s3 := simples3.New(cfg.Region, cfg.AccessKey, cfg.SecretKey)
	if cfg.Endpoint != "" {
		s3.SetEndpoint(cfg.Endpoint)
	}
	return &s3Store{s3: s3, bucket: cfg.Bucket}
}
