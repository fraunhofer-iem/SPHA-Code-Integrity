package main

import (
	"encoding/json"
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
	token       = flag.String("token", "", "GitHub access token")
	mode        = flag.String("mode", "local", "Mode: 'local' or 'clone'")
	cloneTarget = flag.String("cloneTarget", "", "Target to clone. Defaults to tmp")
	logLevel    = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
	out         = flag.String("out", "", "Directory to which the output is written. Defaults to the current working directory.")
	in          = flag.String("in", "", "Input file with the repositories to process.")
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

	file, err := os.Open(*in)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	var input io.Input
	if err := decoder.Decode(&input); err != nil {
		panic(err)
	}
	for _, r := range input.Data.Search.Nodes {

		ownerAndRepoSplit := strings.Split(r.NameWithOwner, "/")

		clonePath := path.Join(*cloneTarget, ownerAndRepoSplit[1])
		config := processor.RepoConfig{
			Owner:     ownerAndRepoSplit[0],
			Repo:      ownerAndRepoSplit[1],
			LocalPath: "",
			ClonePath: clonePath,
			Branch:    "",
			Token:     *token,
			Out:       *out,
		}

		repo, err := processor.ProcessRepo(config)
		if err != nil {
			panic(err)
		}

		*out = path.Join(*out, config.Owner+config.Repo+"-result.json")
		err = io.StoreResult(*out, *repo)
		if err != nil {
			panic(err)
		}
	}
	elapsed := time.Since(start)
	logger.Info("Execution finished", "time elapsed", elapsed)
}
