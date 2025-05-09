package io

import (
	"encoding/json"
	"os"
)

type Input struct {
	Data struct {
		Search struct {
			Nodes []struct {
				NameWithOwner string `json:"nameWithOwner"`
				Stars         int    `json:"stargazerCount"`
				Url           string `json:"url"`
			} `json:"nodes"`
		} `json:"search"`
	} `json:"data"`
}

func GetInput(in string) (*Input, error) {
	file, err := os.Open(in)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	var input Input
	if err := decoder.Decode(&input); err != nil {
		return nil, err
	}

	return &input, nil
}
