package cmd

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/service"
	"github.com/nats-io/nats.go"
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

// builds the final app to run for the server command.
// This handles running an in-memory nats server and thumbnailer based on the config
func newServerApp(cfg *config.Config, logger *zap.Logger, nc *nats.Conn, srv *http.Server, cache cache.CacheInterface[[]byte]) (*serverApp, func(), error) {
	app := &serverApp{
		server:      srv,
		thumbnailer: nil,
	}

	var thumbnailer *service.Thumbnailer
	cleanup := func() {}
	var err error

	if cfg.Nats.InProcess {
		thumbnailer, cleanup, err = initializeThumbnailerInProc(cfg, logger, nc, cache)
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

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

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
			if err := app.thumbnailer.Start(ctx); err != nil {
				logger.Fatal("failed to start thumbnailer", zap.Error(err))
			}
		}()
	}

	<-ctx.Done()
	logger.Info("shutting down")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := app.server.Shutdown(ctx); err != nil {
		logger.Fatal("HTTP shutdown", zap.Error(err))
	}
	wg.Wait()
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
