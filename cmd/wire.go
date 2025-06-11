//go:build wireinject
// +build wireinject

//go:generate go tool wire

package cmd

import (
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/google/wire"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/data/db"
	"github.com/jj-style/eventpix/internal/pkg/imagor"
	"github.com/jj-style/eventpix/internal/pkg/validate"
	"github.com/jj-style/eventpix/internal/server"
	"github.com/jj-style/eventpix/internal/service"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func initializeServer(cfg *config.Config, logger *zap.Logger) (*serverApp, func(), error) {
	panic(wire.Build(config.Provider, newGoogleDriveConfig, newNats, newHtmx, newCache, db.NewDb, validate.NewValidator, service.NewEventpixService, service.NewStorageService, service.NewAuthService, server.NewHttpServer, newServerApp))
}

func initializeThumbnailer(cfg *config.Config, logger *zap.Logger) (*service.Thumbnailer, func(), error) {
	panic(wire.Build(config.Provider, newGoogleDriveConfig, newNats, newCache, db.NewDb, imagor.NewImagor, service.NewThumbnailer))
}

func initializeThumbnailerInProc(cfg *config.Config, logger *zap.Logger, nc *nats.Conn, cache cache.CacheInterface[[]byte]) (*service.Thumbnailer, func(), error) {
	panic(wire.Build(config.Provider, newGoogleDriveConfig, db.NewDb, imagor.NewImagor, service.NewThumbnailer))
}
