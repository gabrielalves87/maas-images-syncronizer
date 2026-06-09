package bucket

import (
	"context"
	"fmt"
	"image-syncer/internal/config"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func BucketClient(ctx context.Context, credentialPath string) (*storage.Client, error) {
	clientOption := option.WithAuthCredentialsFile(
		option.ServiceAccount,
		credentialPath,
	)
	return storage.NewClient(ctx, clientOption)
}

func ReadBootResourcesFile(ctx context.Context, cli *storage.Client, bucketName, prefix string) (*config.MaasImageIndex, error) {
	bkt := cli.Bucket(bucketName)
	bootrsrc := prefix + "/boot_resources.yaml"
	bootrsrcContent, err := bkt.Object(bootrsrc).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading boot_resources.yaml: %w", err)
	}
	defer bootrsrcContent.Close()
	body, err := io.ReadAll(bootrsrcContent)
	if err != nil {
		return nil, fmt.Errorf("error reading boot_resources.yaml: %w", err)
	}
	bodyun, err := config.UnMarshal(body)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling boot_resources.yaml: %w", err)
	}
	return bodyun, nil
}

func DownloadMaasImage(ctx context.Context, cli *storage.Client, img *config.ImageMetadata, bucketName, prefix, defaultImagePath string) (*string, error) {
	bkt := cli.Bucket(bucketName)
	imgscr := fmt.Sprintf("%s/%s", prefix, img.Content)
	filepath := filepath.Join(defaultImagePath, img.Content)
	for {
		imgSourceContent, err := bkt.Object(imgscr).NewReader(ctx)
		if err != nil {
			return nil, fmt.Errorf("error reading boot_resources.yaml: %w", err)
		}
		slog.Info("downloading image")
		localFile, err := os.Create(filepath)
		if err != nil {
			imgSourceContent.Close()
			return nil, fmt.Errorf("error creating local file: %w", err)
		}
		_, copyErr := io.Copy(localFile, imgSourceContent)
		imgSourceContent.Close()
		localFile.Close()
		if copyErr != nil {
			err = os.Remove(filepath)
			return nil, fmt.Errorf("error copying image content to local file: %w", copyErr)
		}
		sha256sum, err := config.GetSHA256Sum(filepath)
		if err != nil {
			return nil, fmt.Errorf("error getting sha256sum: %w", err)
		}
		if img.SHA256Hash == sha256sum {
			slog.Info("sha256sum match")
			break
		}
		err = os.Remove(filepath)
		if err != nil {
			return nil, fmt.Errorf("error removing local file: %w", err)
		}
		slog.Info("sha256sum do not match \n restarting the download process", slog.String("sha256sum", sha256sum))
	}
	return &img.Content, nil
}
