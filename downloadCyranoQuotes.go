package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func downloadAndSave(pageNum int, folderPath string) error {
	// Create folder if it doesn't exist
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return fmt.Errorf("failed to create folder: %v", err)
	}

	// Build URL
	url := fmt.Sprintf("https://1000kitap.com/kitap/normal-insanlar--182700/alintilar?sayfa=%d", pageNum)

	// Download HTML
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	// Generate filename
	filename := fmt.Sprintf("file%d.txt", pageNum)
	filePath := filepath.Join(folderPath, filename)

	// Save to file
	if err := ioutil.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Printf("[%s] Page %d downloaded: %s\n", time.Now().Format("15:04:05"), pageNum, filePath)
	return nil
}

func main() {
	folderPath := "quoteFiles"

	fmt.Printf("Starting Normal İnsanlar quotes downloader...\n")
	fmt.Printf("URL: https://1000kitap.com/kitap/normal-insanlar--182700/alintilar\n")
	fmt.Printf("Saving to: %s/\n", folderPath)
	fmt.Printf("Pages: 1-100\n\n")

	successCount := 0
	failCount := 0

	for pageNum := 1; pageNum <= 100; pageNum++ {
		if err := downloadAndSave(pageNum, folderPath); err != nil {
			log.Printf("Error on page %d: %v", pageNum, err)
			failCount++
		} else {
			successCount++
		}

		// Add a small delay to avoid overwhelming the server
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("\n✓ Download completed!\n")
	fmt.Printf("  Success: %d pages\n", successCount)
	fmt.Printf("  Failed: %d pages\n", failCount)
}
