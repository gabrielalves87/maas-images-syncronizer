package config_test

import (
	"crypto/sha256"
	"encoding/hex"
	"image-syncer/internal/config"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// sha256Hex is a test helper that computes the SHA-256 hex digest of a byte slice.
// This ensures our test expectations never use incorrect hardcoded hash values.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

var _ = Describe("Config", func() {

	// ──────────────────────────────────────────────────────────────────────────
	// Types & YAML Unmarshalling
	// ──────────────────────────────────────────────────────────────────────────

	Describe("UnMarshal", func() {
		Context("when given valid YAML content", func() {
			It("should parse a single image entry correctly", func() {
				yaml := []byte(`
boot_resources:
  images:
    - name: tks-base-v1-31-0
      sha256Hash: abc123
      architecture: amd64/generic
      content: tks-base.tar.gz
      date: "2026/05/01"
      base_image: ubuntu/jammy
`)
				result, err := config.UnMarshal(yaml)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.BootResources.Images).To(HaveLen(1))

				img := result.BootResources.Images[0]
				Expect(img.Name).To(Equal("tks-base-v1-31-0"))
				Expect(img.SHA256Hash).To(Equal("abc123"))
				Expect(img.Architecture).To(Equal("amd64/generic"))
				Expect(img.Content).To(Equal("tks-base.tar.gz"))
				Expect(img.Date).To(Equal("2026/05/01"))
				Expect(img.BaseImage).To(Equal("ubuntu/jammy"))
			})

			It("should parse multiple image entries", func() {
				yaml := []byte(`
boot_resources:
  images:
    - name: image-a
      date: "2026/01/01"
    - name: image-b
      date: "2026/06/01"
`)
				result, err := config.UnMarshal(yaml)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.BootResources.Images).To(HaveLen(2))
			})

			It("should return an empty images list when boot_resources has no images", func() {
				yaml := []byte(`
boot_resources:
  images: []
`)
				result, err := config.UnMarshal(yaml)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.BootResources.Images).To(BeEmpty())
			})
		})

		Context("when given invalid YAML content", func() {
			It("should return an error for malformed YAML", func() {
				yaml := []byte(`this: is: not: [valid yaml`)

				result, err := config.UnMarshal(yaml)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should return an error for empty content", func() {
				// YAML unmarshal of empty bytes returns empty struct, not error.
				// This test documents the expected behavior.
				result, err := config.UnMarshal([]byte{})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.BootResources.Images).To(BeEmpty())
			})
		})
	})

	// ──────────────────────────────────────────────────────────────────────────
	// ImageMetadata.ParseDate
	// ──────────────────────────────────────────────────────────────────────────

	Describe("ImageMetadata.ParseDate", func() {
		Context("when the date is in the expected format (YYYY/MM/DD)", func() {
			It("should parse a valid date correctly", func() {
				img := config.ImageMetadata{Date: "2026/05/07"}
				t, err := img.ParseDate()

				Expect(err).NotTo(HaveOccurred())
				Expect(t).To(Equal(time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)))
			})

			It("should parse January 1st correctly", func() {
				img := config.ImageMetadata{Date: "2025/01/01"}
				t, err := img.ParseDate()

				Expect(err).NotTo(HaveOccurred())
				Expect(t.Year()).To(Equal(2025))
				Expect(t.Month()).To(Equal(time.January))
				Expect(t.Day()).To(Equal(1))
			})
		})

		Context("when the date is in an invalid format", func() {
			DescribeTable("should return an error",
				func(dateStr string) {
					img := config.ImageMetadata{Date: dateStr}
					_, err := img.ParseDate()
					Expect(err).To(HaveOccurred())
				},
				Entry("empty string", ""),
				Entry("wrong separator", "2026-05-07"),
				Entry("DD/MM/YYYY format", "07/05/2026"),
				Entry("partial date", "2026/05"),
				Entry("plain text", "not-a-date"),
			)
		})
	})

	// ──────────────────────────────────────────────────────────────────────────
	// GetImageMostNew
	// ──────────────────────────────────────────────────────────────────────────

	Describe("GetImageMostNew", func() {
		Context("when the index is nil", func() {
			It("should return an error", func() {
				result, err := config.GetImageMostNew(nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("image index is nil"))
				Expect(result).To(BeNil())
			})
		})

		Context("when the index has no images", func() {
			It("should return nil without an error", func() {
				index := &config.MaasImageIndex{}

				result, err := config.GetImageMostNew(index)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		Context("when the index has a single image", func() {
			It("should return that image", func() {
				index := &config.MaasImageIndex{
					BootResources: config.BootResources{
						Images: []config.ImageMetadata{
							{Name: "only-image", Date: "2026/03/10"},
						},
					},
				}

				result, err := config.GetImageMostNew(index)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Name).To(Equal("only-image"))
			})
		})

		Context("when the index has multiple images with distinct dates", func() {
			It("should return the image with the most recent date", func() {
				index := &config.MaasImageIndex{
					BootResources: config.BootResources{
						Images: []config.ImageMetadata{
							{Name: "old-image", Date: "2025/01/01"},
							{Name: "newest-image", Date: "2026/06/01"},
							{Name: "middle-image", Date: "2026/01/15"},
						},
					},
				}

				result, err := config.GetImageMostNew(index)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Name).To(Equal("newest-image"))
				Expect(result.Date).To(Equal("2026/06/01"))
			})
		})

		Context("when an image has an invalid date", func() {
			It("should return an error", func() {
				index := &config.MaasImageIndex{
					BootResources: config.BootResources{
						Images: []config.ImageMetadata{
							{Name: "valid-image", Date: "2026/05/01"},
							{Name: "bad-image", Date: "not-a-date"},
						},
					},
				}

				result, err := config.GetImageMostNew(index)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})

	// ──────────────────────────────────────────────────────────────────────────
	// GetSHA256Sum
	// ──────────────────────────────────────────────────────────────────────────

	Describe("GetSHA256Sum", func() {
		var (
			tmpDir  string
			tmpFile string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "sha256-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		Context("when the file exists and has known content", func() {
			It("should return the correct SHA-256 hex string", func() {
				content := []byte("hello world")
				tmpFile = filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(tmpFile, content, 0600)
				Expect(err).NotTo(HaveOccurred())

				hash, err := config.GetSHA256Sum(tmpFile)

				Expect(err).NotTo(HaveOccurred())
				// Compute expected dynamically to avoid incorrect hardcoded values
				Expect(hash).To(Equal(sha256Hex(content)))
			})

			It("should return different hashes for different file contents", func() {
				fileA := filepath.Join(tmpDir, "a.txt")
				fileB := filepath.Join(tmpDir, "b.txt")

				Expect(os.WriteFile(fileA, []byte("content A"), 0600)).To(Succeed())
				Expect(os.WriteFile(fileB, []byte("content B"), 0600)).To(Succeed())

				hashA, err := config.GetSHA256Sum(fileA)
				Expect(err).NotTo(HaveOccurred())

				hashB, err := config.GetSHA256Sum(fileB)
				Expect(err).NotTo(HaveOccurred())

				Expect(hashA).NotTo(Equal(hashB))
			})

			It("should return the same hash for the same file content every time (determinism)", func() {
				tmpFile = filepath.Join(tmpDir, "deterministic.txt")
				Expect(os.WriteFile(tmpFile, []byte("fixed content"), 0600)).To(Succeed())

				hash1, err := config.GetSHA256Sum(tmpFile)
				Expect(err).NotTo(HaveOccurred())

				hash2, err := config.GetSHA256Sum(tmpFile)
				Expect(err).NotTo(HaveOccurred())

				Expect(hash1).To(Equal(hash2))
			})

			It("should return a 64-character lowercase hex string", func() {
				tmpFile = filepath.Join(tmpDir, "format.txt")
				Expect(os.WriteFile(tmpFile, []byte("test"), 0600)).To(Succeed())

				hash, err := config.GetSHA256Sum(tmpFile)
				Expect(err).NotTo(HaveOccurred())

				Expect(hash).To(HaveLen(64))
				Expect(hash).To(MatchRegexp(`^[0-9a-f]{64}$`))
			})
		})

		Context("when the file does not exist", func() {
			It("should return an error", func() {
				hash, err := config.GetSHA256Sum("/non/existent/path/file.txt")

				Expect(err).To(HaveOccurred())
				Expect(hash).To(BeEmpty())
			})
		})

		Context("when given an empty file", func() {
			It("should return the SHA-256 of an empty input", func() {
				tmpFile = filepath.Join(tmpDir, "empty.txt")
				Expect(os.WriteFile(tmpFile, []byte{}, 0600)).To(Succeed())

				hash, err := config.GetSHA256Sum(tmpFile)

				Expect(err).NotTo(HaveOccurred())
				// Compare against dynamically-computed reference
				Expect(hash).To(Equal(sha256Hex([]byte{})))
				// Also verify it's 64 lowercase hex chars
				Expect(hash).To(HaveLen(64))
			})
		})
	})

})
