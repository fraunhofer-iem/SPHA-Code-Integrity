package main

import (
	"flag"
	"os"
	"path"
	"project-integrity-calculator/internal/io"
	"project-integrity-calculator/internal/logging"
	"project-integrity-calculator/internal/processor"
	"strings"
	"time"
)

var (
	token              = flag.String("token", "", "GitHub access token")
	cloneTarget        = flag.String("cloneTarget", "", "Target to clone. Defaults to tmp")
	logLevel           = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
	out                = flag.String("out", "", "Directory to which the output is written. Defaults to the current working directory.")
	in                 = flag.String("in", "", "Input file with the repositories to process.")
	ignoreFirstCommits = flag.Bool("ignore", false, "If set to true all commits until the first PR has been merged are ignored. Defaults to false.")
)

func main() {

	start := time.Now()
	// TODO: add input validation
	flag.Parse()

	logger := logging.SetUpLogging(*logLevel)

	if *in == "" {
		panic("in is required")
	}

	if *token == "" {
		panic("token is required")
	}

	if *cloneTarget == "" {
		*cloneTarget = path.Join(os.TempDir(), "codeintegrity")
	}

	if *out == "" {
		wd, err := os.Getwd()
		if err != nil {
			panic("Couldn't get workind directory")
		}
		*out = wd
	}

	input, err := io.GetInput(*in)
	if err != nil {
		panic(err)
	}

	failedRepos := 0
	for _, r := range input.Data.Search.Nodes {

		ownerAndRepoSplit := strings.Split(r.NameWithOwner, "/")

		clonePath := path.Join(*cloneTarget, ownerAndRepoSplit[1])
		config := processor.RepoConfig{
			Owner:              ownerAndRepoSplit[0],
			Repo:               ownerAndRepoSplit[1],
			ClonePath:          clonePath,
			Branch:             "",
			Token:              *token,
			Out:                *out,
			IgnoreFirstCommits: *ignoreFirstCommits,
		}

		repo, err := processor.ProcessRepo(config)
		if err != nil {
			failedRepos++
			logger.Warn("Process repo failed", "err", err)
			continue
		}

		fileName := config.Owner + config.Repo + "-result.json"
		err = io.StoreResult(*out, fileName, *repo)
		if err != nil {
			failedRepos++
			logger.Warn("Store result failed", "err", err)
			continue
		}
	}
	elapsed := time.Since(start)
	logger.Info("Execution finished", "time elapsed", elapsed, "number of failed repos", failedRepos)
}
