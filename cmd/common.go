// This file contains a few common things to construct and inject around various bits of the services
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/donseba/go-htmx"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
	memcache_store "github.com/eko/gocache/store/memcache/v4"
	rueidis_store "github.com/eko/gocache/store/rueidis/v4"
	mycache "github.com/jj-style/eventpix/internal/cache"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/server"
	"github.com/nats-io/nats.go"
	go_cache "github.com/patrickmn/go-cache"
	"github.com/redis/rueidis"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

func initLogger() *zap.Logger {
	var logger *zap.Logger
	if cfg.Server.Environment == "development" {
		logger, _ = zap.NewDevelopment(zap.AddStacktrace(zap.DPanicLevel))
	} else {
		logger, _ = zap.NewProduction(zap.AddStacktrace(zap.DPanicLevel))
	}
	return logger
}

func newNats(cfg *config.Nats) (*nats.Conn, func(), error) {
	url := cfg.Url
	var srvShutdown func() = nil
	if cfg.InProcess {
		natsSrv, err := server.NewNatsServer()
		if err != nil {
			return nil, func() {}, err
		}
		go natsSrv.Start()
		if !natsSrv.ReadyForConnections(time.Second * 5) {
			return nil, func() {}, errors.New("embedded nats not ready for connections")
		}
		url = natsSrv.ClientURL()
		srvShutdown = natsSrv.Shutdown
	}
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, func() {}, err
	}
	return nc, func() {
		nc.Close()
		if srvShutdown != nil {
			srvShutdown()
		}
	}, nil
}

func newHtmx() *htmx.HTMX {
	return htmx.New()
}

func newGoogleDriveConfig(cfg *config.Config) (*oauth2.Config, error) {
	f, err := os.ReadFile(cfg.OauthSecrets.Google.SecretsFile)
	if err != nil {
		return nil, err
	}
	return google.ConfigFromJSON(f, drive.DriveFileScope)
}

func newCache(cfg *config.Cache) (mycache.Cache, error) {
	cache, err := newGoCache(cfg)
	if err != nil {
		return nil, err
	}
	return mycache.NewCache(cache), nil
}

func newGoCache(cfg *config.Cache) (cache.CacheInterface[string], error) {
	// default to in memory 60s cache if nothing specified
	if cfg == nil {
		cfg = &config.Cache{Mode: "memory", Ttl: 60}
	}
	switch cfg.Mode {
	case "memory":
		gocacheClient := go_cache.New(time.Second*time.Duration(cfg.Ttl), 10*time.Minute)
		gocacheStore := gocache_store.NewGoCache(gocacheClient)
		cacheManager := cache.New[string](gocacheStore)
		return cacheManager, nil
	case "redis":
		client, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress: strings.Split(cfg.Addr, ","),
			Username:    cfg.Username,
			Password:    cfg.Password,
		})
		if err != nil {
			return nil, err
		}
		cacheManager := cache.New[string](rueidis_store.NewRueidis(
			client,
			store.WithExpiration(time.Second*time.Duration(cfg.Ttl)),
			store.WithClientSideCaching(time.Second*time.Duration(cfg.Ttl))),
		)
		return cacheManager, nil
	case "memcache":
		memcacheStore := memcache_store.NewMemcache(
			memcache.New(strings.Split(cfg.Addr, ",")...),
			store.WithExpiration(time.Second*time.Duration(cfg.Ttl)),
		)
		cacheManager := cache.New[string](memcacheStore)
		return cacheManager, nil
	default:
		return nil, fmt.Errorf("unknown cache mode: '%s'", cfg.Mode)
	}
}
