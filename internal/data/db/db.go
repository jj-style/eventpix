// implementation of the DB logic
package db

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/pkg/utils/auth"
	gormcrypto "github.com/pkasila/gorm-crypto"
	"github.com/pkasila/gorm-crypto/algorithms"
	"github.com/pkasila/gorm-crypto/serialization"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:generate go run github.com/vektra/mockery/v2
type DB interface {
	CreateEvent(context.Context, *Event) (uint, error)
	GetEvents(context.Context, uint) ([]*Event, error)
	GetEvent(context.Context, uint64) (*Event, error)
	GetEventBySlug(context.Context, string) (*Event, error)
	AddFileInfo(context.Context, *FileInfo) error
	GetFileInfo(context.Context, string) (*FileInfo, error)
	AddThumbnailInfo(context.Context, *ThumbnailInfo) error
	GetThumbnails(ctx context.Context, eventId uint, limit int, offset int) ([]*ThumbnailInfo, error)
	GetThumbnailInfo(context.Context, string) (*ThumbnailInfo, error)
	SetEventLive(context.Context, uint64, bool) (*Event, error)
	DeleteEvent(context.Context, uint64) error
	CreateUser(context.Context, string, string) error
	GetUser(context.Context, string) (*User, error)
	UserAuthorizedForEvent(context.Context, uint, uint) (bool, error)
	StoreGoogleToken(ctx context.Context, userId uint, token []byte) error
	GetGoogleToken(ctx context.Context, uideId uint) ([]byte, error)
	DeleteGoogleToken(ctx context.Context, userId uint) error
}

type dbImpl struct {
	db                *gorm.DB
	log               *zap.SugaredLogger
	googleOauthConfig *oauth2.Config
}

func NewDb(cfg *config.Database, logger *zap.Logger, googleOauthConfig *oauth2.Config) (DB, func(), error) {
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

	encryptionKey, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
	if err != nil {
		return nil, func() {}, err
	}

	aes, err := algorithms.NewAES256CBC(encryptionKey)
	if err != nil {
		return nil, func() {}, err
	}
	gormcrypto.Init(aes, serialization.NewJSON())

	// Migrate the schema
	if err := db.AutoMigrate(
		&User{},
		&GoogleDriveToken{},
		&Event{},
		&FileInfo{},
		&ThumbnailInfo{},
		&FileSystemStorage{},
		&S3Storage{},
		&GoogleDriveStorage{},
	); err != nil {
		return nil, func() {}, fmt.Errorf("migrating db: %w", err)
	}

	// create initial admin user
	var admin User
	db.Where(User{Admin: true}).
		Attrs(User{
			Username: "admin",
			Password: string(lo.Must(auth.EncryptPassword("admin"))),
		}).
		FirstOrCreate(&admin)

	return &dbImpl{db, logger.Sugar(), googleOauthConfig}, func() {}, nil
}

func (d *dbImpl) CreateEvent(ctx context.Context, evt *Event) (uint, error) {
	result := d.db.WithContext(ctx).Create(evt)
	if result.Error != nil {
		d.log.Errorf("error creating event in db: %w", result.Error)
		return 0, result.Error
	}
	return evt.ID, nil
}

func (d *dbImpl) SetEventLive(ctx context.Context, id uint64, live bool) (*Event, error) {
	var events []Event
	result := d.db.WithContext(ctx).
		Model(&events).
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Update("live", live)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("%d events affected", result.RowsAffected)
	}
	return &events[0], nil
}

func (d *dbImpl) DeleteEvent(ctx context.Context, id uint64) error {
	return d.db.WithContext(ctx).Delete(&Event{}, id).Error
}

func (d *dbImpl) GetEvents(ctx context.Context, userId uint) ([]*Event, error) {
	var events []Event
	result := d.db.WithContext(ctx).Where(&Event{UserID: userId}).Find(&events)
	if result.Error != nil {
		d.log.Errorf("error querying events in db: %w", result.Error)
		return nil, result.Error
	}
	return lo.Map(events, func(e Event, _ int) *Event { return &e }), nil
}

func (d *dbImpl) GetEvent(ctx context.Context, id uint64) (*Event, error) {
	var event Event
	result := d.db.WithContext(ctx).
		Preload(clause.Associations).
		Preload("User.GoogleDriveToken").
		First(&event, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			d.log.Errorf("event(%d) not found in db", id)
			return nil, result.Error
		} else {
			d.log.Errorf("getting event(%d) from db: %v", id, result.Error)
			return nil, result.Error
		}
	}

	if err := ExtractEventStorage(&event, d.googleOauthConfig); err != nil {
		d.log.Errorf("extracting event(%d) storage: %v", id, err)
		return nil, err
	}

	return &event, nil
}

