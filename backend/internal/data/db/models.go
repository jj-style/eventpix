package db

import (
	"github.com/jj-style/eventpix/backend/internal/data/storage"
	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	Name      string
	Live      bool
	FileInfos []FileInfo
	// TODO: add when some auth middleware
	// UserID    uint

	storage.Storage `gorm:"-"`
	// All available storage options for the event
	// Only one should be set, the rest must be null
	*FileSystemStorage
}

type FileInfo struct {
	gorm.Model
	EventID uint
	StoreID string
}

type User struct {
	gorm.Model
	Username string
	Password string
	Admin    bool
	// TODO: add when some auth middleware
	// Events   []Event
}

type FileSystemStorage struct {
	gorm.Model
	Directory string
	EventID   uint
}
