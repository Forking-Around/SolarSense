package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Forking-Around/SolarSense/internal/config"
	"github.com/Forking-Around/SolarSense/internal/web"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := cfg.Validate(); err != nil {
		log.Error("configuration", "error", err)
		os.Exit(1)
	}
	app, err := web.New(cfg, log)
	if err != nil {
		log.Error("application", "error", err)
		os.Exit(1)
	}
	srv := &http.Server{Addr: cfg.Addr(), Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Info("SolarSense listening", "address", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server", "error", err)
			os.Exit(1)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Error("shutdown", "error", err)
	}
}
