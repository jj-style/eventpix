package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/jj-style/eventpix/internal/data/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	miniotc "github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/wait"
)

// a base test suite for storage implementations
// that runs the same test suite of storing and getting files
// in each store
func TestStorage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s3Conn, s3Username, s3Password, err := createMinioWithBucket(ctx, t)
	if err != nil {
		t.Fatalf("setting up s3: %v", err)
		return
	}

	ftpConn, ftpUsername, ftpPassword := createFtp(ctx, t)
	ftpStore, err := storage.NewFtpStore(&storage.FtpConfig{
		Address:   ftpConn + ":21",
		Username:  ftpUsername,
		Password:  ftpPassword,
		Directory: "/",
	})
	if err != nil {
		t.Fatalf("setting up ftp: %v", err)
		return
	}

	// TODO(jj) fix s3 test, strip http scheme, make `secure` configurable`

	stores := map[string]storage.Storage{
		"memory":     storage.NewMemStore(),
		"filesystem": storage.NewFilesystem(afero.NewMemMapFs(), "/store"),
		"s3": storage.NewS3Store(&storage.S3Config{
			Endpoint:  strings.TrimLeft(s3Conn, "https://"),
			Region:    "us-east-1",
			AccessKey: s3Username,
			SecretKey: s3Password,
			Bucket:    "test",
			Insecure:  true,
		}),
		"ftp": ftpStore,
	}

	// TODO(jj) remove when test fixed
	stores = map[string]storage.Storage{"s3": stores["s3"]}

	for name, store := range stores {
		t.Run(name+" happy", func(t *testing.T) {
			t.Parallel()
			// store some data
			data := bytes.NewReader([]byte("data"))
			id, err := store.Store(ctx, strings.ReplaceAll(t.Name(), "/", "_"), data)
			require.NoError(t, err)
			require.NotEmpty(t, id)

			// read it back out
			got, err := store.Get(ctx, id)
			require.NoError(t, err)
			gotData, _ := io.ReadAll(got)
			require.Equal(t, []byte("data"), gotData)
		})

		t.Run(name+" unhappy file not found", func(t *testing.T) {
			t.Parallel()
			// get file not stored
			got, err := store.Get(ctx, t.Name())
			require.Error(t, err)
			require.ErrorIs(t, err, storage.ErrFileNotFound)
			require.Nil(t, got)
		})

		t.Run(name+" unhappy reading data", func(t *testing.T) {
			t.Parallel()
			// store some data
			data := &errReader{}
			id, err := store.Store(ctx, t.Name(), data)
			require.Error(t, err)
			require.Empty(t, id)
		})
	}
}

type errReader struct{}

func (x *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("boom")
}

func createMinioWithBucket(ctx context.Context, t *testing.T) (string, string, string, error) {
	t.Helper()

	minioContainer, err := miniotc.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z")
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(minioContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	})
	if err != nil {
		return "", "", "", err
	}

	s3Conn, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get s3 connection string: %s", err)
		return "", "", "", err
	}

	minioClient, err := minio.New(s3Conn, &minio.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	if err != nil {
		return "", "", "", err
	}
	if err := minioClient.MakeBucket(ctx, "test", minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
		t.Fatalf("creating minio bucket: %v", err)
		return "", "", "", err
	}

	return "http://" + s3Conn, minioContainer.Username, minioContainer.Password, nil
}

func createFtp(ctx context.Context, t *testing.T) (string, string, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "garethflowers/ftp-server",
		ExposedPorts: []string{"20/tcp", "21/tcp"},
		Env: map[string]string{
			"FTP_USER": "user",
			"FTP_PASS": "123",
		},
		WaitingFor: wait.ForLog("chpasswd: password for 'user' changed"),
	}
	ftp, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, ftp)

	ip, err := ftp.ContainerIP(ctx)
	require.NoError(t, err)

	return ip, "user", "123"
}
