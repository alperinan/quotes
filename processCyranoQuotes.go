package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// CyranoQuote represents a quote from Cyrano de Bergerac
type CyranoQuote struct {
	Text string `json:"text"`
}

func parseQuotesFromHTML(htmlContent string) ([]CyranoQuote, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	var quotes []CyranoQuote
	seenTexts := make(map[string]bool)

	// List of common headings/menu items to filter out
	filterWords := map[string]bool{
		"genel bakış":     true,
		"incelemeler":     true,
		"alıntılar":       true,
		"benzer kitaplar": true,
		"devamını oku":    true,
		"tümünü göster":   true,
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		// Look for quote spans with class "text text text-15"
		if n.Type == html.ElementNode && n.Data == "span" {
			class := getAttr(n, "class")
			if strings.Contains(class, "text-15") {
				quoteText := getTextContent(n)
				quoteText = strings.TrimSpace(quoteText)

				// Clean up the text
				quoteText = cleanText(quoteText)

				// Normalize for duplicate detection and filtering
				normalized := strings.ToLower(strings.TrimSpace(quoteText))

				// Filter out headings and short text
				if normalized != "" &&
					len(quoteText) > 20 &&
					!filterWords[normalized] &&
					!seenTexts[normalized] {
					seenTexts[normalized] = true
					quotes = append(quotes, CyranoQuote{
						Text: quoteText,
					})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return quotes, nil
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	var text strings.Builder
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)
	return text.String()
}

func cleanText(s string) string {
	// Remove extra whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	// Trim spaces
	s = strings.TrimSpace(s)
	return s
}

func processAllFiles(folderPath string) ([]CyranoQuote, error) {
	var allQuotes []CyranoQuote
	globalSeen := make(map[string]bool)

	fmt.Printf("Processing files from %s...\n\n", folderPath)

	for i := 1; i <= 100; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		filePath := filepath.Join(folderPath, filename)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading %s: %v", filename, err)
			continue
		}

		quotes, err := parseQuotesFromHTML(string(content))
		if err != nil {
			log.Printf("Error parsing %s: %v", filename, err)
			continue
		}

		fmt.Printf("File: %s - Found %d quotes\n", filename, len(quotes))

		for _, quote := range quotes {
			normalized := strings.ToLower(strings.TrimSpace(quote.Text))
			if !globalSeen[normalized] {
				globalSeen[normalized] = true
				allQuotes = append(allQuotes, quote)
			}
		}
	}

	return allQuotes, nil
}

func main() {
	folderPath := "quoteFiles"
	outputPath := filepath.Join(folderPath, "output.json")

	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		log.Fatalf("Folder %s does not exist", folderPath)
	}

	quotes, err := processAllFiles(folderPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nFound %d unique quotes\n", len(quotes))

	// Write to JSON file with UTF-8 encoding in cyrano folder
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Error creating JSON file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(quotes); err != nil {
		log.Fatalf("Error writing JSON: %v", err)
	}

	fmt.Printf("✓ Successfully wrote %d quotes to %s\n", len(quotes), outputPath)

	// Show preview
	fmt.Println("\nPreview (first 3 quotes):")
	for i, quote := range quotes {
		if i >= 3 {
			break
		}
		fmt.Printf("%d. %s\n", i+1, quote.Text)
	}
}
