package io

import (
	"encoding/json"
	"os"
	"path"
)

// StoreResult Create outDir if not exists
// create outdir + fileName if not exists
// stores repo in a created file
func StoreResult(outDir, fileName string, repo Repo) error {
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		return err
	}
	outFile := path.Join(outDir, fileName)
	file, err := os.Create(outFile)
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return it to avoid masking the original error
			_ = err // explicitly ignore the error
		}
	}()
	encoder := json.NewEncoder(file)

	return encoder.Encode(&repo)
}
