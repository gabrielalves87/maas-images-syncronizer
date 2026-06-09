package syncronizer

import (
    "context"
    "image-syncer/internal/config"
    "log/slog"
)

type Syncer struct {
    source    ImageSource
    target    ImageTarget
    imagePath string
}

func New(source ImageSource, target ImageTarget, imagePath string) *Syncer {
    return &Syncer{source: source, target: target, imagePath: imagePath}
}

func (s *Syncer) Run(ctx context.Context) error {
    index, err := s.source.ListImages(ctx)
    if err != nil {
        return err
    }

    for _, img := range index.BootResources.Images {
        slog.Info("found image", slog.String("name", img.Name))
    }

    latest, err := config.GetImageMostNew(index)
    if err != nil {
        return err
    }
    if latest == nil {
        slog.Info("no images found in boot resources index")
        return nil
    }

    slog.Info("latest image", slog.String("date", latest.Date), slog.String("name", latest.Name))

    if err := s.source.Download(ctx, latest, s.imagePath); err != nil {
        return err
    }
    slog.Info("image downloaded successfully")

    existing, err := s.target.ListExisting(ctx)
    if err != nil {
        return err
    }

    if existing[latest.Name] {
        slog.Info("image already exists in MAAS, skipping upload", slog.String("name", latest.Name))
        return nil
    }

    slog.Info("uploading image to MAAS", slog.String("name", latest.Name))
    if err := s.target.Upload(ctx, latest, s.imagePath); err != nil {
        return err
    }
    slog.Info("image uploaded successfully")
    return nil
}