func (d *dbImpl) GetEventBySlug(ctx context.Context, slug string) (*Event, error) {
	var event Event
	result := d.db.WithContext(ctx).
		Preload(clause.Associations).
		Preload("User.GoogleDriveToken").
		First(&event, "slug = ?", slug)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			d.log.Errorf("event with slug(%s) not found in db", slug)
			return nil, result.Error
		} else {
			d.log.Errorf("getting event with slug(%s) from db: %v", slug, result.Error)
			return nil, result.Error
		}
	}

	if err := ExtractEventStorage(&event, d.googleOauthConfig); err != nil {
		d.log.Errorf("extracting event with slug(%s) storage: %v", slug, err)
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
	result := d.db.
		WithContext(ctx).
		Preload(clause.Associations).
		First(&fi, "id = ?", id)
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

func (d *dbImpl) AddThumbnailInfo(ctx context.Context, ti *ThumbnailInfo) error {
	result := d.db.WithContext(ctx).Create(ti)
	if result.Error != nil {
		d.log.Error("creating thumbnail info in db: %w", result.Error)
		return result.Error
	}
	return nil
}

func (d *dbImpl) GetThumbnails(ctx context.Context, eventId uint, limit int, offset int) ([]*ThumbnailInfo, error) {
	var thumbnails []ThumbnailInfo
	result := d.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Preload(clause.Associations).
		Order("created_at desc").
		Find(&thumbnails, ThumbnailInfo{EventID: eventId})
	if result.Error != nil {
		d.log.Errorf("error querying thumbnails in event(%d): %w", eventId, result.Error)
		return nil, result.Error
	}
	return lo.Map(thumbnails, func(e ThumbnailInfo, _ int) *ThumbnailInfo { return &e }), nil
}

func (d *dbImpl) GetThumbnailInfo(ctx context.Context, id string) (*ThumbnailInfo, error) {
	var ti ThumbnailInfo
	result := d.db.
		WithContext(ctx).
		Preload(clause.Associations).
		First(&ti, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			d.log.Errorf("thumbnail(%s) not found in db", id)
			return nil, result.Error
		} else {
			d.log.Errorf("getting thumbnail(%s) from db: %v", id, result.Error)
			return nil, result.Error
		}
	}
	return &ti, nil
}

func (d *dbImpl) CreateUser(ctx context.Context, username string, password string) error {
	var existingUser User
	result := d.db.WithContext(ctx).Where("username = ?", username).First(&existingUser)
	if result.RowsAffected > 0 {
		return errors.New("user already exists")
	}

	password, err := auth.EncryptPassword(password)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %v", err)
	}
	if err := d.db.WithContext(ctx).Create(&User{Username: username, Password: password, Admin: false}).Error; err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}
	return nil
}

func (d *dbImpl) GetUser(ctx context.Context, username string) (*User, error) {
	var user User
	result := d.db.WithContext(ctx).Preload(clause.Associations).Where("username = ?", username).First(&user)
	if result.RowsAffected < 1 {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (d *dbImpl) UserAuthorizedForEvent(ctx context.Context, userId, eventId uint) (bool, error) {
	var event Event
	err := d.db.WithContext(ctx).Where(&Event{Model: gorm.Model{ID: eventId}, UserID: userId}).First(&event).Error
	if err == nil {
		return true, nil
	} else {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		} else {
			return false, err
		}
	}
}

func (d *dbImpl) StoreGoogleToken(ctx context.Context, userId uint, token []byte) error {
	return d.db.WithContext(ctx).
		Create(&GoogleDriveToken{Token: gormcrypto.EncryptedValue{Raw: token}, UserID: userId}).Error
}

func (d *dbImpl) GetGoogleToken(ctx context.Context, userId uint) ([]byte, error) {
	var goog GoogleDriveToken
	result := d.db.WithContext(ctx).
		Where(&GoogleDriveToken{UserID: userId}).
		First(&goog)
	if err := result.Error; err != nil {
		return nil, err
	}
	return goog.Token.Raw.([]byte), nil
}

func (d *dbImpl) DeleteGoogleToken(ctx context.Context, userId uint) error {
	err := d.db.WithContext(ctx).Unscoped().Delete(&GoogleDriveToken{}, "user_id = ?", userId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	} else {
		return nil
	}
}
