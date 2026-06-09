package maas

import (
	"context"
	"fmt"
	"github.com/spectrocloud/maas-client-go/maasclient"
	"image-syncer/internal/config"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func BootResources(ctx context.Context, maasURL, maasAPIKey string) (*[]maasclient.BootResource, error) {

	c := maasclient.NewAuthenticatedClientSet(maasURL, maasAPIKey)
	resourcesboots, err := c.BootResources().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boot resources: %w", err)
	}
	for _, resource := range resourcesboots {
		slog.Info("MAAS Boot Resource", slog.String("name", resource.Name()), slog.Any("id", resource.ID()))
	}
	return &resourcesboots, nil
}


func UploadMaasImage(ctx context.Context, image *config.ImageMetadata, maasURL, maasAPIKey, defaultImagePath string, pollingTimeout time.Duration) error {
	c := maasclient.NewAuthenticatedClientSet(maasURL, maasAPIKey)
	filePath := filepath.Join(defaultImagePath, image.Content)
	name := fmt.Sprintf("custom/%s",image.Content)
	architecture := image.Architecture
	sha256Hash := image.SHA256Hash
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to check local file info: %w", err)
	}
	fileSize := int(fileInfo.Size())
	slog.Info("Starting MAAS boot resource upload process", 
		slog.String("file", filePath), 
		slog.Int("size_bytes", fileSize),
	)
	builder := c.BootResources().Builder(
		name,
		architecture,
		sha256Hash,
		filePath,
		fileSize,
	)
	builder.WithTitle(image.Name).
		WithFileType("tgz").
		WithBaseImage(image.BaseImage)

	slog.Info("Registering boot resource metadata in MAAS...")

	resource, err := builder.Create(ctx)
	if err != nil {
		return fmt.Errorf("failed to register boot resource in MAAS: %w", err)
	}
	slog.Info("Resource registered successfully. Starting chunked upload...", slog.Int("resource_id", resource.ID()))
	err = resource.Upload(ctx)
	if err != nil {
		return fmt.Errorf("failed to upload boot resource file chunks: %w", err)
	}
	slog.Info("Boot resource upload completed successfully!", slog.Int("resource_id", resource.ID()))
	slog.Info("Waiting for MAAS to synchronize and compile the image...")
	timeout := time.After(pollingTimeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for boot resource %d to sync on MAAS", resource.ID())
		case <-ticker.C:
			updatedResource, err := resource.Get(ctx)
			if err != nil {
				slog.Warn("Failed to get updates from MAAS, retrying...", slog.Any("error", err))
				continue
			}
			sets := updatedResource.Sets()
			if len(sets) == 0 {
				slog.Info("Image files are being processed by MAAS...")
				continue
			}
			allComplete := true
			for version, set := range sets {
				slog.Info("Sync status",
					slog.String("version", version),
					slog.Float64("progress", set.Progress),
					slog.Bool("complete", set.Complete),
				)
				if !set.Complete {
					allComplete = false
				}
			}
			if allComplete {
				slog.Info("Boot resource successfully synced and ready for use in MAAS!", slog.Int("resource_id", resource.ID()))
				return nil
			}
		}
	}
}
