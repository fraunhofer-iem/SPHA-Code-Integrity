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
	ownerAndRepo = flag.String("ownerAndRepo", "", "GitHub repository link (e.g., https://github.com/owner/repo)")
	token        = flag.String("token", "", "GitHub access token")
	targetBranch = flag.String("branch", "", "Target branch to analyze. Defaults to the default branch of the repository")
	cloneTarget  = flag.String("cloneTarget", "", "Target to clone. Defaults to tmp")
	logLevel     = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
	out          = flag.String("out", "", "Directory to which the output is written. Defaults to the current working directory.")
)

func main() {

	start := time.Now()
	// TODO: add input validation
	flag.Parse()

	logger := logging.SetUpLogging(*logLevel)

	if *ownerAndRepo == "" {
		panic("ownerAndRepo is required")
	}

	ownerAndRepoSplit := strings.Split(*ownerAndRepo, "/")
	// TODO: do validation for format

	if *token == "" {
		panic("token is required")
	}

	if *cloneTarget == "" {
		*cloneTarget = path.Join(os.TempDir(), "codeintegrity", ownerAndRepoSplit[1])
	}

	if *out == "" {
		wd, err := os.Getwd()
		if err != nil {
			panic("Couldn't get workind directory")
		}
		*out = wd
	}

	config := processor.RepoConfig{
		Owner:     ownerAndRepoSplit[0],
		Repo:      ownerAndRepoSplit[1],
		ClonePath: *cloneTarget,
		Branch:    *targetBranch,
		Token:     *token,
		Out:       *out,
	}

	repo, err := processor.ProcessRepo(config)
	if err != nil {
		panic(err)
	}

	*out = path.Join(*out, "result.json")
	err = io.StoreResult(*out, *repo)
	if err != nil {
		panic(err)
	}

	elapsed := time.Since(start)
	logger.Info("Execution finished", "time elapsed", elapsed)
}
