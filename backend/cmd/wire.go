//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

package cmd

import (
	"net/http"

	"github.com/google/wire"
	"github.com/jj-style/eventpix/backend/internal/config"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/events"
	"github.com/jj-style/eventpix/backend/internal/pkg/thumber"
	"github.com/jj-style/eventpix/backend/internal/server"
	"github.com/jj-style/eventpix/backend/internal/service"
	"go.uber.org/zap"
)

func initializeServer(cfg *config.Config, logger *zap.Logger) (*http.Server, func(), error) {
	panic(wire.Build(config.Provider, db.NewDb, events.NewNats, service.NewPictureServiceServer, server.NewServer))
}

func initializeThumbnailer(cfg *config.Config, logger *zap.Logger) (*service.Thumbnailer, func(), error) {
	panic(wire.Build(config.Provider, db.NewDb, events.NewNats, thumber.NewThumber, service.NewThumbnailer))
}
