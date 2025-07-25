// defines the models stored in the DB
package db

import (
	"github.com/jj-style/eventpix/internal/data/storage"
	gormcrypto "github.com/pkasila/gorm-crypto"
	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	Name string
	Slug string `gorm:"uniqueIndex"`
	Live bool
	// whether or not to cache media stored in the event
	Cache          bool
	FileInfos      []FileInfo
	ThumbnailInfos []ThumbnailInfo
	UserID         uint
	User           User
	Active         bool
	Password       *gormcrypto.EncryptedValue

	storage.Storage `gorm:"-"`
	// All available storage options for the event
	// Only one should be set, the rest must be null
	FileSystemStorage  *FileSystemStorage
	S3Storage          *S3Storage
	GoogleDriveStorage *GoogleDriveStorage
	FtpStorage         *FtpStorage
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
	Username         string
	Password         string
	Admin            bool
	Events           []Event
	GoogleDriveToken *GoogleDriveToken
}

type FileSystemStorage struct {
	gorm.Model
	Directory string
	EventID   uint
}

type S3Storage struct {
	gorm.Model
	Region    string
	AccessKey gormcrypto.EncryptedValue
	SecretKey gormcrypto.EncryptedValue
	Bucket    string
	Endpoint  string
	EventID   uint
	Insecure  bool
}

type GoogleDriveStorage struct {
	gorm.Model
	DirectoryID string
	EventID     uint
}

type GoogleDriveToken struct {
	gorm.Model
	Token  gormcrypto.EncryptedValue
	UserID uint
}

type FtpStorage struct {
	gorm.Model
	Address   string
	Directory string
	Username  gormcrypto.EncryptedValue
	Password  gormcrypto.EncryptedValue
	EventID   uint
}
