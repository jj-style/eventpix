package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/jj-style/eventpix/internal/config"
	db "github.com/jj-style/eventpix/internal/data/db"
	mockdb "github.com/jj-style/eventpix/internal/data/db/mocks"
	mockStorage "github.com/jj-style/eventpix/internal/data/storage/mocks"
	eventsv1 "github.com/jj-style/eventpix/internal/gen/events/v1"
	mockImagor "github.com/jj-style/eventpix/internal/pkg/imagor/mocks"
	"github.com/jj-style/eventpix/internal/service"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func natsServer(t *testing.T) *server.Server {
	t.Helper()

	opts := &server.Options{}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)

	return ns
}

func TestThumbnailer(t *testing.T) {
	is := require.New(t)

	ns := natsServer(t)
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Error("nats not ready for connection")
		t.FailNow()
	}
	t.Cleanup(ns.Shutdown)
	nc, err := nats.Connect(ns.ClientURL())
	is.NoError(err)
	defer nc.Drain()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	// arrange
	mdb := mockdb.NewMockDB(t)
	mstorage := mockStorage.NewMockStorage(t)
	mimg := mockImagor.NewMockImagor(t)

	thumber, err := service.NewThumbnailer(
		&config.Config{Server: &config.Server{ServerUrl: "http://example.com"}},
		mdb,
		mimg,
		nc,
		zap.NewNop())
	is.NoError(err)

	go thumber.Start(ctx)

	// == setp mocks == //
	// get event with events mock storage
	mdb.EXPECT().GetEvent(mock.Anything, uint64(1)).Return(&db.Event{
		Storage: mstorage,
	}, nil)

	// retrieve the file info
	mdb.EXPECT().
		GetFileInfo(mock.Anything, "abc").
		Return(&db.FileInfo{ID: "abc", Name: "file.jpg"}, nil)

	// retrieve the original file
	// create thumbnail from original
	mockThumbnailData := io.NopCloser(bytes.NewReader([]byte("thumbnail file")))
	mimg.EXPECT().
		ThumbImage("http://example.com/storage/picture/abc").
		Return(mockThumbnailData, nil)

	// store thumbnail info with event
	mdb.EXPECT().
		AddThumbnailInfo(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, ti *db.ThumbnailInfo) error {
			is.Equal("thumb_file.jpg", ti.Name)
			is.Equal("abc", ti.FileInfoID) // foreign key link to main file
			is.Equal(uint(1), ti.EventID)
			return nil
		})

	// store thumbnail
	mstorage.EXPECT().
		Store(mock.Anything, "thumb_file.jpg", mockThumbnailData).
		Return(nil)

	// act
	msg := &eventsv1.NewMedia{FileId: "abc", EventId: uint64(1), Type: eventsv1.NewMedia_IMAGE}
	is.NoError(nc.Publish("new-photo", lo.Must(json.Marshal(msg))))

	<-ctx.Done()
}
