package imagedownloader

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	_ "golang.org/x/image/webp"
)

// ImageDownloader is a service for downloading images
type ImageDownloader struct {
	DestinationFolder string
	Concurrency       int
}

// ImageDownloadResult represents the result of a single image download
type ImageDownloadResult struct {
	URL      string
	FilePath string
	Error    error
	Width    int
	Height   int
}

type ImageDetails struct {
	URL      string
	SubDir   string
	FileName string
}

// NewImageDownloader creates a new ImageDownloader instance
func NewImageDownloader(destFolder string, concurrency int) *ImageDownloader {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &ImageDownloader{
		DestinationFolder: destFolder,
		Concurrency:       concurrency,
	}
}

// DownloadImages downloads images from the given URLs and saves them to the destination folder
func (id *ImageDownloader) DownloadImages(images []ImageDetails) []ImageDownloadResult {
	results := make([]ImageDownloadResult, len(images))
	semaphore := make(chan struct{}, id.Concurrency)
	var wg sync.WaitGroup

	for i, image := range images {
		wg.Add(1)
		go func(i int, details ImageDetails) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			filePath, err := id.downloadSingleImage(details)
			w, h, err := getImageDimensions(filePath)
			if err != nil {
				fmt.Println(err)
				os.Exit(69)
			}
			results[i] = ImageDownloadResult{
				URL:      details.URL,
				FilePath: filePath,
				Error:    err,
				Width:    w,
				Height:   h,
			}
		}(i, image)
	}

	wg.Wait()
	return results
}

// downloadSingleImage downloads a single image from the given URL and saves it to the destination folder
func (id *ImageDownloader) downloadSingleImage(details ImageDetails) (string, error) {
	client := &http.Client{}

	resp, err := client.Get(details.URL)
	if err != nil {
		return "", fmt.Errorf("error downloading image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	dirPath := filepath.Join(id.DestinationFolder, details.SubDir)
	filePath := filepath.Join(dirPath, details.FileName)

	err = os.MkdirAll(dirPath, 0764)
	if err != nil {
		fmt.Println(err)
		os.Exit(99)
	}
	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error saving image: %v", err)
	}

	return filePath, nil
}

func getImageDimensions(filename string) (int, int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}
