package storage

import (
	"bytes"
	"context"
	"io"
	"sync"
)

type memStore struct {
	files map[string][]byte
	sync.RWMutex
}

func (m *memStore) Get(_ context.Context, name string) (io.ReadCloser, error) {
	m.RLock()
	defer m.RUnlock()
	if got, ok := m.files[name]; ok {
		return io.NopCloser(bytes.NewReader(got)), nil
	} else {
		return nil, ErrFileNotFound
	}
}

func (m *memStore) Store(_ context.Context, name string, data io.Reader) error {
	m.Lock()
	defer m.Unlock()
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
