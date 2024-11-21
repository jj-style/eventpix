package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	eventsv1 "github.com/jj-style/eventpix/backend/gen/events/v1"
	"github.com/jj-style/eventpix/backend/internal/config"
	db "github.com/jj-style/eventpix/backend/internal/data/db"
	mockdb "github.com/jj-style/eventpix/backend/internal/data/db/mocks"
	mockStorage "github.com/jj-style/eventpix/backend/internal/data/storage/mocks"
	"github.com/jj-style/eventpix/backend/internal/pkg/pubsub"
	mockThumber "github.com/jj-style/eventpix/backend/internal/pkg/thumber/mocks"
	"github.com/jj-style/eventpix/backend/internal/service"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestThumbnailer(t *testing.T) {
	is := require.New(t)

	// arrange
	mdb := mockdb.NewMockDB(t)
	mstorage := mockStorage.NewMockStorage(t)
	mthumb := mockThumber.NewMockThumber(t)

	publisher, cleanup, err := pubsub.NewPublisher(&config.PubSub{Mode: "memory"})
	is.NoError(err)
	t.Cleanup(cleanup)
	subscriber, err := pubsub.NewMemorySubscriberFromPublisher(publisher)
	is.NoError(err)

	thumber, err := service.NewThumbnailer(
		mdb,
		mthumb,
		subscriber,
		zap.NewNop())
	is.NoError(err)

	is.NoError(thumber.Start(context.Background(), 1))

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
	mockFileData := io.NopCloser(bytes.NewReader([]byte("original file")))
	mstorage.EXPECT().
		Get(mock.Anything, "file.jpg").
		Return(mockFileData, nil)

	// create thumbnail from original
	mockThumbnailData := bytes.NewReader([]byte("thumbnail file"))
	mthumb.EXPECT().
		Thumb(mockFileData).
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
	msg := &eventsv1.NewPhoto{FileId: "abc", EventId: 1}
	is.NoError(publisher.Publish("new-photo", message.NewMessage(watermill.NewShortUUID(), lo.Must(json.Marshal(msg)))))

	// TODO - this better
	// ideally can shutdown thumber with a timeout to finish processing messages?
	time.Sleep(time.Millisecond * 50)
}
