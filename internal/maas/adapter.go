package maas

import (
    "context"
    "image-syncer/internal/config"
    "time"
)

type Adapter struct {
    maasURL        string
    maasAPIKey     string
    pollingTimeout time.Duration
}

func NewAdapter(maasURL, maasAPIKey string, pollingTimeout time.Duration) *Adapter {
    return &Adapter{maasURL: maasURL, maasAPIKey: maasAPIKey, pollingTimeout: pollingTimeout}
}

// ListExisting implementa syncer.ImageTarget
func (a *Adapter) ListExisting(ctx context.Context) (map[string]bool, error) {
    resources, err := BootResources(ctx, a.maasURL, a.maasAPIKey)
    if err != nil {
        return nil, err
    }
    existing := make(map[string]bool, len(*resources))
    for _, r := range *resources {
        existing[r.Name()] = true
    }
    return existing, nil
}

// Upload implementa syncer.ImageTarget
func (a *Adapter) Upload(ctx context.Context, img *config.ImageMetadata, imagePath string) error {
    return UploadMaasImage(ctx, img, a.maasURL, a.maasAPIKey, imagePath, a.pollingTimeout)
}