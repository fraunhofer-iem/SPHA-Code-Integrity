package io

type Result struct {
	Repo []Repo
}

type Repo struct {
	Branch           string
	Head             string
	Url              string
    NumberForcePushes int
	Stats            Stats
	CommitsWithoutPR []Commit
	UnsignedCommits  []Commit
}

type Stats struct {
	NumberCommits     int
	NumberPRs         int
	Languages         []string
	Stars             int
}

type Commit struct {
	GitOID  string
	Message string
	Date    string
	// show "G" for a good (valid) signature, "B" for a bad signature,
	// "U" for a good signature with unknown validity,
	// "X" for a good signature that has expired,
	// "Y" for a good signature made by an expired key,
	// "R" for a good signature made by a revoked key,
	// "E" if the signature cannot be checked (e.g. missing key)
	// and "N" for no signature
	Signed string
}
