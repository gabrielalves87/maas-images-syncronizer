# MAAS Image Synchronizer

A production-ready Go CLI tool that automates the synchronization of custom OS images from a Google Cloud Storage (GCS) bucket to a [MAAS (Metal as a Service)](https://maas.io/) server.

## Overview

`image-syncer` reads an image index (`boot_resources.yaml`) from a GCS bucket, selects the most recently built image, downloads it to a local path, verifies its integrity via SHA-256 checksum, and registers and uploads it to a MAAS server. After uploading, it polls the MAAS API until the image is fully synchronized and ready for deployment on bare-metal nodes.

## Architecture

```
GCS Bucket
└── <prefix>/
    ├── boot_resources.yaml     ← Image index (source of truth)
    └── <image>.tar.gz          ← Image tarballs

                │
                ▼
        image-syncer (CLI)
                │
       ┌────────┴────────┐
       │                 │
  Download &         Upload to
  SHA-256 verify     MAAS API
                         │
                         ▼
                  MAAS Boot Resources
                  (polling until synced)
```

### Package Structure

```
.
├── cmd/
│   └── image-syncer/
│       └── main.go             # Application entrypoint
├── internal/
│   ├── config/
│   │   ├── env.go              # Configuration loading via caarlos0/env
│   │   ├── types.go            # Domain types (MaasImageIndex, ImageMetadata)
│   │   ├── date.go             # Image date sorting logic (GetImageMostNew)
│   │   └── sha256.go           # SHA-256 file integrity verification
│   ├── gcs/
│   │   └── google-storage.go   # GCS client, image download, index reading
│   └── maas/
│       └── bootresources.go    # MAAS API: list, upload, and polling logic
├── boot_resources.yaml         # Example image index file (stored in GCS)
├── Dockerfile                  # Multi-stage build for the binary
└── Makefile                    # Build, lint, fmt, and test targets
```

## Prerequisites

- **Go** `>= 1.26` (declared in `go.mod`)
- **Docker** (for containerized execution)
- A running **MAAS** server (v3.x+) with API access
- A **Google Cloud Storage** bucket with:
  - A `boot_resources.yaml` index file
  - Image tarballs (`.tar.gz`) in the same prefix

## Configuration

All configuration is done via **environment variables**. The tool will fail fast on startup if required variables are missing.

| Variable | Required | Default | Description |
|---|---|---|---|
| `MAAS_URL` | ✅ | — | Full MAAS API URL (e.g. `http://maas.example.com/MAAS/api/2.0`) |
| `MAAS_API_KEY` | ✅ | — | MAAS API key (format: `consumer:token:secret`) |
| `IMAGE_DOWNLOAD_GCS_CREDENTIALS` | ✅ | — | Path to GCS Service Account JSON key file |
| `IMAGE_DOWNLOAD_GCS_BUCKET` | ❌ | `maas-images-br` | GCS bucket name where images are stored |
| `IMAGE_DOWNLOAD_GCS_PREFIX` | ❌ | `ambiente-prod` | Path prefix inside the bucket (e.g. `ambiente-prod`) |
| `DEFAULT_IMAGE_PATH` | ❌ | `/tmp` | Local directory for downloading images before upload |

## The `boot_resources.yaml` Index File

This file lives inside the GCS bucket at `<GCS_PREFIX>/boot_resources.yaml`. It acts as the source of truth for which images are available.

```yaml
boot_resources:
  images:
    - name: base-v1-31-13-20260507
      sha256Hash: 7671ee154cae4d8850ba8b23d7026ea0f6651de368044b2ced7e2499def5b04d
      architecture: amd64/generic
      content: base-v1-31-13-20260507.tar.gz
      k8s_version: v1-31-13
      so: Ubuntu
      versao: "22.04.04"
      date: "2026/05/07"
      base_image: ubuntu/jammy
```

The tool automatically selects the entry with the **most recent `date`** field.

## Running Locally

### 1. Build the binary

```bash
go build -o bin/image-syncer cmd/image-syncer/main.go
```

### 2. Set environment variables

```bash
export MAAS_URL="http://your-maas-server/MAAS/api/2.0"
export MAAS_API_KEY="consumer_key:token_key:token_secret"
export IMAGE_DOWNLOAD_GCS_CREDENTIALS="/path/to/service-account.json"
export IMAGE_DOWNLOAD_GCS_BUCKET="my-images-bucket"
export IMAGE_DOWNLOAD_GCS_PREFIX="prod/images"
export DEFAULT_IMAGE_PATH="/tmp"
```

### 3. Run

```bash
./bin/image-syncer
```

## Running with Docker

### Build the image

```bash
docker build -t maas-image-syncer:latest .
```

### Run the container

Mount your GCS credentials JSON file into the container and pass environment variables:

```bash
docker run --rm \
  -e MAAS_URL="http://your-maas-server/MAAS/api/2.0" \
  -e MAAS_API_KEY="consumer_key:token_key:token_secret" \
  -e IMAGE_DOWNLOAD_GCS_CREDENTIALS="/credentials/service-account.json" \
  -e IMAGE_DOWNLOAD_GCS_BUCKET="my-images-bucket" \
  -e IMAGE_DOWNLOAD_GCS_PREFIX="prod/images" \
  -v "/path/to/local/service-account.json:/credentials/service-account.json:ro" \
  maas-image-syncer:latest
```

## Execution Flow

The tool executes the following steps sequentially:

1. **Load & validate configuration** from environment variables.
2. **Initialize GCS client** using the provided Service Account credentials.
3. **Read `boot_resources.yaml`** from the GCS bucket.
4. **Select the most recent image** by comparing the `date` field across all entries.
5. **Download the image tarball** from GCS to `DEFAULT_IMAGE_PATH`, retrying on SHA-256 mismatch.
6. **Verify image integrity** using SHA-256 checksum.
7. **Fetch existing MAAS boot resources** to check if the image already exists.
8. **Skip upload** if the image already exists in MAAS.
9. **Register & upload** the image to MAAS if it doesn't exist yet.
10. **Poll MAAS** (every 5 seconds, up to 5 minutes) until all image file sets report `complete: true`.

## Development

### Available Makefile targets

```bash
make build       # Compile the binary to bin/
make fmt         # Run go fmt
make vet         # Run go vet
make lint        # Run golangci-lint
make lint-fix    # Run golangci-lint with auto-fix
make test        # Run all tests
```

### Key Dependencies

| Package | Purpose |
|---|---|
| `cloud.google.com/go/storage` | Google Cloud Storage client |
| `github.com/spectrocloud/maas-client-go` | MAAS API client |
| `github.com/caarlos0/env/v11` | Environment variable parsing |
| `gopkg.in/yaml.v3` | YAML parsing for image index |
| `google.golang.org/api/option` | GCS authentication options |

## Logging

All logs are structured using Go's standard `log/slog` package (Go 1.21+) and are printed to `stdout` in a human-readable key-value format. In production, you can redirect these to a log aggregation system.

Example log output:

```
2026/06/03 10:00:00 INFO Boot resources fetched successfully.
2026/06/03 10:00:00 INFO Image: name=base-v1-31-13-20260507
2026/06/03 10:00:00 INFO Most new image date: date=2026/05/07
2026/06/03 10:00:00 INFO downloading image
2026/06/03 10:00:45 INFO sha256sum match
2026/06/03 10:00:45 INFO Image downloaded successfully.
2026/06/03 10:00:45 INFO Image not found in MAAS, uploading... image_name=custom/base-v1-31-13-20260507
2026/06/03 10:01:30 INFO Boot resource upload completed successfully! resource_id=12
2026/06/03 10:01:35 INFO Sync status version=20260507 progress=0.5 complete=false
2026/06/03 10:01:40 INFO Boot resource successfully synced and ready for use in MAAS! resource_id=12
```

## Security

- GCS authentication uses `option.WithAuthCredentialsFile(option.ServiceAccount, path)` — the modern, type-safe API that replaces the deprecated `WithCredentialsFile`.
- The credential file path is never hardcoded; it is always passed via environment variable.
- Image integrity is always verified with SHA-256 before uploading to MAAS.

## License

This project is for internal use. Refer to your organization's licensing policy.
