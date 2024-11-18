package service

import (
	"context"
	"io"

	eventsv1 "github.com/jj-style/eventpix/backend/gen/events/v1"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Thumbnailer interface {
	Run() (func(), error)
}

type thumbnailer struct {
	db  db.DB
	nc  *nats.Conn
	log *zap.SugaredLogger
}

func NewThumbnailer(db db.DB, nc *nats.Conn, log *zap.SugaredLogger) Thumbnailer {
	return &thumbnailer{db, nc, log}
}

func (t *thumbnailer) Run() (func(), error) {
	ctx := context.TODO()

	conn, err := t.nc.Subscribe("new-photo", func(msg *nats.Msg) {
		var req eventsv1.NewPhoto
		if err := proto.Unmarshal(msg.Data, &req); err != nil {
			t.log.Errorf("unmarshaling new photo message: %w", err)
			msg.Nak()
			return
		}

		evt, err := t.db.GetEvent(ctx, req.GetEventId())
		if err != nil {
			t.log.Errorf("getting photo's event: %w", err)
			msg.Nak()
			return
		}

		buf, err := evt.Storage.Get(ctx, req.GetFilename())
		if err != nil {
			t.log.Errorf("getting photo: %w", err)
			msg.Nak()
			return
		}
		data, _ := io.ReadAll(buf)
		_ = buf.Close()
		t.log.Infow("creating thumbnail", "file", req.GetFilename(), "bytes", len(data))
	})
	if err != nil {
		return func() {}, err
	}
	return func() { conn.Drain() }, nil
}
