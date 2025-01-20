// defines the models stored in the DB
package db

import (
	"github.com/jj-style/eventpix/internal/data/storage"
	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	Name           string
	Live           bool
	FileInfos      []FileInfo
	ThumbnailInfos []ThumbnailInfo
	UserID         uint

	storage.Storage `gorm:"-"`
	// All available storage options for the event
	// Only one should be set, the rest must be null
	FileSystemStorage *FileSystemStorage
	S3Storage         *S3Storage
}

type FileInfo struct {
	gorm.Model
	ID      string
	EventID uint
	Event   Event
	Name    string
	Video   bool
}

type ThumbnailInfo struct {
	gorm.Model
	ID         string
	Name       string
	EventID    uint
	Event      Event
	FileInfoID string
	FileInfo   FileInfo
}

type User struct {
	gorm.Model
	Username string
	Password string
	Admin    bool
	Events   []Event
}

type FileSystemStorage struct {
	gorm.Model
	Directory string
	EventID   uint
}

type S3Storage struct {
	gorm.Model
	Region    string
	AccessKey string
	// TODO(jj): encrypt at rest
	SecretKey string
	Bucket    string
	Endpoint  string
	EventID   uint
}
