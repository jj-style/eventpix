package service_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jj-style/eventpix/internal/data/db"
	mdb "github.com/jj-style/eventpix/internal/data/db/mocks"
	mstorage "github.com/jj-style/eventpix/internal/data/storage/mocks"
	"github.com/jj-style/eventpix/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStorageService(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mdb := mdb.NewMockDB(t)
	mstorage := mstorage.NewMockStorage(t)
	svc := service.NewStorageService(mdb, zap.NewNop())

	t.Run("happy get picture", func(t *testing.T) {
		t.Parallel()
		is := require.New(t)

		filename := t.Name()

		mdb.EXPECT().
			GetFileInfo(ctx, filename).
			Return(&db.FileInfo{
				EventID: 1,
				Name:    t.Name(),
			}, nil)

		mdb.EXPECT().
			GetEvent(ctx, uint64(1)).
			Return(&db.Event{
				Storage: mstorage,
			}, nil)

		mstorage.EXPECT().
			Get(ctx, filename).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), nil)

		gotName, gotData, err := svc.GetPicture(ctx, filename)

		is.NoError(err)
		is.Equal(filename, gotName)
		is.Equal([]byte("data"), gotData)
	})

	t.Run("unhappy get picture", func(t *testing.T) {
		t.Parallel()
		is := require.New(t)

		filename := t.Name()

		mdb.EXPECT().
			GetFileInfo(ctx, filename).
			Return(nil, errors.New("boom"))

		gotName, gotData, err := svc.GetPicture(ctx, filename)

		is.Error(err)
		is.Equal("", gotName)
		is.Equal([]byte(nil), gotData)
	})

	t.Run("happy get thumbnail", func(t *testing.T) {
		t.Parallel()
		is := require.New(t)

		filename := t.Name()

		mdb.EXPECT().
			GetThumbnailInfo(ctx, filename).
			Return(&db.ThumbnailInfo{
				EventID: 1,
				Name:    t.Name(),
			}, nil)

		mdb.EXPECT().
			GetEvent(ctx, uint64(1)).
			Return(&db.Event{
				Storage: mstorage,
			}, nil)

		mstorage.EXPECT().
			Get(ctx, filename).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), nil)

		gotName, gotData, err := svc.GetThumbnail(ctx, filename)

		is.NoError(err)
		is.Equal(filename, gotName)
		is.Equal([]byte("data"), gotData)
	})

	t.Run("unhappy get thumbnail", func(t *testing.T) {
		t.Parallel()
		is := require.New(t)

		filename := t.Name()

		mdb.EXPECT().
			GetThumbnailInfo(ctx, filename).
			Return(nil, errors.New("boom"))

		gotName, gotData, err := svc.GetThumbnail(ctx, filename)

		is.Error(err)
		is.Equal("", gotName)
		is.Equal([]byte(nil), gotData)
	})

}
