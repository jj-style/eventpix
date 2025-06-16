package cache

import (
	"context"
	"encoding/base64"

	gocache "github.com/eko/gocache/lib/v4/cache"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte) error
}

type cache struct {
	c gocache.CacheInterface[string]
}

func NewCache(c gocache.CacheInterface[string]) Cache {
	// TODO(jj): encrypt data?

	return &cache{c}
}

func (c *cache) Get(ctx context.Context, key string) ([]byte, error) {
	got, err := c.c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(got)
}

func (c *cache) Set(ctx context.Context, key string, val []byte) error {
	enc := base64.StdEncoding.EncodeToString(val)
	return c.c.Set(ctx, key, enc)
}
