package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path"
	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/logging"
	"project-integrity-calculator/internal/vcs"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v70/github"
)

var (
	ownerAndRepo = flag.String("ownerAndRepo", "", "GitHub repository link (e.g., https://github.com/owner/repo)")
	token        = flag.String("token", "", "GitHub access token")
	targetBranch = flag.String("branch", "", "Target branch to analyze. Defaults to the default branch of the repository")
	mode         = flag.String("mode", "local", "Mode: 'local' or 'clone'")
	localPath    = flag.String("localPath", "", "Path to the local repository (required if mode is 'local')")
	cloneTarget  = flag.String("cloneTarget", "", "Target to clone. Defaults to tmp")
	logLevel     = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
)

func main() {

	start := time.Now()
	// TODO: add input validation
	flag.Parse()

	logger := logging.SetUpLogging(*logLevel)

	if *ownerAndRepo == "" {
		panic("ownerAndRepo is required")
	}

	if *token == "" {
		panic("token is required")
	}

	if *mode == "local" && *localPath == "" {
		panic("localPath is required if mode is 'local'")
	}

	if *mode == "clone" && *cloneTarget == "" {
		*cloneTarget = path.Join(os.TempDir(), "codeintegrity")
	}

	client := github.NewClient(nil).WithAuthToken(*token)

	ownerAndRepoSplit := strings.Split(*ownerAndRepo, "/")
	// possible protection rules https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets
	r, _, err := client.Repositories.Get(context.Background(), ownerAndRepoSplit[0], ownerAndRepoSplit[1])
	if err != nil {
		log.Fatal(err)
	}

	if *targetBranch == "" {
		*targetBranch = r.GetDefaultBranch()
	}

	var lc *git.Repository
	var repoDir string
	if *mode == "clone" {
		// TODO: check auth to make this work on non public repos
		//	Auth: &http.BasicAuth{
		// Username: "abc123", // anything except an empty string
		// Password: "github_access_token",
		// },
		logger.Info("Cloning", "clone url", *r.CloneURL, "branch", *targetBranch, "clone target", *cloneTarget)
		lc, err = git.PlainClone(*cloneTarget, true, &git.CloneOptions{URL: *r.CloneURL})
		defer os.RemoveAll(*cloneTarget)
		repoDir = *cloneTarget
	} else {
		lc, err = git.PlainOpen(*localPath)
		repoDir = *localPath
	}
	if err != nil {
		panic(err)
	}

	methodTimer := time.Now()
	prs, err := gh.GetPullRequests(ownerAndRepoSplit[0], ownerAndRepoSplit[1], *targetBranch, *token)
	if err != nil {
		panic(err)
	}
	elapsed := time.Since(methodTimer)
	logger.Info("Time to query all Pull requests", "time", elapsed)

	methodTimer = time.Now()
	allPrCommits, err := vcs.GetMergedPrHashs(prs, lc, repoDir)
	if err != nil {
		panic(err)
	}
	elapsed = time.Since(methodTimer)
	logger.Info("Time to get pr hashes", "time", elapsed)

	methodTimer = time.Now()
	allCommits, err := vcs.GetCommitData(lc, repoDir, *targetBranch)
	if err != nil {
		panic(err)
	}
	elapsed = time.Since(methodTimer)
	logger.Info("Time to get commit hashes from target branch", "time", elapsed)

	ach := allCommits.Hashs
	logger.Info("Number all commits", *targetBranch, ach.Size())

	commitsWithoutPr := ach.Difference(allPrCommits)

	logger.Info("Number commits from PRs", "number", allPrCommits.Size())
	logger.Info("Number commits without PR", "number", commitsWithoutPr.Size())

	file, err := os.Create("data.json")
	if err != nil {
		panic(err)
	}

	defer file.Close()
	encoder := json.NewEncoder(file)

	err = encoder.Encode(&ach)
	if err != nil {
		panic(err)
	}

	elapsed = time.Since(start)
	logger.Info("Execution finished", "time elapsed", elapsed)
}
