package storage

import (
	"context"
	"errors"
	"io"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

// Storage is a simple interface for storing and retrieving data (files) from somewhere
//
//go:generate go tool mockery
type Storage interface {
	Store(context.Context, string, io.Reader) (string, error)
	Get(context.Context, string) (io.ReadCloser, error)
}
