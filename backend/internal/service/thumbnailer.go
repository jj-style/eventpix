package service

import (
	"context"

	"github.com/google/uuid"
	eventsv1 "github.com/jj-style/eventpix/backend/gen/events/v1"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/pkg/thumber"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Thumbnailer struct {
	db      db.DB
	nc      *nats.Conn
	thumber thumber.Thumber
	log     *zap.SugaredLogger
	conn    *nats.Subscription
}

func NewThumbnailer(
	db db.DB,
	nc *nats.Conn,
	thumber thumber.Thumber,
	log *zap.Logger) (*Thumbnailer, error) {
	return &Thumbnailer{
		db:      db,
		nc:      nc,
		thumber: thumber,
		log:     log.Sugar(),
	}, nil
}

func (t *Thumbnailer) Start(ctx context.Context) error {
	conn, err := t.nc.Subscribe("new-photo", t.processMsg(ctx))
	if err != nil {
		return err
	}
	t.conn = conn
	return nil
}

func (t *Thumbnailer) Stop() error {
	return t.conn.Drain()
}

func (t *Thumbnailer) processMsg(ctx context.Context) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var req eventsv1.NewPhoto
		if err := proto.Unmarshal(msg.Data, &req); err != nil {
			t.log.Errorf("unmarshaling new photo message: %v", err)
			msg.Nak()
			return
		}

		evt, err := t.db.GetEvent(ctx, req.GetEventId())
		if err != nil {
			t.log.Errorf("getting photo's event: %v", err)
			msg.Nak()
			return
		}

		fi, err := t.db.GetFileInfo(ctx, req.GetFileId())
		if err != nil {
			t.log.Errorf("getting photo ifno: %v", err)
			msg.Nak()
			return
		}

		buf, err := evt.Storage.Get(ctx, fi.Name)
		if err != nil {
			t.log.Errorf("getting photo: %v", err)
			msg.Nak()
			return
		}
		defer buf.Close()

		t.log.Info("creating thumbnail")
		thumnail, err := t.thumber.Thumb(buf)
		if err != nil {
			t.log.Errorf("creating thumbnail: %v", err)
			msg.Nak()
			return
		}

		if err := t.db.AddFileInfo(ctx, &db.FileInfo{
			ID:        uuid.NewString(),
			Name:      "thumb_" + fi.Name,
			EventID:   uint(req.GetEventId()),
			Thumbnail: true,
		}); err != nil {
			t.log.Errorf("saving thumbnail info to db: %v", err)
			msg.Nak()
			return
		}

		if err := evt.Storage.Store(ctx, "thumb_"+fi.Name, thumnail); err != nil {
			t.log.Errorf("storing thumbnail: %v", err)
			msg.Nak()
			return
		}

		msg.Ack()
	}
}
