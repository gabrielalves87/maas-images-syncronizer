package main

import (
	"context"
	"image-syncer/internal/config"
	"image-syncer/internal/gcs"
	"image-syncer/internal/maas"
	"log/slog"
	"os"
)

func main() {
	ctx:= context.Background()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load environment variables:", "error", err)
		os.Exit(1)
	}
	cli, err := bucket.BucketClient(ctx,cfg.GcsCredentials)
	if err != nil {
		slog.Error("Error fetching bucket client:", "error", err)
		os.Exit(1)
	}
	defer cli.Close()
	index, err := bucket.ReadBootResourcesFile(ctx,cli, cfg.GcsBucket, cfg.GcsPrefix)
	if err != nil {
		slog.Error("failed to read boot resources file:", "error", err)
		os.Exit(1)
	}
	slog.Info("Boot resources fetched successfully.")
	for _, image := range index.BootResources.Images {
		slog.Info("Image:", slog.String("name", image.Name))
	}
	imagemostNew, err := config.GetImageMostNew(index)
	if err != nil {
		slog.Error("failed to get most new image:", "error", err)
		os.Exit(1)
	}
	if imagemostNew == nil {
    slog.Info("No images found in boot resources index.")
    return
}
	slog.Info("Most new image date:", slog.Any("date", imagemostNew.Date))
	_, err = bucket.DownloadMaasImage(ctx,cli, imagemostNew, cfg.GcsBucket, cfg.GcsPrefix, cfg.DefaultImagePath)
	if err != nil {
		slog.Error("failed to download MAAS image:", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Image downloaded successfully.")
	slog.Info("Fetching boot resources...")
	bootResources, err := maas.BootResources(ctx,cfg.MaasURL, cfg.MaasAPIKey)
	if err != nil {
		slog.Error("Error fetching boot resources:", slog.Any("error", err))
		os.Exit(1)
	}
	existingResources := make(map[string]bool)
	for _, resource := range *bootResources {
		existingResources[resource.Name()] = true
	}

	imageName := imagemostNew.Name
	if existingResources[imageName] {
		slog.Info("Image already exists in MAAS, skipping upload", slog.String("image_name", imageName))
	} else {
		slog.Info("Image not found in MAAS, uploading...", slog.String("image_name", imageName))
		err = maas.UploadMaasImage(ctx,imagemostNew, cfg.MaasURL, cfg.MaasAPIKey, cfg.DefaultImagePath)
		if err != nil {
			slog.Error("failed to upload MAAS image:", slog.Any("error", err))
			os.Exit(1)
		}
		slog.Info("Image uploaded successfully")
	}

}
