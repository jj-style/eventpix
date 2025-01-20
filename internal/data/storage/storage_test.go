package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jj-style/eventpix/internal/data/storage"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// a base test suite for storage implementations
// that runs the same test suite of storing and getting files
// in each store
func TestStorage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	stores := map[string]storage.Storage{
		"memory":     storage.NewMemStore(),
		"filesystem": storage.NewFilesystem(afero.NewMemMapFs(), "/store"),
		// TODO((jj): make this a testcontainer spun up to derive access key/secret key
		"s3": storage.NewS3Store(&storage.S3Config{
			Endpoint:  "http://127.0.0.1:9000",
			Region:    "us-east-1",
			AccessKey: "AAA",
			SecretKey: "BBB",
			Bucket:    "test",
		}),
	}

	for name, store := range stores {
		t.Run(name+" happy", func(t *testing.T) {
			t.Parallel()

			// store some data
			data := bytes.NewReader([]byte("data"))
			err := store.Store(ctx, t.Name(), data)
			require.NoError(t, err)

			// read it back out
			got, err := store.Get(ctx, t.Name())
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
			err := store.Store(ctx, t.Name(), data)
			require.Error(t, err)
		})
	}
}

type errReader struct{}

func (x *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("boom")
}
