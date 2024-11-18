/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
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

	thumbnailer := service.NewThumbnailer(db, nc, sugar)

	var cleanup3 func()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sugar.Info("starting thumbnailer")
		cleanup3, err = thumbnailer.Run()
		if err != nil {
			sugar.Fatalf("error starting thumbnailer: %v", err)
		}
	}()
	<-signals
	cleanup3()
	// cleanup

}

func init() {
	rootCmd.AddCommand(thumbnailerCmd)
}
