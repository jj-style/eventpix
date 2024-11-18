package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/samber/lo"
)

type memStore struct {
	files map[string][]byte
}

func (m *memStore) Get(_ context.Context, name string) (io.ReadCloser, error) {
	if got, ok := m.files[name]; ok {
		return io.NopCloser(bytes.NewReader(got)), nil
	} else {
		return nil, fmt.Errorf("%s not found", name)
	}
}

func (m *memStore) List(_ context.Context) ([]io.ReadCloser, error) {
	return lo.MapToSlice(m.files, func(_ string, data []byte) io.ReadCloser {
		return io.NopCloser(bytes.NewReader(data))
	}), nil
}

func (m *memStore) Store(_ context.Context, name string, data io.Reader) error {
	buf, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	m.files[name] = buf
	return nil
}

func NewMemStore() Storage {
	return &memStore{files: make(map[string][]byte)}
}
