package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/icegreg/chat-smpl/services/health/internal/checker"
	"github.com/icegreg/chat-smpl/services/health/internal/config"
	"github.com/icegreg/chat-smpl/services/health/internal/handler"
)

func main() {
	// Setup logger
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Load config
	cfg := config.Load()
	log.Info("config loaded",
		"http_port", cfg.HTTPPort,
		"check_interval", cfg.CheckInterval,
		"chat_service", cfg.ChatServiceAddr,
		"centrifugo_ws", cfg.CentrifugoWSURL,
	)

	// Create checker
	chk, err := checker.NewChecker(cfg, log)
	if err != nil {
		log.Error("failed to create checker", "error", err)
		os.Exit(1)
	}

	// Start checker background loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := chk.Start(ctx); err != nil {
		log.Error("failed to start checker", "error", err)
		os.Exit(1)
	}

	// Create HTTP handler
	h := handler.NewHandler(chk, log)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Routes
	r.Get("/health", h.Health)
	r.Get("/api/health/system", h.SystemHealth)           // Always 200, status in body
	r.Get("/api/health/system/check", h.SystemHealthCheck) // HTTP status reflects health
	r.Handle("/metrics", promhttp.Handler())

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("starting HTTP server", "addr", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down...")

	// Graceful shutdown
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}

	if err := chk.Stop(); err != nil {
		log.Error("checker stop error", "error", err)
	}

	log.Info("shutdown complete")
}
