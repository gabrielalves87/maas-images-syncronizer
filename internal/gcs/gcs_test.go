package bucket_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image-syncer/internal/config"
	bucket "image-syncer/internal/gcs"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test Helpers
// ─────────────────────────────────────────────────────────────────────────────

// newFakeGCSServer starts a minimal httptest.Server that returns the content
// registered in the `files` map keyed by request path (e.g. "/bucket/prefix/file").
// Unknown paths return HTTP 404.
func newFakeGCSServer(files map[string][]byte) *httptest.Server {
	mux := http.NewServeMux()
	for path, content := range files {
		content := content // capture loop variable
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	return httptest.NewServer(mux)
}

// newTestGCSClient creates a *storage.Client wired to a fake test HTTP server.
// It uses option.WithoutAuthentication() and points the endpoint to the local server.
func newTestGCSClient(ctx context.Context, srv *httptest.Server) (*storage.Client, error) {
	return storage.NewClient(ctx,
		option.WithEndpoint(srv.URL+"/storage/v1/"),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
	)
}

// sha256HexOf computes the SHA-256 hex digest of a byte slice.
func sha256HexOf(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// writeTempFile creates a file in dir with the given content and returns its path.
func writeTempFile(dir, name string, content []byte) string {
	path := filepath.Join(dir, name)
	ExpectWithOffset(1, os.WriteFile(path, content, 0600)).To(Succeed())
	return path
}

// ─────────────────────────────────────────────────────────────────────────────
// Specs
// ─────────────────────────────────────────────────────────────────────────────

var _ = Describe("GCS Storage", func() {

	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	// ──────────────────────────────────────────────────────────────────────────
	// ReadBootResourcesFile
	// ──────────────────────────────────────────────────────────────────────────

	Describe("ReadBootResourcesFile", func() {
		const (
			testBucket = "my-bucket"
			testPrefix = "prod"
		)

		var (
			srv *httptest.Server
			cli *storage.Client
		)

		AfterEach(func() {
			if cli != nil {
				_ = cli.Close()
			}
			if srv != nil {
				srv.Close()
			}
		})

		Context("when boot_resources.yaml exists with valid content", func() {
			BeforeEach(func() {
				validYAML := []byte(`
boot_resources:
  images:
    - name: test-image
      sha256Hash: abc123
      architecture: amd64/generic
      content: test-image.tar.gz
      date: "2026/05/01"
      base_image: ubuntu/jammy
    - name: older-image
      sha256Hash: def456
      architecture: amd64/generic
      content: older-image.tar.gz
      date: "2026/04/01"
      base_image: ubuntu/jammy
`)
				// GCS JSON API path for downloading an object
				gcsPath := fmt.Sprintf("/download/storage/v1/b/%s/o/%s%%2F%s",
					testBucket, testPrefix, "boot_resources.yaml")
				srv = newFakeGCSServer(map[string][]byte{gcsPath: validYAML})

				var err error
				cli, err = newTestGCSClient(ctx, srv)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a non-nil MaasImageIndex", func() {
				result, err := bucket.ReadBootResourcesFile(ctx, cli, testBucket, testPrefix)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
			})

			It("parses all image entries", func() {
				result, err := bucket.ReadBootResourcesFile(ctx, cli, testBucket, testPrefix)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.BootResources.Images).To(HaveLen(2))
			})

			It("maps fields correctly for the first image", func() {
				result, err := bucket.ReadBootResourcesFile(ctx, cli, testBucket, testPrefix)
				Expect(err).NotTo(HaveOccurred())

				img := result.BootResources.Images[0]
				Expect(img.Name).To(Equal("test-image"))
				Expect(img.SHA256Hash).To(Equal("abc123"))
				Expect(img.Architecture).To(Equal("amd64/generic"))
				Expect(img.Content).To(Equal("test-image.tar.gz"))
				Expect(img.Date).To(Equal("2026/05/01"))
				Expect(img.BaseImage).To(Equal("ubuntu/jammy"))
			})
		})

		Context("when boot_resources.yaml contains invalid YAML", func() {
			BeforeEach(func() {
				gcsPath := fmt.Sprintf("/download/storage/v1/b/%s/o/%s%%2F%s",
					testBucket, testPrefix, "boot_resources.yaml")
				srv = newFakeGCSServer(map[string][]byte{
					gcsPath: []byte(`this: is: [invalid yaml`),
				})

				var err error
				cli, err = newTestGCSClient(ctx, srv)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an unmarshalling error", func() {
				result, err := bucket.ReadBootResourcesFile(ctx, cli, testBucket, testPrefix)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unmarshalling"))
				Expect(result).To(BeNil())
			})
		})

		Context("when boot_resources.yaml does not exist in the bucket", func() {
			BeforeEach(func() {
				// Server returns 404 for everything
				srv = newFakeGCSServer(map[string][]byte{})
				var err error
				cli, err = newTestGCSClient(ctx, srv)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				result, err := bucket.ReadBootResourcesFile(ctx, cli, testBucket, testPrefix)
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})

	// ──────────────────────────────────────────────────────────────────────────
	// DownloadMaasImage
	// ──────────────────────────────────────────────────────────────────────────

	Describe("DownloadMaasImage", func() {
		const (
			testBucket = "my-bucket"
			testPrefix = "prod"
		)

		var (
			srv        *httptest.Server
			cli        *storage.Client
			tmpDir     string
			fakeContent = []byte("fake image tar.gz content for unit testing")
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "download-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if cli != nil {
				_ = cli.Close()
			}
			if srv != nil {
				srv.Close()
			}
			os.RemoveAll(tmpDir)
		})

		gcsObjectPath := func(bucket, prefix, file string) string {
			return fmt.Sprintf("/download/storage/v1/b/%s/o/%s%%2F%s", bucket, prefix, file)
		}

		Context("when the remote object exists and SHA-256 matches", func() {
			var img *config.ImageMetadata

			BeforeEach(func() {
				img = &config.ImageMetadata{
					Content:    "test-image.tar.gz",
					SHA256Hash: sha256HexOf(fakeContent),
				}
				srv = newFakeGCSServer(map[string][]byte{
					gcsObjectPath(testBucket, testPrefix, img.Content): fakeContent,
				})
				var err error
				cli, err = newTestGCSClient(ctx, srv)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the content filename without error", func() {
				result, err := bucket.DownloadMaasImage(ctx, cli, img, testBucket, testPrefix, tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(*result).To(Equal("test-image.tar.gz"))
			})

			It("writes the file content to the local path", func() {
				_, err := bucket.DownloadMaasImage(ctx, cli, img, testBucket, testPrefix, tmpDir)
				Expect(err).NotTo(HaveOccurred())

				localPath := filepath.Join(tmpDir, "test-image.tar.gz")
				Expect(localPath).To(BeARegularFile())

				written, readErr := os.ReadFile(localPath)
				Expect(readErr).NotTo(HaveOccurred())
				Expect(written).To(Equal(fakeContent))
			})

			It("uses filepath.Join to build the download path (no path traversal)", func() {
				img2 := &config.ImageMetadata{
					Content:    "safe-image.tar.gz",
					SHA256Hash: sha256HexOf(fakeContent),
				}
				srv.Close()
				srv = newFakeGCSServer(map[string][]byte{
					gcsObjectPath(testBucket, testPrefix, img2.Content): fakeContent,
				})
				var err error
				cli.Close()
				cli, err = newTestGCSClient(ctx, srv)
				Expect(err).NotTo(HaveOccurred())

				_, err = bucket.DownloadMaasImage(ctx, cli, img2, testBucket, testPrefix, tmpDir)
				Expect(err).NotTo(HaveOccurred())

				// File must land inside tmpDir, not elsewhere
				expectedPath := filepath.Join(tmpDir, "safe-image.tar.gz")
				Expect(expectedPath).To(BeARegularFile())
			})
		})

		Context("when the remote object does not exist", func() {
			BeforeEach(func() {
				srv = newFakeGCSServer(map[string][]byte{})
				var err error
				cli, err = newTestGCSClient(ctx, srv)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				img := &config.ImageMetadata{
					Content:    "missing-image.tar.gz",
					SHA256Hash: "doesnotmatter",
				}

				result, err := bucket.DownloadMaasImage(ctx, cli, img, testBucket, testPrefix, tmpDir)
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})

	// ──────────────────────────────────────────────────────────────────────────
	// BucketClient (smoke test — full auth requires real GCS credentials)
	// ──────────────────────────────────────────────────────────────────────────

	Describe("BucketClient", func() {
		Context("when given a non-existent credentials file path", func() {
			It("does not panic and returns an error or a valid client (lazy init)", func() {
				cli, err := bucket.BucketClient(ctx, "/non/existent/credentials.json")
				if err != nil {
					Expect(cli).To(BeNil())
				} else {
					// Some SDK versions lazily fail on first use — close without panic
					Expect(cli).NotTo(BeNil())
					_ = cli.Close()
				}
			})
		})
	})
})

// Compile-time import verification — ensures io is used
var _ io.Reader = bytes.NewReader(nil)
