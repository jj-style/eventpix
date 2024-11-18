/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/events"
	"github.com/jj-style/eventpix/backend/internal/service"
	"github.com/spf13/cobra"
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
	sugar := logger.Sugar()

	db, cleanup, err := db.NewDb(cfg, logger)
	if err != nil {
		sugar.Fatalf("initialising database: %v", err)
	}
	defer cleanup()

	nc, cleanup1, err := events.NewNats(&cfg.Nats)
	if err != nil {
		sugar.Fatalf("connecting to nats: %v", err)
	}
	defer cleanup1()

	thumbnailer, err := service.NewThumbnailer(db, nc, sugar)
	if err != nil {
		sugar.Fatalf("creating thumbnailer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := thumbnailer.Start(ctx); err != nil {
			sugar.Fatalf("starting thumbnailer: %v", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	// cleanup
	if err := thumbnailer.Stop(); err != nil {
		sugar.Fatalf("stopping thumbnailer: %v", err)
	}
}

func init() {
	rootCmd.AddCommand(thumbnailerCmd)
}
