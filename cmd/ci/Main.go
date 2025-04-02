package main

import (
	"context"
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

	// TODO: fix 2025/03/29 09:18:35 branch is not protected response
	// ic, err := gh.GetIntegrityConfig(ownerAndRepoSplit[0], ownerAndRepoSplit[1], *targetBranch, client)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// logger.Info("code integrity", "conf", ic)

	var lc *git.Repository
	var repoDir string
	if *mode == "clone" {
		// TODO: check auth to make this work on non public repos
		//	Auth: &http.BasicAuth{
		// Username: "abc123", // anything except an empty string
		// Password: "github_access_token",
		// },
		logger.Info("Cloning", "clone url", *r.CloneURL, "clone target", *cloneTarget)
		lc, err = git.PlainClone(*cloneTarget, true, &git.CloneOptions{URL: *r.CloneURL})
		defer os.RemoveAll(*cloneTarget)
		repoDir = *cloneTarget
	} else {
		lc, err = git.PlainOpen(*localPath)
		repoDir = *localPath
	}
	if err != nil {
		log.Fatal(err)
	}

	prs, err := gh.GetPullRequests(ownerAndRepoSplit[0], ownerAndRepoSplit[1], *targetBranch, *token)
	allPrCommits := vcs.GetMergedPrHashs(prs, lc, repoDir)

	allCommits, err := vcs.GetCommitData(lc, repoDir, *targetBranch)
	ach := allCommits.Hashs
	logger.Info("Number all commits", *targetBranch, len(ach))

	npr := 0
	for _, pn := range allPrCommits {
		for h := range pn {
			npr++
			delete(ach, h)
		}
	}

	logger.Info("Number commits from PRs", "number", npr)
	logger.Info("Number commits without PR", "number", len(ach))

	elapsed := time.Since(start)
	logger.Info("Execution finished", "time elapsed", elapsed)
}
