package cmd

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/jj-style/eventpix/backend/internal/server"
	"github.com/jj-style/eventpix/backend/internal/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the main eventpix server",
	Run:   run,
}

func run(cmd *cobra.Command, args []string) {
	var logger *zap.Logger
	if cfg.Server.Environment == "development" {
		logger, _ = zap.NewDevelopment(zap.AddStacktrace(zap.DPanicLevel))
	} else {
		logger, _ = zap.NewProduction(zap.AddStacktrace(zap.DPanicLevel))
	}
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	db, cleanup, err := db.NewDb(cfg, logger)
	if err != nil {
		log.Fatalf("initialising database: %v", err)
	}
	defer cleanup()
	svc := service.NewPictureServiceServer(logger, db)
	srv := server.NewServer(cfg, svc, logger)

	// fmt.Println("Listening on", cfg.Server.Address)
	sugar.Infof("listening on %s", cfg.Server.Address)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP listen and serve: %v", err)
		}
	}()

	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP shutdown: %v", err) //nolint:gocritic
	}
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
