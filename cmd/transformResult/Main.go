package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"project-integrity-calculator/internal/io"
	"strings"
	"time"
)

var in = flag.String("in", "", "Path to the result to be transformed")
var out = flag.String("out", "", "Path to write the transformed result to")

func main() {
	flag.Parse()

	err := writeCommitsToCSVFile(*in, *out)
	if err != nil {
		panic(err)
	}
}

func writeCommitsToCSVFile(folderPath, outputFilePath string) error {
	// 1. Create the output CSV file
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFilePath, err)
	}
	defer outputFile.Close() // Ensure the file is closed

	csvWriter := csv.NewWriter(outputFile)
	// No defer Flush() here, we will call Flush explicitly at the end

	// 2. Write the header row to the single output file
	header := []string{"Weekday", "Time"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Define the date layout string matching the provided examples
	const layout = "2006-01-02 15:04:05 -0700"

	// 3. Read the contents of the input folder
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("failed to read folder %s: %w", folderPath, err)
	}

	// 4. Process each entry in the folder
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()

		// Skip files that don't end with .json (case-insensitive check)
		if !strings.HasSuffix(strings.ToLower(fileName), ".json") {
			continue
		}

		// Get the full path to the JSON file
		filePath := filepath.Join(folderPath, fileName)

		// 5. Parse the JSON file into a Repo struct
		repo, err := io.GetResult(filePath)
		if err != nil {
			// Log the error and skip this file, but continue processing others
			log.Printf("Skipping file %s due to parsing error: %v", filePath, err)
			continue
		}

		// 6. Process commits in the parsed Repo and write to CSV
		for _, commit := range repo.CommitsWithoutPR {
			// Parse the commit date string
			t, err := time.Parse(layout, commit.Date)
			if err != nil {
				// Log the error and skip this commit
				log.Printf("Skipping commit %s (Date: %s) in file %s due to date parsing error: %v", commit.GitOID, commit.Date, filePath, err)
				continue
			}

			// Extract the weekday name
			weekdayStr := t.Weekday().String()

			// Format the time part as HH:MM:SS (24-hour format)
			timeStr := t.Format("15:04:05")

			// Prepare the row data
			row := []string{weekdayStr, timeStr}

			// Write the row to the single CSV file writer
			err = csvWriter.Write(row)
			if err != nil {
				// If writing a row fails, it's a critical error for the output file
				// Return the error immediately.
				return fmt.Errorf("failed to write CSV row for commit %s in file %s: %w", commit.GitOID, filePath, err)
			}
		}
	}

	// 7. Ensure all buffered data is written to the file
	csvWriter.Flush()

	// 8. Check for any errors during flushing
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("failed during CSV writer flush: %w", err)
	}

	return nil // Success
}
