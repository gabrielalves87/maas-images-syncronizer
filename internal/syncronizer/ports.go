package syncronizer

import (
    "context"
    "image-syncer/internal/config"
)

type ImageSource interface {
    ListImages(ctx context.Context) (*config.MaasImageIndex, error)
    Download(ctx context.Context, img *config.ImageMetadata, destPath string) error
}

type ImageTarget interface {
    ListExisting(ctx context.Context) (map[string]bool, error)
    Upload(ctx context.Context, img *config.ImageMetadata, imagePath string) error
}
