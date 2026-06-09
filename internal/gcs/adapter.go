package bucket

import (
    "cloud.google.com/go/storage"
    "context"
    "image-syncer/internal/config"
)

type Adapter struct {
    client     *storage.Client
    bucketName string
    prefix     string
}

func NewAdapter(client *storage.Client, bucketName, prefix string) *Adapter {
    return &Adapter{client: client, bucketName: bucketName, prefix: prefix}
}

// ListImages implementa syncer.ImageSource
func (a *Adapter) ListImages(ctx context.Context) (*config.MaasImageIndex, error) {
    return ReadBootResourcesFile(ctx, a.client, a.bucketName, a.prefix)
}

// Download implementa syncer.ImageSource
func (a *Adapter) Download(ctx context.Context, img *config.ImageMetadata, destPath string) error {
    _, err := DownloadMaasImage(ctx, a.client, img, a.bucketName, a.prefix, destPath)
    return err
}