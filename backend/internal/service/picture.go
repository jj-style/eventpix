package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	eventsv1 "github.com/jj-style/eventpix/backend/gen/events/v1"
	picturev1 "github.com/jj-style/eventpix/backend/gen/picture/v1"
	"github.com/jj-style/eventpix/backend/gen/picture/v1/picturev1connect"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/service/prodto"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type pictureServiceServer struct {
	log *zap.SugaredLogger
	db  db.DB
	picturev1connect.UnimplementedPictureServiceHandler
	publisher message.Publisher
}

func NewPictureServiceServer(logger *zap.Logger, db db.DB, publisher message.Publisher) picturev1connect.PictureServiceHandler {
	return &pictureServiceServer{
		log:       logger.Sugar(),
		db:        db,
		publisher: publisher,
	}
}

func (p *pictureServiceServer) CreateEvent(ctx context.Context, req *connect.Request[picturev1.CreateEventRequest]) (*connect.Response[picturev1.CreateEventResponse], error) {
	msg := req.Msg
	p.log.Infow("creating event", "name", msg.GetName())
	createEvent := &db.Event{
		Name: msg.GetName(),
		Live: msg.GetLive(),
	}
	switch st := msg.GetStorage().(type) {
	case *picturev1.CreateEventRequest_Filesystem:
		createEvent.FileSystemStorage = &db.FileSystemStorage{
			Directory: st.Filesystem.GetDirectory(),
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported storage type"))
	}
	id, err := p.db.CreateEvent(ctx, createEvent)

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
	// TODO: mock
	id := uuid.NewString()

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
		ID:      id,
		EventID: uint(req.Msg.GetEventId()),
		Name:    req.Msg.GetFile().GetName(),
	}); err != nil {
		p.log.Errorf("error storing file info: %w", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	newPhotoMsg := &eventsv1.NewPhoto{
		EventId: uint64(req.Msg.GetEventId()),
		FileId:  id,
	}
	payload, err := json.Marshal(newPhotoMsg)
	if err != nil {
		p.log.Errorf("serializing event message: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := p.publisher.Publish("new-photo", message.NewMessage(watermill.NewShortUUID(), payload)); err != nil {
		p.log.Errorf("publishing new photo event: %v", err)
	}

	resp := &picturev1.UploadResponse{}
	return connect.NewResponse(resp), nil
}

func (p *pictureServiceServer) GetThumbnails(ctx context.Context, req *connect.Request[picturev1.GetThumbnailsRequest]) (*connect.Response[picturev1.GetThumbnailsResponse], error) {
	msg := req.Msg

	thumbnails, err := p.db.GetThumbnails(ctx, uint(msg.GetEventId()), int(msg.GetLimit()), int(msg.GetOffset()))
	if err != nil {
		p.log.Errorf("getting thumbnails from db: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &picturev1.GetThumbnailsResponse{
		Thumbnails: lo.Map(thumbnails, func(item *db.ThumbnailInfo, _ int) *picturev1.Thumbnail { return prodto.Thumbnail(item) }),
	}
	return connect.NewResponse(resp), nil
}
