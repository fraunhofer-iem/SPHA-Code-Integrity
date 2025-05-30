package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"project-integrity-calculator/internal/io"
	"strings"
)

var in = flag.String("in", "", "Path to the result to be transformed")
var out = flag.String("out", "", "Path to write the transformed result to")

func main() {
	flag.Parse()
	folderPath := *in
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		panic(err)
	}

	allCommits := make([]io.Commit, 0, len(entries)*50)
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

		for _, c := range repo.CommitsWithoutPR {
			c.Message = strings.ReplaceAll(strings.ToLower(c.Message), "-", " ")

		}

		allCommits = append(allCommits, repo.CommitsWithoutPR...)
	}

	file, err := os.Create(*out)
	if err != nil {
		panic(err)
	}

	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&allCommits)
	if err != nil {
		panic(err)
	}
}
