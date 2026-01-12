package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// TriviaQuestion represents a trivia question with category, question, and answer
type TriviaQuestion struct {
	Category string
	Question string
	Answer   string
}

func readTriviaFromFile(filename string) ([]TriviaQuestion, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var trivia []TriviaQuestion
	seen := make(map[string]bool)

	// Split by newlines
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by comma, but we need to handle commas inside curly brackets
		var parts []string
		var current strings.Builder
		insideBrackets := 0

		for _, char := range line {
			if char == '{' {
				insideBrackets++
			} else if char == '}' {
				insideBrackets--
			} else if char == ',' && insideBrackets == 0 {
				parts = append(parts, current.String())
				current.Reset()
				continue
			}
			current.WriteRune(char)
		}
		// Add the last part
		if current.Len() > 0 {
			parts = append(parts, current.String())
		}

		// Expect 3 parts: category, question, answer
		if len(parts) < 3 {
			log.Printf("Skipping line %d: not enough columns (%d)", lineNum+1, len(parts))
			continue
		}

		// Remove curly brackets and trim spaces
		category := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(parts[0], "{", ""), "}", ""))
		question := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(parts[1], "{", ""), "}", ""))
		answer := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(parts[2], "{", ""), "}", ""))

		// Skip empty entries
		if category == "" || question == "" || answer == "" {
			continue
		}

		// Remove duplicates based on question text
		key := strings.ToLower(question)
		if !seen[key] {
			seen[key] = true
			trivia = append(trivia, TriviaQuestion{
				Category: category,
				Question: question,
				Answer:   answer,
			})
		}
	}

	return trivia, nil
}

func insertTriviaIntoDatabase(trivia []TriviaQuestion, dbPath string) error {
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
	_, err = db.Exec("DROP TABLE IF EXISTS trivia")
	if err != nil {
		return fmt.Errorf("failed to drop table: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE trivia (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            category TEXT NOT NULL,
            question TEXT NOT NULL,
            answer TEXT NOT NULL,
            viewCount INTEGER NOT NULL DEFAULT 0
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

	stmt, err := tx.Prepare("INSERT INTO trivia (category, question, answer, viewCount) VALUES (?, ?, ?, 0)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, q := range trivia {
		_, err = stmt.Exec(q.Category, q.Question, q.Answer)
		if err != nil {
			log.Printf("Warning: failed to insert trivia: %v", err)
			continue
		}
		inserted++
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("✓ Inserted %d trivia questions into database.db\n", inserted)
	return nil
}

func main() {
	triviaFile := "trivia.txt"
	dbPath := "database.db"

	// Check if trivia file exists
	if _, err := os.Stat(triviaFile); os.IsNotExist(err) {
		log.Fatalf("File %s does not exist", triviaFile)
	}

	fmt.Printf("Reading trivia from %s...\n", triviaFile)

	// Read trivia from file with custom parsing
	trivia, err := readTriviaFromFile(triviaFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d unique trivia questions (duplicates removed)\n", len(trivia))

	// Insert into database
	if err := insertTriviaIntoDatabase(trivia, dbPath); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Database operations completed successfully")

	// Show first 5 trivia questions as preview
	fmt.Println("\nPreview (first 5 trivia questions):")
	for i, q := range trivia {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. [%s] Q: %s\n   A: %s\n", i+1, q.Category, q.Question, q.Answer)
	}
}
