![SPHA Logo](docs/img/Software_Project_Health_Assistant_Secondary-Logo.png)

## About

SPHA is a fully automated tool suite that assesses and communicates all aspects
of software product quality. It does so by combining data about your projects
from sources like ticketing systems, and static analysis tools. For more details
see [software-product.health](https://www.software-product.health).

This project contains the calculations of a Code Integrity Score. Code integrity refers to the assurance that the code has not been tampered with or altered by an unauthorized individual. Evaluating code integrity can be challenging, particularly for larger projects that receive hundreds of commits daily from numerous contributors.

## Usage

You need to have git installed in order to run this program as some of the calculations (`git patch-id`) is not implemented in go-git) rely on calling the git CLI.
This project contains a CLI application written in GO. To run it and get all available `flags` execute:

```
go run Main.go --help
```
For the Code Integrity Score calculation multiple requests to GitHub's REST API are necessary. Consider using our cache function to avoid rate limiting [work in progress, will be added soon].

## Metrics and Data Model
The following metrics can be calculated and exported by the CLI tool:
```
type Repo struct {
	Branch           string
	Head             string
	Url              string
	Stats            Stats
	CommitsWithoutPR []Commit
	UnsignedCommits  []Commit
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
	// show "G" for a good (valid) signature, "B" for a bad signature,
	// "U" for a good signature with unknown validity,
	// "X" for a good signature that has expired,
	// "Y" for a good signature made by an expired key,
	// "R" for a good signature made by a revoked key,
	// "E" if the signature cannot be checked (e.g. missing key)
	// and "N" for no signature
	Signed string
}
```

## Contribute

You are welcome to contribute to SPHA and all its related projects. Please make sure you adhere to our
[contributing](CONTRIBUTING.md) guidelines.
First time contributors are asked to accept our
[contributor license agreement (CLA)](CLA.md).
For questions about the CLA please contact us at _SPHA(at)iem.fraunhofer.de_ or create an issue.

## License

Copyright (C) Fraunhofer IEM.
Software Product Health Assistant (SPHA) and all its components are published under the MIT license.

<picture>
<source media="(prefers-color-scheme: dark)" srcset="./docs/img/IEM_Logo_White.png">
<img alt="Logo IEM" src="./docs/img/IEM_Logo_Dark.png">
</picture>
