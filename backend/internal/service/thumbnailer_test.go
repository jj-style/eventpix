package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	eventsv1 "github.com/jj-style/eventpix/backend/gen/events/v1"
	db "github.com/jj-style/eventpix/backend/internal/data/db"
	mockdb "github.com/jj-style/eventpix/backend/internal/data/db/mocks"
	mockStorage "github.com/jj-style/eventpix/backend/internal/data/storage/mocks"
	mockThumber "github.com/jj-style/eventpix/backend/internal/pkg/thumber/mocks"
	"github.com/jj-style/eventpix/backend/internal/service"
	test_utils "github.com/jj-style/eventpix/backend/internal/tests"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func TestThumbnailer(t *testing.T) {
	is := require.New(t)

	// arrange
	mdb := mockdb.NewMockDB(t)
	mstorage := mockStorage.NewMockStorage(t)
	mthumb := mockThumber.NewMockThumber(t)

	nc, cleanup, err := test_utils.NewInProcessNATSServer(t)
	is.NoError(err)
	t.Cleanup(cleanup)

	thumber, err := service.NewThumbnailer(
		mdb,
		nc,
		mthumb,
		zap.NewNop())
	is.NoError(err)

	is.NoError(thumber.Start(context.Background()))
	t.Cleanup(func() { is.NoError(thumber.Stop()) })

	// == setp mocks == //
	// get event with events mock storage
	mdb.EXPECT().GetEvent(mock.Anything, uint64(1)).Return(&db.Event{
		Storage: mstorage,
	}, nil)

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

	// store thumbnail
	mstorage.EXPECT().
		Store(mock.Anything, "thumb_file.jpg", mockThumbnailData).
		Return(nil)

	// act
	msg := &eventsv1.NewPhoto{Filename: "file.jpg", EventId: 1}
	is.NoError(nc.Publish("new-photo", lo.Must(proto.Marshal(msg))))

	// TODO - this better
	// ideally can shutdown thumber with a timeout to finish processing messages?
	time.Sleep(time.Millisecond * 50)
}
