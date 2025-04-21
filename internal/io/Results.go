package io

type Result struct {
	Repo []Repo
}

type Repo struct {
	Branch                  string
	Head                    string
	Url                     string
	Stats                   Stats
	CommitsWithoutPR        []Commit
	CommitsWithoutSignature []Commit
}

type Stats struct {
	NumberCommits      int
	NumberPRs          int
	NumberContributors int
	Languages          []string
	Stars              int
}

type Commit struct {
	GitOID  string
	Message string
	Date    string
}
