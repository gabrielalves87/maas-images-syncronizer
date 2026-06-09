package main

import (
    "context"
    "image-syncer/internal/config"
    "image-syncer/internal/gcs"
    "image-syncer/internal/maas"
    "image-syncer/internal/syncer"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"
)

var (
    version = "dev"
    commit  = "unknown"
)

func main() {
    slog.Info("maas-image-syncronizer starting", slog.String("version", version), slog.String("commit", commit))

    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer cancel()
    ctx, cancelTimeout := context.WithTimeout(ctx, time.Duration(cfg.SyncTimeoutMinutes)*time.Minute)
    defer cancelTimeout()

    gcsClient, err := bucket.BucketClient(ctx, cfg.GcsCredentials)
    if err != nil {
        slog.Error("failed to create GCS client", "error", err)
        os.Exit(1)
    }
    defer gcsClient.Close()

    source := bucket.NewAdapter(gcsClient, cfg.GcsBucket, cfg.GcsPrefix)
    target := maas.NewAdapter(cfg.MaasURL, cfg.MaasAPIKey, time.Duration(cfg.PollingTimeoutMinutes)*time.Minute)

    s := syncer.New(source, target, cfg.DefaultImagePath)
    err = s.Run(ctx)
	if err != nil {
        slog.Error("sync failed", "error", err)
        os.Exit(1)
    }
}
