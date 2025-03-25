package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/jj-style/eventpix/internal/data/db"
	eventsv1 "github.com/jj-style/eventpix/internal/gen/events/v1"
	picturev1 "github.com/jj-style/eventpix/internal/gen/picture/v1"
	"github.com/jj-style/eventpix/internal/pkg/validate"
	"github.com/jj-style/eventpix/internal/service/prodto"
	"github.com/nats-io/nats.go"
	gormcrypto "github.com/pkasila/gorm-crypto"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type EventpixService interface {
	GetEvent(context.Context, *picturev1.GetEventRequest) (*picturev1.GetEventResponse, error)
	GetEvents(context.Context, *picturev1.GetEventsRequest, uint) (*picturev1.GetEventsResponse, error)
	CreateEvent(context.Context, uint, *picturev1.CreateEventRequest) (*picturev1.CreateEventResponse, error)
	GetThumbnails(context.Context, *picturev1.GetThumbnailsRequest) (*picturev1.GetThumbnailsResponse, error)
	SetEventLive(context.Context, *picturev1.SetEventLiveRequest) (*picturev1.SetEventLiveResponse, error)
	DeleteEvent(context.Context, *picturev1.DeleteEventRequest) (*emptypb.Empty, error)
	Upload(context.Context, uint64, string, io.Reader, string) error
	GetThumbnailInfo(context.Context, string) (*picturev1.Thumbnail, error)
}

type eventpixSvc struct {
	logger    *zap.SugaredLogger
	db        db.DB
	nc        *nats.Conn
	validator validate.Validator
}

func NewEventpixService(logger *zap.Logger, db db.DB, nc *nats.Conn, validator validate.Validator) EventpixService {
	return &eventpixSvc{logger: logger.Sugar(), db: db, nc: nc, validator: validator}
}

func (p *eventpixSvc) CreateEvent(ctx context.Context, userId uint, req *picturev1.CreateEventRequest) (*picturev1.CreateEventResponse, error) {
	createEvent := &db.Event{
		Name:   req.GetName(),
		Slug:   req.GetSlug(),
		Live:   req.GetLive(),
		UserID: userId,
	}
	switch st := req.GetStorage().(type) {
	case *picturev1.CreateEventRequest_Filesystem:
		createEvent.FileSystemStorage = &db.FileSystemStorage{
			Directory: st.Filesystem.GetDirectory(),
		}
	case *picturev1.CreateEventRequest_S3:
		createEvent.S3Storage = &db.S3Storage{
			Bucket:    st.S3.GetBucket(),
			AccessKey: gormcrypto.EncryptedValue{Raw: st.S3.GetAccessKey()},
			SecretKey: gormcrypto.EncryptedValue{Raw: st.S3.GetSecretKey()},
			Region:    st.S3.GetRegion(),
			Endpoint:  st.S3.GetEndpoint(),
		}
	case *picturev1.CreateEventRequest_GoogleDrive:
		createEvent.GoogleDriveStorage = &db.GoogleDriveStorage{
			DirectoryID: st.GoogleDrive.GetFolderId(),
		}
	default:
		return nil, errors.New("unsupported storage type")
	}

	if err := p.validator.ValidateEvent(createEvent); err != nil {
		return nil, err
	}

	p.logger.Infof("creating event: %+v\n", createEvent)
	id, err := p.db.CreateEvent(ctx, createEvent)
	if err != nil {
		p.logger.Errorf("creating event: %v", err)
		return nil, fmt.Errorf("creating event: %v", err)
	}

	resp := prodto.CreateEventResponse(id)
	return resp, nil
}

func (p *eventpixSvc) SetEventLive(ctx context.Context, req *picturev1.SetEventLiveRequest) (*picturev1.SetEventLiveResponse, error) {
	evt, err := p.db.SetEventLive(ctx, req.GetId(), req.GetLive())
	return &picturev1.SetEventLiveResponse{Event: prodto.Event(evt, false)}, err
}

