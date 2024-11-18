package service

import (
	"bytes"
	"context"

	"connectrpc.com/connect"
	picturev1 "github.com/jj-style/eventpix/backend/gen/picture/v1"
	"github.com/jj-style/eventpix/backend/gen/picture/v1/picturev1connect"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/prodto"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type pictureServiceServer struct {
	log *zap.SugaredLogger
	db  db.DB
	picturev1connect.UnimplementedPictureServiceHandler
}

func NewPictureServiceServer(logger *zap.Logger, db db.DB) picturev1connect.PictureServiceHandler {
	return &pictureServiceServer{
		log: logger.Sugar(),
		db:  db,
	}
}

func (p *pictureServiceServer) CreateEvent(ctx context.Context, req *connect.Request[picturev1.CreateEventRequest]) (*connect.Response[picturev1.CreateEventResponse], error) {
	msg := req.Msg
	p.log.Infow("creating event", "name", msg.GetName())
	id, err := p.db.CreateEvent(ctx, &db.Event{
		Name: msg.GetName(),
		Live: msg.GetLive(),
	})

	if err != nil {
		p.log.Errorf("error creating event: %w", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := prodto.CreateEventResponse(id)
	return connect.NewResponse(resp), nil
}

func (p *pictureServiceServer) GetEvent(ctx context.Context, req *connect.Request[picturev1.GetEventRequest]) (*connect.Response[picturev1.GetEventResponse], error) {
	event, err := p.db.GetEvent(ctx, req.Msg.GetId())
	if err != nil {
		p.log.Errorf("getting event", "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &picturev1.GetEventResponse{Event: prodto.Event(event, true)}
	return connect.NewResponse(resp), nil
}

func (p *pictureServiceServer) GetEvents(ctx context.Context, req *connect.Request[picturev1.GetEventsRequest]) (*connect.Response[picturev1.GetEventsResponse], error) {
	events, err := p.db.GetEvents(ctx)
	if err != nil {
		p.log.Errorf("error getting events: %w", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &picturev1.GetEventsResponse{
		Events: lo.Map(events, func(item *db.Event, _ int) *picturev1.Event { return prodto.Event(item, false) }),
	}
	return connect.NewResponse(resp), nil
}

func (p *pictureServiceServer) Upload(ctx context.Context, req *connect.Request[picturev1.UploadRequest]) (*connect.Response[picturev1.UploadResponse], error) {
	evt, err := p.db.GetEvent(ctx, req.Msg.GetEventId())
	if err != nil {
		p.log.Errorf("getting event to upload to: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	buffer := bytes.NewReader(req.Msg.GetFile().GetData())
	if err = evt.Storage.Store(ctx, req.Msg.GetFile().GetName(), buffer); err != nil {
		p.log.Errorf("error storing image: %w", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := p.db.AddFileInfo(ctx, &db.FileInfo{
		EventID: uint(req.Msg.GetEventId()),
		StoreID: req.Msg.GetFile().GetName(),
	}); err != nil {
		p.log.Errorf("error storing file info: %w", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &picturev1.UploadResponse{}
	return connect.NewResponse(resp), nil
}

// func (p *pictureServiceServer) ListGallery(ctx context.Context, req *connect.Request[picturev1.ListRequest]) (*connect.Response[picturev1.ListResponse], error) {
// 	images, err := p.store.List()
// 	if err != nil {
// 		return nil, connect.NewError(connect.CodeInternal, err)
// 	}
// 	p.log.Infof("returning %d images", len(images))
// 	return connect.NewResponse(&picturev1.ListResponse{Files: images}), nil
// }
