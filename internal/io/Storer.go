package io

import (
	"encoding/json"
	"os"
)

func StoreResult(out string, repo Repo) error {
	file, err := os.Create(out)
	if err != nil {
		return err
	}

	defer file.Close()
	encoder := json.NewEncoder(file)

	return encoder.Encode(&repo)
}
