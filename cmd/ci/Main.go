package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/vcs"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v70/github"
)

var (
	ownerAndRepo = flag.String("ownerAndRepo", "", "GitHub repository link (e.g., https://github.com/owner/repo)")
	token        = flag.String("token", "", "GitHub access token")
	targetBranch = flag.String("branch", "", "Target branch to analyze. Defaults to the default branch of the repository")
	mode         = flag.String("mode", "local", "Mode: 'local' or 'clone'")
	localPath    = flag.String("localPath", "", "Path to the local repository (required if mode is 'local')")
	cloneTarget  = flag.String("cloneTarget", "", "Target to clone. Defaults to tmp")
)

func main() {
	// TODO: add input validation
	flag.Parse()

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
	if *mode == "clone" {
		// TODO: check auth to make this work on non public repos
		//	Auth: &http.BasicAuth{
		// Username: "abc123", // anything except an empty string
		// Password: "github_access_token",
		// },
		fmt.Printf("Cloning %s to %s\n", *r.CloneURL, *cloneTarget)
		lc, err = git.PlainClone(*cloneTarget, true, &git.CloneOptions{URL: *r.CloneURL})
		defer os.RemoveAll(*cloneTarget)
	} else {
		lc, err = git.PlainOpen(*localPath)
	}
	if err != nil {
		log.Fatal(err)
	}

	prs, err := gh.GetPullRequests(ownerAndRepoSplit[0], ownerAndRepoSplit[1], *targetBranch, *token)
	fmt.Printf("PRS %+v\n", prs)
	allNewCommits := vcs.GetMergedPrHashs(prs, *lc)

	allCommits, err := vcs.GetCommitData(lc, *targetBranch)
	ach := allCommits.Hashs
	fmt.Printf("Number all commits %d\n", len(ach))

	fmt.Println("all commits hashes")
	for h, c := range ach {
		fmt.Printf("hash %s, commit %+v", h, c)
	}

	fmt.Println("PR Commits list")
	npr := 0
	for _, pn := range allNewCommits {
		for h, c := range pn {
			fmt.Printf("Removing commit %+v \n", c)
			npr++
			delete(ach, h)
		}
	}

	fmt.Printf("Number PR commits %d\n", npr)
	fmt.Printf("Number all commits after delete %d\n", len(ach))

	fmt.Println("-----------------Commits without PR-------")

	for h := range allCommits.Hashs {
		c, err := lc.CommitObject(plumbing.NewHash(h))
		if err != nil {
			continue
		}
		fmt.Printf("Commits without PR: %+v\n", c)
	}

	// sc, err := vcs.GetCommitData(lc, *targetBranch)
	// fmt.Printf("Number of commits: %d\n", sc.NumberCommits)
	// fmt.Printf("Number of verified commits: %d\n", sc.NumberVerified)

	// // TODO: fix 2025/03/29 09:18:35 branch is not protected response
	// // ic, err := gh.GetIntegrityConfig(ownerAndRepoSplit[0], ownerAndRepoSplit[1], *targetBranch, client)
	// // if err != nil {
	// // 	log.Fatal(err)
	// // }

	// // prStats, err := gh.GetPullRequestStats(ownerAndRepoSplit[0], ownerAndRepoSplit[1], *targetBranch, *token, 1) //ic.ApprovingCount)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Printf("Number of sufficient reviews: %d\n", prStats.NumberSufficientReviews)
	// fmt.Printf("PR Numbers: %v\n", prStats.PRNumbers)

	// refSpecs := []config.RefSpec{}

	// for _, prn := range prStats.PRNumbers {
	// 	refspec := fmt.Sprintf("+refs/pull/%d/head:pull/%d", prn, prn)
	// 	log.Printf("Refspec %s", refspec)
	// 	refSpecs = append(refSpecs, config.RefSpec(refspec))
	// }

	// lc.Fetch(&git.FetchOptions{
	// 	RefSpecs: refSpecs},
	// )
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Printf("Overall commit count: %d\n", sc.NumberCommits)

	// for _, prn := range prStats.PRNumbers {

	// 	prBranch := fmt.Sprintf("pull/%d", prn)

	// 	sc2, err := vcs.GetCommitData(lc, prBranch)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Printf("Number of commits: %d\n", sc2.NumberCommits)
	// 	fmt.Printf("Number of verified commits: %d\n", sc2.NumberVerified)

	// 	// TODO: this doesn't work right now investigate map
	// 	// TODO: this is not the right way to do this. we currently take all commits from the PR
	// 	// this is a flaw, as later branches contain commits which have been directly introduced
	// 	// without a pr. therefore, we accidently validate them.
	// 	// we need to compare base branch with pr branch at the moment of the pr
	// 	fmt.Printf("sc2 hashs %+v\n", sc2.Hashs)
	// 	for k := range sc2.Hashs {
	// 		fmt.Printf("Deleting %s\n", k)
	// 		delete(sc.Hashs, k)
	// 	}
	// }

	// fmt.Printf("commit count without PR: %d\n", len(sc.Hashs))
	// fmt.Printf("remaining hashs %+v\n", sc.Hashs)
}
