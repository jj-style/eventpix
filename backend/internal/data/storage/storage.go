package storage

import (
	"context"
	"errors"
	"io"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

//go:generate go run github.com/vektra/mockery/v2
type Storage interface {
	Store(context.Context, string, io.Reader) error
	Get(context.Context, string) (io.ReadCloser, error)
}
