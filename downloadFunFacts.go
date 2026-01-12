package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func downloadAndSave(url string, folderPath string) error {
	// Create folder if it doesn't exist
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return fmt.Errorf("failed to create folder: %v", err)
	}

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

	// Generate random filename
	timestamp := time.Now().Unix()
	randomNum := rand.Intn(100000)
	filename := fmt.Sprintf("funfact_%d_%d.txt", timestamp, randomNum)
	filePath := filepath.Join(folderPath, filename)

	// Save to file
	if err := ioutil.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Printf("[%s] Downloaded and saved to: %s\n", time.Now().Format("15:04:05"), filePath)
	return nil
}

func main() {
	url := "https://uselessfacts.jsph.pl/random.html?language=en"
	folderPath := "funfacts"

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	fmt.Printf("Starting fun facts downloader...\n")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Saving to: %s/\n", folderPath)
	fmt.Printf("Interval: 5 seconds\n")
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Download immediately on start
	if err := downloadAndSave(url, folderPath); err != nil {
		log.Printf("Error: %v", err)
	}

	// Create a ticker that fires every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Download on every tick
	for range ticker.C {
		if err := downloadAndSave(url, folderPath); err != nil {
			log.Printf("Error: %v", err)
		}
	}
}
