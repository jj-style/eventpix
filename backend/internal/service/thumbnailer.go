package service

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	eventsv1 "github.com/jj-style/eventpix/backend/gen/events/v1"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/pkg/thumber"
	"go.uber.org/zap"
)

type Thumbnailer struct {
	db         db.DB
	thumber    thumber.Thumber
	subscriber message.Subscriber
	log        *zap.SugaredLogger
}

func NewThumbnailer(
	db db.DB,
	thumber thumber.Thumber,
	subscriber message.Subscriber,
	log *zap.Logger) (*Thumbnailer, error) {
	return &Thumbnailer{
		db:         db,
		thumber:    thumber,
		subscriber: subscriber,
		log:        log.Sugar(),
	}, nil
}

func (t *Thumbnailer) Start(ctx context.Context, workers int) error {
	messages, err := t.subscriber.Subscribe(ctx, "new-photo")
	if err != nil {
		t.log.Error("subscribing to topic", zap.Error(err))
		return err
	}

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			t.log.Infof("running thumbnailer %d", i)
			for msg := range messages {
				go func() {
					if err := t.Thumb(ctx, msg); err != nil {
						t.log.Errorf("processing thumbnail msg(%s): %v", msg.UUID, err)
						msg.Nack()
						return
					}
					// we need to Acknowledge that we received and processed the message,
					// otherwise, it will be resent over and over again.
					msg.Ack()
					t.log.Infof("message %s handled by worker %d", msg.UUID, i)
				}()
			}
			t.log.Info("ending thumbnailer")
		}()
	}
	return nil
}

func (t *Thumbnailer) Thumb(ctx context.Context, msg *message.Message) error {
	var req eventsv1.NewPhoto
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		t.log.Errorf("unmarshaling new photo message: %v", err)
		return err
	}

	evt, err := t.db.GetEvent(ctx, req.GetEventId())
	if err != nil {
		t.log.Errorf("getting photo's event: %v", err)
		return err
	}

	fi, err := t.db.GetFileInfo(ctx, req.GetFileId())
	if err != nil {
		t.log.Errorf("getting photo info: %v", err)
		return err
	}

	buf, err := evt.Storage.Get(ctx, fi.Name)
	if err != nil {
		t.log.Errorf("getting photo: %v", err)
		return err
	}
	defer buf.Close()

	t.log.Info("creating thumbnail")
	thumnail, err := t.thumber.Thumb(buf)
	if err != nil {
		t.log.Errorf("creating thumbnail: %v", err)
		return err
	}

	if err := t.db.AddFileInfo(ctx, &db.FileInfo{
		ID:        uuid.NewString(),
		Name:      "thumb_" + fi.Name,
		EventID:   uint(req.GetEventId()),
		Thumbnail: true,
	}); err != nil {
		t.log.Errorf("saving thumbnail info to db: %v", err)
		return err
	}

	if err := evt.Storage.Store(ctx, "thumb_"+fi.Name, thumnail); err != nil {
		t.log.Errorf("storing thumbnail: %v", err)
		return err
	}

	return nil
}
