package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/config"
	"github.com/Dyuzhovsergey/gophermart/internal/httpserver"
	"github.com/Dyuzhovsergey/gophermart/internal/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Parse()

	zapLogger, err := logger.Init()
	if err != nil {
		// логгера нет — остаётся только аварийный выход
		panic(err)
	}
	defer func() { _ = zapLogger.Sync() }()

	router := httpserver.NewRouter(httpserver.Deps{
		Logger: zapLogger,
	})

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	go func() {
		zapLogger.Info("starting server", zap.String("addr", cfg.RunAddress))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("listen failed", zap.Error(err))
		}
	}()

	<-stop
	zapLogger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("shutdown error", zap.Error(err))
	}

	zapLogger.Info("stopped")
}
