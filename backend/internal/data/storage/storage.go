package storage

import (
	"context"
	"errors"
	"io"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

type Storage interface {
	Store(context.Context, string, io.Reader) error
	Get(context.Context, string) (io.ReadCloser, error)
}
