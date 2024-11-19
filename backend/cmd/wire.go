//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

package cmd

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"github.com/jj-style/eventpix/backend/internal/config"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/pkg/pubsub"
	"github.com/jj-style/eventpix/backend/internal/pkg/thumber"
	"github.com/jj-style/eventpix/backend/internal/server"
	"github.com/jj-style/eventpix/backend/internal/service"
	"go.uber.org/zap"
)

func initializeServer(cfg *config.Config, logger *zap.Logger) (*serverApp, func(), error) {
	panic(wire.Build(config.Provider, pubsub.NewPublisher, db.NewDb, service.NewPictureServiceServer, server.NewServer, newServerApp))
}

func initializeThumbnailer(cfg *config.Config, logger *zap.Logger) (*service.Thumbnailer, func(), error) {
	panic(wire.Build(config.Provider, pubsub.NewSubscriber, db.NewDb, thumber.NewThumber, service.NewThumbnailer))
}

func initializeMemThumbnailer(cfg *config.Config, logger *zap.Logger, subscriber message.Subscriber) (*service.Thumbnailer, func(), error) {
	panic(wire.Build(config.Provider, db.NewDb, thumber.NewThumber, service.NewThumbnailer))
}
