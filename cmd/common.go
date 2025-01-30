// This file contains a few common things to construct and inject around various bits of the services
package cmd

import (
	"errors"
	"os"
	"time"

	"github.com/donseba/go-htmx"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/server"
	"github.com/nats-io/nats.go"
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
