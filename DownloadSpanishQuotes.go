package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func downloadAndSave(url string) error {
	// Extract filename from URL
	// Example: https://fraseslibros.com/autores/a/1 -> a1.text
	re := regexp.MustCompile(`/autores/([a-z]+)/(\d+)`)
	matches := re.FindStringSubmatch(url)

	var filename string
	if len(matches) >= 3 {
		filename = fmt.Sprintf("%s%s.text", matches[1], matches[2])
	} else {
		// Fallback: use last part of URL
		parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
		if len(parts) > 0 {
			filename = parts[len(parts)-1] + ".text"
		} else {
			filename = "download.text"
		}
	}

	// Create folder if it doesn't exist
	folderPath := "fraseslibros"
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return fmt.Errorf("failed to create folder: %v", err)
	}

	// Download HTML
	client := &http.Client{}
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

	// Save to file
	filePath := filepath.Join(folderPath, filename)
	if err := ioutil.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Printf("Downloaded and saved to: %s\n", filePath)
	return nil
}

func main() {
	url := "https://fraseslibros.com/autores/z/1"
	if err := downloadAndSave(url); err != nil {
		log.Fatal(err)
	}
}
