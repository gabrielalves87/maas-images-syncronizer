package config

import (
	"fmt"
	"time"
)

const dateLayout = "2006/01/02"

func GetImageMostNew(image *MaasImageIndex) (*ImageMetadata, error) {
	if image == nil {
        return nil, fmt.Errorf("image index is nil")
    }
	var mostRecent *ImageMetadata
	var mostRecentTime time.Time
	for i := range image.BootResources.Images {
		t, err := image.BootResources.Images[i].ParseDate()
		if err != nil {
			return nil, fmt.Errorf("erro ao parsear data '%s': %w", image.BootResources.Images[i].Date, err)
		}

		if mostRecent == nil || t.After(mostRecentTime) {
			mostRecent = &image.BootResources.Images[i]
			mostRecentTime = t
		}
	}

	return mostRecent, nil
}
