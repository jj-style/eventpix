package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jj-style/eventpix/backend/internal/config"
	"github.com/jj-style/eventpix/backend/internal/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the main eventpix server",
	Run:   runServer,
}

type serverApp struct {
	server      *http.Server
	thumbnailer *service.Thumbnailer
}

func newServerApp(cfg *config.Config, logger *zap.Logger, srv *http.Server, publisher message.Publisher) (*serverApp, func(), error) {
	app := &serverApp{
		server:      srv,
		thumbnailer: nil,
	}

	var thumbnailer *service.Thumbnailer
	cleanup := func() {}
	var err error

	if cfg.PubSub.Mode == "memory" {
		thumbnailer, cleanup, err = initializeMemThumbnailer(cfg, logger, publisher.(*gochannel.GoChannel))
	} else if cfg.PubSub.InProcess {
		thumbnailer, cleanup, err = initializeThumbnailer(cfg, logger)
	}
	if err != nil {
		return nil, func() {}, err
	}
	app.thumbnailer = thumbnailer
	return app, cleanup, nil
}

func runServer(cmd *cobra.Command, args []string) {
	logger := initLogger()
	defer logger.Sync() // flushes buffer, if any

	app, cleanup, err := initializeServer(cfg, logger)
	if err != nil {
		logger.Fatal("initializating server", zap.Error(err))
	}
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("listening on", zap.String("address", cfg.Server.Address))
		if err := app.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP listen and serve", zap.Error(err))
		}
	}()

	// spawn thumbnailer worker in same process as server
	if app.thumbnailer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := app.thumbnailer.Start(ctx, int(cfg.PubSub.Workers)); err != nil {
				logger.Fatal("failed to start thumbnailer", zap.Error(err))
			}
		}()
	}

	<-signals
	logger.Info("shutting down")
	if err := app.server.Shutdown(ctx); err != nil {
		logger.Fatal("HTTP shutdown", zap.Error(err))
	}
	cancel()
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	serverCmd.Flags().StringP("environment", "e", "production", "Set servers environment")
	viper.BindPFlag("server.environment", serverCmd.Flags().Lookup("environment"))
}
