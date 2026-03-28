package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"vps-monitor/internal/cloudflare"
	"vps-monitor/internal/cluster"
	"vps-monitor/internal/config"
	"vps-monitor/internal/engine"
	"vps-monitor/internal/model"
	"vps-monitor/internal/monitor"
	"vps-monitor/internal/notify"
	"vps-monitor/internal/store"
	"vps-monitor/internal/web"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "monitor.yaml", "path to monitor config")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.Storage.DataDir, 0o755); err != nil {
		logger.Error("create data dir failed", "error", err)
		os.Exit(1)
	}

	sqliteStore, err := store.Open(cfg.Storage.SQLitePath)
	if err != nil {
		logger.Error("open sqlite failed", "error", err)
		os.Exit(1)
	}
	defer sqliteStore.Close()

	clusterManager, err := cluster.NewManager(cfg, sqliteStore, logger)
	if err != nil {
		logger.Error("start cluster manager failed", "error", err)
		os.Exit(1)
	}

	submitter := cluster.NewSubmitter(clusterManager, cfg)
	collector := monitor.NewCollector(cfg, sqliteStore, submitter, logger)
	prober := monitor.NewProber(cfg, clusterManager, sqliteStore, submitter, logger)
	notifiers := notify.Build(cfg, logger)
	engineLoop := engine.New(cfg, sqliteStore, clusterManager, cloudflare.New(cfg), notifiers, logger)
	webServer, err := web.New(cfg, sqliteStore, clusterManager, submitter, notifiers, logger)
	if err != nil {
		logger.Error("build web server failed", "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:              cfg.Network.ListenAddr,
		Handler:           webServer.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	loopInterval := cfg.LoopInterval()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("http server listening", "addr", cfg.Network.ListenAddr, "node_id", cfg.Cluster.NodeID)
		var serveErr error
		if cfg.Network.TLSCertFile != "" && cfg.Network.TLSKeyFile != "" {
			serveErr = httpServer.ListenAndServeTLS(cfg.Network.TLSCertFile, cfg.Network.TLSKeyFile)
		} else {
			serveErr = httpServer.ListenAndServe()
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", "error", serveErr)
			stop()
		}
	}()

	if clusterManager.NeedsJoin() {
		logger.Info("dynamic membership join required", "node_id", cfg.Cluster.NodeID, "join_seeds", cfg.NormalizedJoinSeeds())
		if err := clusterManager.AutoJoin(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("automatic cluster join failed", "error", err)
			stop()
		}
	}

	if ctx.Err() == nil {
		go collector.Run(ctx, loopInterval)
		go prober.Run(ctx, loopInterval)
		go engineLoop.Run(ctx, loopInterval)
		go emitBootstrapEvent(ctx, cfg, clusterManager, logger)
	}

	<-ctx.Done()
	logger.Info("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	_ = clusterManager.Shutdown(shutdownCtx)
}

func emitBootstrapEvent(ctx context.Context, cfg *config.Config, clusterManager *cluster.Manager, logger *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !clusterManager.IsLeader() {
				continue
			}
			_, err := clusterManager.Apply(ctx, cluster.CommandEvent, model.Event{
				ID:        filepath.Base(cfg.Storage.SQLitePath) + "-leader-online",
				Kind:      "leader_online",
				Severity:  model.StatusHealthy,
				NodeID:    cfg.Cluster.NodeID,
				Title:     fmt.Sprintf("%s holds leadership", cfg.Cluster.NodeID),
				Body:      "leader loop is active and accepting replicated observations",
				CreatedAt: time.Now().UTC(),
				Meta: map[string]any{
					"listen_addr": cfg.Network.ListenAddr,
				},
			})
			if err == nil {
				return
			}
			logger.Warn("bootstrap event apply failed", "error", err)
		}
	}
}
