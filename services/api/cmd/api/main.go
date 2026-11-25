package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/db"
	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../../../.env")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config_error", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db_error", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	store, err := storage.NewFromDriver(cfg.StorageDriver, cfg.StorageLocalPath, cfg.StorageEndpoint, cfg.StorageBucket, cfg.StorageAccessKey, cfg.StorageSecretKey)
	if err != nil {
		logger.Error("storage_error", "error", err)
		os.Exit(1)
	}

	router, worker, extras := httpapi.NewRouterWithExtras(httpapi.Deps{
		Cfg:     cfg,
		DB:      pool,
		Logger:  logger,
		Storage: store,
	})

	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go worker.Run(workerCtx)
	go runSystemHealthLoops(workerCtx, extras)

	go func() {
		logger.Info("api_listening", "addr", cfg.APIAddr, "env", cfg.AppEnv, "email_mode", cfg.EmailMode, "ai_configured", cfg.AIAPIKey != "")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	workerCancel()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Info("api_stopped")
}

func runSystemHealthLoops(ctx context.Context, extras httpapi.RouterExtras) {
	if extras.Checker == nil || extras.Health == nil {
		return
	}
	run := func() {
		_, _ = extras.Checker.RunChecks(context.Background())
		extras.Health.CleanupRetention(context.Background())
		if extras.Alerter != nil {
			_ = extras.Alerter.EvaluateRules(context.Background())
		}
	}
	// Initial delay so listen is up; then every 2 minutes.
	select {
	case <-ctx.Done():
		return
	case <-time.After(3 * time.Second):
	}
	run()
	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}
