package vcs

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"project-integrity-calculator/internal/gh"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type CommitData struct {
	NumberCommits  int
	NumberVerified int
	Hashs          map[string]*object.Commit
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func GetCommitData(lc *git.Repository, targetBranch string) (*CommitData, error) {

	hash, err := lc.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, err
	}

	fmt.Printf("Hash %s\n", hash.String())
	c, _ := lc.CommitObject(*hash)
	fmt.Printf("Commit %+v\n", c)
	iter, _ := lc.Log(&git.LogOptions{From: c.Hash})
	hashs := make(map[string]*object.Commit)
	cc := 0
	csc := 0

	iter.ForEach(func(curr *object.Commit) error {
		if !curr.Hash.IsZero() {
			hashs[curr.Hash.String()] = curr
			cc++
			if c.PGPSignature != "" {
				csc++
			}
		}
		return nil
	})

	return &CommitData{
		NumberCommits:  cc,
		NumberVerified: csc,
		Hashs:          hashs,
	}, nil
}

func GetMergedPrHashs(prs []gh.PR, lc git.Repository) map[int]map[string]*object.Commit {

	refspecs := []config.RefSpec{}
	for _, pr := range prs {
		if pr.State == "MERGED" {
			prn := pr.Number
			// git fetch origin pull/<pr_number>/head:<local_branch_name>
			refspec := fmt.Sprintf("+refs/pull/%d/head:pull/%d", prn, prn)
			refspecs = append(refspecs, config.RefSpec(refspec))
		}
	}

	err := lc.Fetch(&git.FetchOptions{
		RefSpecs: refspecs},
	)
	if err != nil {
		log.Fatal(err)
	}

	allNewCommits := make(map[int]map[string]*object.Commit)

	for _, pr := range prs {

		// fmt.Printf("START PR Number: %d\n", pr.Number)

		newCommits := make(map[string]*object.Commit)

		iter, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.HeadRefOid)})
		iter.ForEach(func(curr *object.Commit) error {
			newCommits[curr.Hash.String()] = curr
			// fmt.Printf("Head Commit %+v\n", curr)
			return nil
		})

		iterBase, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.BaseRefOid)})
		iterBase.ForEach(func(curr *object.Commit) error {
			delete(newCommits, curr.Hash.String())
			// fmt.Printf("Base Commit %+v\n", curr)
			return nil
		})

		allNewCommits[pr.Number] = newCommits
	}

	return allNewCommits
}

func GetPatchId(dir string, hash string) (string, error) {
	showCmd := exec.Command("git", "show", hash)
	showCmd.Dir = dir
	out, err := showCmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	patchIdCmd := exec.Command("git", "patch-id", "--stable")
	patchIdCmd.Dir = dir
	patchIdCmd.Stdin = out

	patchOut, err := patchIdCmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	patchIdCmd.Start()
	showCmd.Start()

	scanner := bufio.NewScanner(patchOut)

	output := ""
	for scanner.Scan() {
		output += scanner.Text()
	}

	showCmd.Wait()
	patchIdCmd.Wait()

	patchId := strings.Split(output, " ")[0]

	return patchId, nil
}
