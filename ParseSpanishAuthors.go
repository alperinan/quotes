package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

// Author represents an author with their quote count
type Author struct {
	Name       string `json:"name"`
	QuoteCount int    `json:"quoteCount"`
	Link       string `json:"link"`
}

func parseAuthorsFromHTML(htmlContent string) ([]Author, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	var authors []Author
	seenInFile := make(map[string]bool)

	// Pattern to extract quote count from text like "(123)"
	countRe := regexp.MustCompile(`\((\d+)\)`)

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		// Look for <div> tags containing author info
		if n.Type == html.ElementNode && n.Data == "div" {
			// Check if this div contains an <a> tag with author link
			var authorLink *html.Node
			var authorName string
			var authorHref string
			var quoteCount int

			// Search for <a> tag inside this div
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "a" {
					href := getAttr(c, "href")
					// Check if it's an author link (not /autor/ but direct author link)
					if href != "" && !strings.Contains(href, "telf") {
						authorLink = c
						authorName = getTextContent(c)
						authorName = strings.TrimSpace(authorName)
						authorHref = href
						break
					}
				}
			}

			// If we found an author link, look for the quote count
			if authorLink != nil && authorName != "" {
				// Get all text content from the div
				divText := getTextContent(n)

				// Extract quote count from pattern like "(123)"
				matches := countRe.FindStringSubmatch(divText)
				if len(matches) >= 2 {
					quoteCount, _ = strconv.Atoi(matches[1])
				}

				// Build full URL
				fullLink := authorHref
				if !strings.HasPrefix(authorHref, "http") {
					if strings.HasPrefix(authorHref, "/") {
						fullLink = "https://fraseslibros.com" + authorHref
					} else {
						fullLink = "https://fraseslibros.com/" + authorHref
					}
				}

				// Filter valid names (at least 3 chars, contains letters)
				if len(authorName) >= 3 && regexp.MustCompile(`[a-zA-ZÀ-ÿ]`).MatchString(authorName) {
					key := authorName
					if !seenInFile[key] {
						seenInFile[key] = true
						authors = append(authors, Author{
							Name:       authorName,
							QuoteCount: quoteCount,
							Link:       fullLink,
						})
					}
				}
			}
		}

		// Continue traversing
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return authors, nil
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

func parseAuthorsFromFile(filename string) ([]Author, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	return parseAuthorsFromHTML(string(content))
}

func parseAllAuthorsFromFolder(folderPath string) ([]Author, error) {
	var allAuthors []Author
	globalSeen := make(map[string]bool)

	files, err := filepath.Glob(filepath.Join(folderPath, "*.text"))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .text files found in %s", folderPath)
	}

	fmt.Printf("Processing %d files...\n\n", len(files))

	for _, file := range files {
		authors, err := parseAuthorsFromFile(file)
		if err != nil {
			log.Printf("Error parsing %s: %v", file, err)
			continue
		}

		fmt.Printf("File: %s - Found %d authors\n", filepath.Base(file), len(authors))

		for _, author := range authors {
			key := author.Name
			// Track globally to avoid duplicates across all files
			if !globalSeen[key] {
				globalSeen[key] = true
				allAuthors = append(allAuthors, author)
			}
		}
	}

	return allAuthors, nil
}

func insertAuthorsToDatabase(authors []Author, dbPath string) error {
	// Open database with UTF-8 encoding parameters
	db, err := sql.Open("sqlite3", dbPath+"?charset=utf8&parseTime=true")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Set UTF-8 encoding pragmas
	_, err = db.Exec("PRAGMA encoding = 'UTF-8'")
	if err != nil {
		return fmt.Errorf("failed to set encoding: %v", err)
	}

	// Drop and recreate table to ensure proper encoding
	_, err = db.Exec("DROP TABLE IF EXISTS frasesauthors")
	if err != nil {
		return fmt.Errorf("failed to drop table: %v", err)
	}

	// Create table with explicit TEXT type
	_, err = db.Exec(`
		CREATE TABLE frasesauthors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			authorName TEXT NOT NULL,
			authorLink TEXT NOT NULL,
			quoteCount INTEGER NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	stmt, err := tx.Prepare("INSERT INTO frasesauthors (authorName, authorLink, quoteCount) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, author := range authors {
		if author.Name != "" && author.Link != "" && strings.Contains(author.Link, "fraseslibros.com") {
			_, err = stmt.Exec(author.Name, author.Link, author.QuoteCount)
			if err != nil {
				log.Printf("Warning: failed to insert %s: %v", author.Name, err)
				continue
			}
			inserted++
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("\n✓ Inserted %d authors into database.db with UTF-8 encoding\n", inserted)
	return nil
}

func main() {
	folderPath := "fraseslibros"
	dbPath := "database.db"

	// Check if folder exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		log.Fatalf("Folder %s does not exist", folderPath)
	}

	authors, err := parseAllAuthorsFromFolder(folderPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nFound %d unique authors:\n\n", len(authors))
	for _, author := range authors {
		fmt.Printf("%-50s (%3d quotes) - %s\n", author.Name, author.QuoteCount, author.Link)
	}

	// Save to JSON file with proper UTF-8 encoding
	jsonFilePath := "fraseslibros.json"
	file, err := os.Create(jsonFilePath)
	if err != nil {
		log.Fatalf("Error creating JSON file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // This prevents escaping of UTF-8 characters

	if err := encoder.Encode(authors); err != nil {
		log.Fatalf("Error writing JSON: %v", err)
	}

	fmt.Printf("\nSaved %d authors to %s with proper UTF-8 encoding\n", len(authors), jsonFilePath)

	// Insert into database
	if err := insertAuthorsToDatabase(authors, dbPath); err != nil {
		log.Fatalf("Database error: %v", err)
	}

	fmt.Println("✓ Database operations completed successfully")
}


/etc/default/paperpi.ini

─2872 bash /usr/local/bin/paperpi -d
             └─2874 python3 /usr/local/paperpi/paperpi.py -d


			 # Activate the virtual environment
source /usr/local/paperpi/venv_paperpi/bin/activate

# Uninstall existing Pillow
sudo pip uninstall pillow -y

# Reinstall Pillow (it will recompile with FreeType support)
sudo pip install --no-cache-dir pillow

# Deactivate
deactivate