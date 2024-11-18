package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// thumbnailerCmd represents the thumbnailer command
var thumbnailerCmd = &cobra.Command{
	Use:   "thumbnailer",
	Short: "Run a thumbnailer service",
	Long:  `Listens for new photo events, creates a thumbnail for the photo and stores it back in the storage`,
	Run:   runThumbnailer,
}

func runThumbnailer(cmd *cobra.Command, args []string) {
	logger := initLogger()
	defer logger.Sync() // flushes buffer, if any

	thumbnailer, cleanup, err := initializeThumbnailer(cfg, &cfg.Nats, logger)
	if err != nil {
		logger.Fatal("creating thumbnailer: %v", zap.Error(err))
	}
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		logger.Info("starting thumbailer service")
		if err := thumbnailer.Start(ctx); err != nil {
			logger.Fatal("starting thumbnailer", zap.Error(err))
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	// cleanup
	if err := thumbnailer.Stop(); err != nil {
		logger.Fatal("stopping thumbnailer", zap.Error(err))
	}
}

func init() {
	rootCmd.AddCommand(thumbnailerCmd)
}
