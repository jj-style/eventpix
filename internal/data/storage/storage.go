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
//go:generate go run github.com/vektra/mockery/v2
type Storage interface {
	Store(context.Context, string, io.Reader) error
	Get(context.Context, string) (io.ReadCloser, error)
}