func (p *eventpixSvc) DeleteEvent(ctx context.Context, req *picturev1.DeleteEventRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, p.db.DeleteEvent(ctx, req.GetId())
}

func (p *eventpixSvc) GetEvent(ctx context.Context, req *picturev1.GetEventRequest) (*picturev1.GetEventResponse, error) {
	var event *db.Event
	var err error
	switch value := req.Value.(type) {
	case *picturev1.GetEventRequest_Id:
		event, err = p.db.GetEvent(ctx, value.Id)
	case *picturev1.GetEventRequest_Slug:
		event, err = p.db.GetEventBySlug(ctx, value.Slug)
	}
	if err != nil {
		p.logger.Errorf("getting event %d: %v", req.GetId(), err)
		return nil, fmt.Errorf("getting event: %v", err)
	}

	resp := &picturev1.GetEventResponse{Event: prodto.Event(event, true)}
	return resp, nil
}

func (p *eventpixSvc) GetEvents(ctx context.Context, _ *picturev1.GetEventsRequest, userId uint) (*picturev1.GetEventsResponse, error) {
	events, err := p.db.GetEvents(ctx, userId)
	if err != nil {
		p.logger.Errorf("getting events: %v", err)
		return nil, fmt.Errorf("getting events: %v", err)
	}

	resp := &picturev1.GetEventsResponse{
		Events: lo.Map(events, func(item *db.Event, _ int) *picturev1.Event { return prodto.Event(item, false) }),
	}
	return resp, nil
}

func (p *eventpixSvc) GetThumbnails(ctx context.Context, req *picturev1.GetThumbnailsRequest) (*picturev1.GetThumbnailsResponse, error) {
	thumbnails, err := p.db.GetThumbnails(ctx, uint(req.GetEventId()), int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		p.logger.Errorf("getting thumbnails from db: %v", err)
		return nil, err
	}

	resp := &picturev1.GetThumbnailsResponse{
		Thumbnails: lo.Map(thumbnails, func(item *db.ThumbnailInfo, _ int) *picturev1.Thumbnail { return prodto.Thumbnail(item) }),
	}
	return resp, nil
}

func (p *eventpixSvc) GetThumbnailInfo(ctx context.Context, id string) (*picturev1.Thumbnail, error) {
	t, err := p.db.GetThumbnailInfo(ctx, id)
	if err != nil {
		return nil, err
	}
	return prodto.Thumbnail(t), nil
}

func (p *eventpixSvc) Upload(ctx context.Context, eventId uint64, filename string, src io.Reader, contentType string) error {
	var mt eventsv1.NewMedia_MediaType

	switch contentType {
	case "image/png", "image/jpeg", "image/heif":
		mt = eventsv1.NewMedia_IMAGE
	case "video/avi", "video/mp4", "video/mpeg", "video/webm":
		mt = eventsv1.NewMedia_VIDEO
	default:
		return fmt.Errorf("unsupported content-type: '%s'", contentType)
	}

	evt, err := p.db.GetEvent(ctx, eventId)
	if err != nil {
		p.logger.Errorf("getting event to upload to: %w", err)
		return err
	}

	if !evt.Live {
		return errors.New("event is not live")
	}

	id, err := evt.Storage.Store(ctx, filename, src)
	if err != nil {
		p.logger.Errorf("error storing image: %w", err)
		return err
	}
	if err := p.db.AddFileInfo(ctx, &db.FileInfo{
		ID:      id,
		EventID: uint(eventId),
		Name:    filename,
		Video:   mt == eventsv1.NewMedia_VIDEO,
	}); err != nil {
		p.logger.Errorf("error storing file info: %w", err)
		return err
	}

	newPhotoMsg := &eventsv1.NewMedia{
		EventId: uint64(eventId),
		FileId:  id,
		Type:    mt,
	}
	payload, err := json.Marshal(newPhotoMsg)
	if err != nil {
		p.logger.Errorf("serializing event message: %v", err)
		return err
	}
	if err := p.nc.Publish("new-photo", payload); err != nil {
		p.logger.Errorf("publishing new photo event: %v", err)
	}

	return nil
}
