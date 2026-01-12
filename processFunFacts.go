package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// FunFact represents the structure of the JSON data
type FunFact struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func parseFunFactsFromFolder(folderPath string) ([]FunFact, error) {
	var allFacts []FunFact
	seenTexts := make(map[string]bool)

	// Find all .txt files in the folder
	files, err := filepath.Glob(filepath.Join(folderPath, "*.txt"))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .txt files found in %s", folderPath)
	}

	fmt.Printf("Processing %d files...\n", len(files))

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Error reading %s: %v", file, err)
			continue
		}

		var fact FunFact
		if err := json.Unmarshal(content, &fact); err != nil {
			log.Printf("Error parsing JSON in %s: %v", file, err)
			continue
		}

		// Normalize text for comparison (trim spaces, lowercase)
		normalizedText := strings.TrimSpace(strings.ToLower(fact.Text))

		// Skip duplicates based on text content
		if normalizedText != "" && !seenTexts[normalizedText] {
			seenTexts[normalizedText] = true
			allFacts = append(allFacts, fact)
		}
	}

	return allFacts, nil
}

func insertIntoDatabase(facts []FunFact, dbPath string) error {
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

	// Drop and recreate table
	_, err = db.Exec("DROP TABLE IF EXISTS funFacts")
	if err != nil {
		return fmt.Errorf("failed to drop table: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE funFacts (
            id TEXT PRIMARY KEY,
            text TEXT NOT NULL
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

	stmt, err := tx.Prepare("INSERT INTO funFacts (id, text) VALUES (?, ?)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, fact := range facts {
		if fact.ID != "" && fact.Text != "" {
			_, err = stmt.Exec(fact.ID, fact.Text)
			if err != nil {
				log.Printf("Warning: failed to insert fact %s: %v", fact.ID, err)
				continue
			}
			inserted++
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("✓ Inserted %d fun facts into database.db\n", inserted)
	return nil
}

func main() {
	folderPath := "funfacts"
	dbPath := "database.db"

	// Check if folder exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		log.Fatalf("Folder %s does not exist", folderPath)
	}

	// Parse all fun facts (with duplicate removal based on text)
	facts, err := parseFunFactsFromFolder(folderPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d unique fun facts (duplicates removed)\n", len(facts))

	// Insert into database
	if err := insertIntoDatabase(facts, dbPath); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Database operations completed successfully")

	// Show first 5 facts as preview
	fmt.Println("\nPreview (first 5 facts):")
	for i, fact := range facts {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. [%s] %s\n", i+1, fact.ID[:8], fact.Text)
	}
}
