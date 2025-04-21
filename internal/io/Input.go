package io

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
