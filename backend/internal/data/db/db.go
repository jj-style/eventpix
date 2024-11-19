package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jj-style/eventpix/backend/internal/config"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:generate go run github.com/vektra/mockery/v2
type DB interface {
	CreateEvent(context.Context, *Event) (uint, error)
	GetEvents(context.Context) ([]*Event, error)
	GetEvent(context.Context, uint64) (*Event, error)
	AddFileInfo(context.Context, *FileInfo) error
	GetFileInfo(context.Context, string) (*FileInfo, error)
}

type dbImpl struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewDb(cfg *config.Database, logger *zap.Logger) (DB, func(), error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.Uri)
	case "mysql":
		dialector = mysql.Open(cfg.Uri)
	default:
		return nil, func() {}, fmt.Errorf("unsupported db driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, func() {}, fmt.Errorf("opening database: %w", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(
		&User{},
		&Event{},
		&FileInfo{},
		&FileSystemStorage{},
	); err != nil {
		return nil, func() {}, fmt.Errorf("migrating db: %w", err)
	}

	// create initial admin user
	var admin User
	db.Where(User{Admin: true}).
		Attrs(User{
			Username: "admin",
			Password: string(lo.Must(bcrypt.GenerateFromPassword([]byte("admin"), 8))),
		}).
		FirstOrCreate(&admin)

	return &dbImpl{db, logger.Sugar()}, func() {}, nil
}

func (d *dbImpl) CreateEvent(ctx context.Context, evt *Event) (uint, error) {
	result := d.db.WithContext(ctx).Create(evt)
	if result.Error != nil {
		d.log.Errorf("error creating event in db: %w", result.Error)
		return 0, result.Error
	}
	return evt.ID, nil
}
func (d *dbImpl) GetEvents(ctx context.Context) ([]*Event, error) {
	var events []Event
	result := d.db.WithContext(ctx).Find(&events)
	if result.Error != nil {
		d.log.Errorf("error querying events in db: %w", result.Error)
		return nil, result.Error
	}
	return lo.Map(events, func(e Event, _ int) *Event { return &e }), nil
}

func (d *dbImpl) GetEvent(ctx context.Context, id uint64) (*Event, error) {
	var event Event
	result := d.db.WithContext(ctx).Preload(clause.Associations).First(&event, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			d.log.Errorf("event(%d) not found in db", id)
			return nil, result.Error
		} else {
			d.log.Errorf("getting event(%d) from db: %v", id, result.Error)
			return nil, result.Error
		}
	}

	if err := ExtractEventStorage(&event); err != nil {
		d.log.Errorf("extracting event(%d) storage: %v", id, err)
		return nil, err
	}

	return &event, nil
}

func (d *dbImpl) AddFileInfo(ctx context.Context, fi *FileInfo) error {
	result := d.db.WithContext(ctx).Create(fi)
	if result.Error != nil {
		d.log.Error("creating file info in db: %w", result.Error)
		return result.Error
	}
	return nil
}

func (d *dbImpl) GetFileInfo(ctx context.Context, id string) (*FileInfo, error) {
	var fi FileInfo
	result := d.db.WithContext(ctx).First(&fi, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			d.log.Errorf("file(%s) not found in db", id)
			return nil, result.Error
		} else {
			d.log.Errorf("getting file(%s) from db: %v", id, result.Error)
			return nil, result.Error
		}
	}
	return &fi, nil
}
