package service

import (
	"bytes"
	"context"
	"io"

	eventsv1 "github.com/jj-style/eventpix/backend/gen/events/v1"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/nats-io/nats.go"
	"github.com/prplecake/go-thumbnail"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Thumbnailer struct {
	db          db.DB
	nc          *nats.Conn
	log         *zap.SugaredLogger
	thumbConfig thumbnail.Generator
	conn        *nats.Subscription
}

func NewThumbnailer(db db.DB, nc *nats.Conn, log *zap.SugaredLogger) (*Thumbnailer, error) {
	thumber := &Thumbnailer{
		db:  db,
		nc:  nc,
		log: log,
		thumbConfig: thumbnail.Generator{
			Scaler: "CatmullRom",
			Width:  64,
			Height: 64,
		}}
	return thumber, nil
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

		buf, err := evt.Storage.Get(ctx, req.GetFilename())
		if err != nil {
			t.log.Errorf("getting photo: %v", err)
			msg.Nak()
			return
		}
		data, _ := io.ReadAll(buf)
		_ = buf.Close()
		t.log.Infow("creating thumbnail", "file", req.GetFilename(), "bytes", len(data))

		img, err := t.thumbConfig.NewImageFromByteArray(data)
		if err != nil {
			t.log.Errorf("creating image from bytes: %w", err)
			msg.Nak()
			return
		}

		thumBuf, err := t.thumbConfig.CreateThumbnail(img)
		if err != nil {
			t.log.Errorf("creating thumbnail: %v", err)
			msg.Nak()
			return
		}

		if err := evt.Storage.Store(ctx, "thumb_"+req.GetFilename(), bytes.NewReader(thumBuf)); err != nil {
			t.log.Errorf("storing thumbnail: %v", err)
			msg.Nak()
			return
		}

		msg.Ack()
	}
}
