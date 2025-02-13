package service

import (
	"context"
	"io"

	"github.com/jj-style/eventpix/internal/data/db"
	"go.uber.org/zap"
)

type StorageService interface {
	GetThumbnail(ctx context.Context, id string) (string, []byte, error)
	GetPicture(ctx context.Context, id string) (string, []byte, error)
}

func NewStorageService(db db.DB, log *zap.Logger) StorageService {
	return &storageService{
		db:  db,
		log: log,
	}
}

type storageService struct {
	db  db.DB
	log *zap.Logger
}

func (s *storageService) GetThumbnail(ctx context.Context, id string) (string, []byte, error) {
	// get thumbnail details from db
	ti, err := s.db.GetThumbnailInfo(ctx, id)
	if err != nil {
		s.log.Sugar().Errorf("getting thumbnail info for %s: %v", id, err)
		return "", nil, err
	}

	// get storage from thumbanils event
	evt, err := s.db.GetEvent(ctx, uint64(ti.EventID))
	if err != nil {
		s.log.Sugar().Errorf("getting event for thumbnail: %s: %v", id, err)
		return "", nil, err
	}

	data, err := evt.Storage.Get(ctx, ti.ID)
	if err != nil {
		s.log.Sugar().Errorf("getting thumbnail data for %s: %v", id, err)
		return "", nil, err
	}
	defer data.Close()

	buf, err := io.ReadAll(data)
	if err != nil {
		s.log.Sugar().Errorf("error reading data: %v", err)
		return "", nil, err
	}
	return ti.Name, buf, nil
}

func (s *storageService) GetPicture(ctx context.Context, id string) (string, []byte, error) {
	// get picture details from db
	fi, err := s.db.GetFileInfo(ctx, id)
	if err != nil {
		s.log.Sugar().Errorf("getting file info for %s: %v", id, err)
		return "", nil, err
	}

	// get storage from files event
	evt, err := s.db.GetEvent(ctx, uint64(fi.EventID))
	if err != nil {
		s.log.Sugar().Errorf("getting event for event: %s: %v", id, err)
		return "", nil, err
	}

	data, err := evt.Storage.Get(ctx, fi.ID)
	if err != nil {
		s.log.Sugar().Errorf("getting file data for %s: %v", id, err)
		return "", nil, err
	}
	defer data.Close()

	buf, err := io.ReadAll(data)
	if err != nil {
		s.log.Sugar().Errorf("error reading data: %v", err)
		return "", nil, err
	}
	return fi.Name, buf, nil
}
