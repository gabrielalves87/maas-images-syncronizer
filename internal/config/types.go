package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type MaasImageIndex struct {
	BootResources BootResources `yaml:"boot_resources"`
}

type BootResources struct {
	Images []ImageMetadata `yaml:"images"`
}

type ImageMetadata struct {
	Name         string `yaml:"name"`
	MD5Sum       string `yaml:"md5sum"`
	SHA256Hash   string `yaml:"sha256Hash"`
	Architecture string `yaml:"architecture"`
	Content      string `yaml:"content"` // Name of the .tar.gz file (e.g. tks-image.tar.gz)
	K8sVersion   string `yaml:"k8s_version"`
	So           string `yaml:"so"`
	Version      string `yaml:"version"`
	Date         string `yaml:"date"`
	Subarches    string `yaml:"subarches"`
	BaseImage    string `yaml:"base_image"`
}

func (i ImageMetadata) ParseDate() (time.Time, error) {
	return time.Parse(dateLayout, i.Date)
}

func UnMarshal(content []byte) (*MaasImageIndex, error) {
	var u MaasImageIndex
	err := yaml.Unmarshal(content, &u)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML content: %w", err)
	}
	return &u, nil
}

type ImageTask struct {
	Name         string `json:"name"`
	Architecture string `json:"architecture"`
	TarballURL   string `json:"tarball_url"` // e.g. gs://bucket-name/filename.tar.gz
}
