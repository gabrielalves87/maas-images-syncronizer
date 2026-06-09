package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	MaasURL          string `env:"MAAS_URL,required"`
	MaasAPIKey       string `env:"MAAS_API_KEY,required"`
	GcsCredentials   string `env:"IMAGE_DOWNLOAD_GCS_CREDENTIALS,required"`
	GcsBucket        string `env:"IMAGE_DOWNLOAD_GCS_BUCKET" envDefault:"maas-images-br"`
	GcsPrefix        string `env:"IMAGE_DOWNLOAD_GCS_PREFIX" envDefault:"ambiente-prod"`
	DefaultImagePath string `env:"DEFAULT_IMAGE_PATH" envDefault:"/tmp"`
	SyncTimeoutMinutes    int    `env:"SYNC_TIMEOUT_MINUTES" envDefault:"30"`
	PollingTimeoutMinutes int    `env:"POLLING_TIMEOUT_MINUTES" envDefault:"5"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
