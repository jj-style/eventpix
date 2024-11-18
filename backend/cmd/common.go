package cmd

import "go.uber.org/zap"

func initLogger() *zap.Logger {
	var logger *zap.Logger
	if cfg.Server.Environment == "development" {
		logger, _ = zap.NewDevelopment(zap.AddStacktrace(zap.DPanicLevel))
	} else {
		logger, _ = zap.NewProduction(zap.AddStacktrace(zap.DPanicLevel))
	}
	return logger
}
