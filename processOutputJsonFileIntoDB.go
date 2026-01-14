package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// CyranoQuote represents a quote from the JSON file
type CyranoQuote struct {
	Text string `json:"text"`
}

func readQuotesFromJSON(filename string) ([]CyranoQuote, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %v", err)
	}

	var quotes []CyranoQuote
	if err := json.Unmarshal(content, &quotes); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return quotes, nil
}

func insertQuotesIntoDatabase(quotes []CyranoQuote, dbPath string) error {
	// Open database with UTF-8 encoding
	db, err := sql.Open("sqlite3", dbPath+"?charset=utf8&parseTime=true")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Set UTF-8 encoding
	_, err = db.Exec("PRAGMA encoding = 'UTF-8'")
	if err != nil {
		return fmt.Errorf("failed to set encoding: %v", err)
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	stmt, err := tx.Prepare("INSERT INTO quotes (text, author, lang, viewCount) VALUES (?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	author := "Sally Rooney - Normal İnsanlar"
	lang := "tr"
	viewCount := 0

	inserted := 0
	for _, quote := range quotes {
		if quote.Text != "" {
			_, err = stmt.Exec(quote.Text, author, lang, viewCount)
			if err != nil {
				log.Printf("Warning: failed to insert quote: %v", err)
				continue
			}
			inserted++
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("✓ Inserted %d quotes into database.db\n", inserted)
	return nil
}

func main() {
	jsonFile := "quoteFiles/output.json"
	dbPath := "database.db"

	// Check if JSON file exists
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		log.Fatalf("File %s does not exist", jsonFile)
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("Database %s does not exist", dbPath)
	}

	fmt.Printf("Reading quotes from %s...\n", jsonFile)

	// Read quotes from JSON
	quotes, err := readQuotesFromJSON(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d quotes in JSON file\n", len(quotes))

	// Insert into database
	if err := insertQuotesIntoDatabase(quotes, dbPath); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Database operations completed successfully")

	// Show preview
	fmt.Println("\nPreview (first 3 quotes inserted):")
	for i, quote := range quotes {
		if i >= 3 {
			break
		}
		fmt.Printf("%d. %s\n", i+1, quote.Text)
	}
}
