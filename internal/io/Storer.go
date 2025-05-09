package io

import (
	"encoding/json"
	"os"
	"path"
)

// Create outDir if not exists
// create outdir + fileName if not exists
// stores repo in created file
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

	defer file.Close()
	encoder := json.NewEncoder(file)

	return encoder.Encode(&repo)
}
