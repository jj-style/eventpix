package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jj-style/eventpix/internal/cache"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/data/db"
	eventsv1 "github.com/jj-style/eventpix/internal/gen/events/v1"
	"github.com/jj-style/eventpix/internal/pkg/imagor"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type Thumbnailer struct {
	db        db.DB
	thumber   imagor.Imagor
	nc        *nats.Conn
	log       *zap.SugaredLogger
	serverUrl string
	cache     cache.Cache
}

func NewThumbnailer(
	cfg *config.Config,
	db db.DB,
	thumber imagor.Imagor,
	nc *nats.Conn,
	log *zap.Logger,
	cache cache.Cache,
) (*Thumbnailer, error) {
	return &Thumbnailer{
		db:        db,
		thumber:   thumber,
		nc:        nc,
		log:       log.Sugar(),
		serverUrl: cfg.Server.InternalServerUrl,
		cache:     cache,
	}, nil
}

func (t *Thumbnailer) Start(ctx context.Context) error {
	go func() {
		t.log.Info("thumbnailer subscribing")
		sub, err := t.nc.QueueSubscribe("new-photo", "thumbnailer", func(msg *nats.Msg) {
			if err := t.Thumb(ctx, msg); err != nil {
				t.log.Error("processing thumbnail", zap.Error(err))
				msg.Nak()
				return
			}
			// we need to Acknowledge that we received and processed the message,
			// otherwise, it will be resent over and over again.
			msg.Ack()
			t.log.Infof("thumbnail processed")
		})
		if err != nil {
			t.log.Error("subscribing to topic", zap.Error(err))
		}
		defer sub.Drain()
		<-ctx.Done()
		t.log.Info("stopping thumbnailer")
	}()
	return nil
}

func (t *Thumbnailer) Thumb(ctx context.Context, msg *nats.Msg) error {
	var req eventsv1.NewMedia
	if err := json.Unmarshal(msg.Data, &req); err != nil {
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

	var thumbnail io.ReadCloser
	switch req.GetType() {
	case eventsv1.NewMedia_IMAGE:
		thumbnail, err = t.thumber.ThumbImage(fmt.Sprintf("%s/storage/picture/%s", t.serverUrl, req.GetFileId()))
	case eventsv1.NewMedia_VIDEO:
		thumbnail, err = t.thumber.ThumbVideo(fmt.Sprintf("%s/storage/picture/%s", t.serverUrl, req.GetFileId()))
	}
	if err != nil {
		t.log.Errorf("failed when generating thumbnail: %v", err)
		return err
	}
	defer thumbnail.Close()
	cacheBuf := bytes.NewBuffer(nil)
	var thumbTee io.Reader = thumbnail
	if evt.Cache {
		thumbTee = io.TeeReader(thumbnail, cacheBuf)
	}

	tname := "thumb_" + strings.TrimRight(fi.Name, filepath.Ext(fi.Name)) + ".webp"

	id, err := evt.Storage.Store(ctx, tname, thumbTee)
	if err != nil {
		t.log.Errorf("storing thumbnail: %v", err)
		return err
	}
	if evt.Cache {
		if err := t.cache.Set(ctx, fmt.Sprintf("%d:%s", evt.ID, id), cacheBuf.Bytes()); err != nil {
			t.log.Warnf("failed to store thumbnail in cache: %v", err)
		}
	}
	if err := t.db.AddThumbnailInfo(ctx, &db.ThumbnailInfo{
		ID:         id,
		Name:       tname,
		EventID:    uint(req.GetEventId()),
		FileInfoID: fi.ID,
	}); err != nil {
		t.log.Errorf("saving thumbnail info to db: %v", err)
		return err
	}

	if err := t.nc.Publish("new-thumbnail", []byte(id)); err != nil {
		t.log.Errorf("sending new thumbnail message: %v", err)
		return err
	}

	return nil
}
