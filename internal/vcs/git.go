package vcs

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"unicode"

	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"

	"github.com/hashicorp/go-set/v3"
)

func GetCommitShaForMergedPr(prs []gh.PR, repoDir string) (*map[int]*set.Set[string], error) {
	logger := slog.Default()

	if len(prs) == 0 {
		logger.Warn("No PRs provided to GetCommitShaForMergedPr", "repoDir", repoDir)
		return &map[int]*set.Set[string]{}, nil
	}

	err := fetchAllRefs(prs, repoDir)
	if err != nil {
		return nil, err
	}

	res := make(map[int]*set.Set[string], len(prs))

	for _, pr := range prs {
		newCommits, err := getCommitHashsForPr(repoDir, pr)
		if err != nil {
			continue
		}
		logger.Debug("Commits from pr", "len", newCommits.Size())
		res[pr.Number] = newCommits
	}

	return &res, nil
}

func fetchAllRefs(prs []gh.PR, dir string) error {
	if len(prs) == 0 {
		return nil
	}

	args := []string{"fetch", "origin"}
	for _, pr := range prs {
		if pr.State == "MERGED" {
			prn := pr.Number
			args = append(args, fmt.Sprintf("pull/%d/head:pull/%d", prn, prn))
		}
	}

	// git fetch origin pull/<pr_number>/head:<local_branch_name> ...
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	o, err := cmd.CombinedOutput()
	if err != nil {
		slog.Default().Error("Git fetch failed", "target dir", dir, "output", o)
		return err
	}

	return nil
}

func getCommitHashsForPr(dir string, pr gh.PR) (*set.Set[string], error) {

	slog.Default().Debug("processing pr", "pr number", pr.Number, "base ref", pr.BaseRefOid, "head ref", pr.HeadRefOid)
	newCommits, err := getRevList(dir, fmt.Sprintf("%s..%s", pr.BaseRefOid, pr.HeadRefOid))
	if err != nil {
		return nil, err
	}
	commitSet := set.From(newCommits)
	commitSet.Insert(pr.MergeCommit.Oid)

	return commitSet, nil
}

// getRevList executes the git rev-list command and returns commit hashes.
func getRevList(repoPath, arg string) ([]string, error) {
	cmd := exec.Command("git", "rev-list", arg)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Split by newline to get each commit hash.
	commits := strings.Split(strings.TrimSpace(string(out)), "\n")
	return commits, nil
}

func CloneRepo(url, dir string) error {
	cmd := exec.Command("git", "clone", "--bare", url, dir)
	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func GetCommitsFromHashs(repoPath string, hashs []string) ([]io.Commit, error) {
	return getCommit(show, repoPath, hashs)
}

func GetCommitsFromBranch(repoPath, branch string) ([]io.Commit, error) {
	return getCommit(log, repoPath, []string{branch})
}

func GetPatchIdAndUnsignedCommits(repoPath, branch string, cache *PatchIdCache) (*map[string]*io.Commit, *[]io.Commit, error) {
	allCommits, err := GetCommitsFromBranch(repoPath, branch)
	if err != nil {
		return nil, nil, err
	}
	numberCommits := len(allCommits)
	slog.Default().Info("Number all commits", branch, numberCommits)

	patchIdToCommit := make(map[string]*io.Commit, len(allCommits))
	unsignedCommits := make([]io.Commit, 0, len(allCommits)/3)

	for i := range allCommits {
		c := &allCommits[i]
		pi, err := cache.GetOrCreatePatchId(repoPath, c.GitOID)
		if err != nil || pi == "" {
			slog.Default().Debug("Get patch id failed or is empty. Setting patch id to original commit id", "err", err)
			pi = c.GitOID
		}
		patchIdToCommit[pi] = c
		if c.Signed == "N" || c.Signed == "B" {
			unsignedCommits = append(unsignedCommits, *c)
		}
	}

	return &patchIdToCommit, &unsignedCommits, nil
}

type GitCmd string

const (
	show GitCmd = "show"
	log  GitCmd = "log"
)

type DELIMITER string

const (
	lineBreak DELIMITER = "<<<CUSTOM_LINEBREAK>>>"
	value     DELIMITER = "<<<VALUE>>>"
)

func getCommit(gitCmd GitCmd, repoPath string, input []string) ([]io.Commit, error) {
	format := "--pretty=tformat:%H" + value + "%f %b" + value + "%ci" + value + "%G?" + value + lineBreak
	args := append([]string{string(gitCmd), "--no-patch", "--expand-tabs", string(format)}, input...)

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		slog.Default().Error("error during get commit", "err", err, "input", input)
		return nil, err
	}
	str := removeControls(string(out))
	// Split by newline to get each commit hash.
	rawCommits := strings.Split(strings.TrimSuffix(str, string(lineBreak)), string(lineBreak))

	return parseCommits(rawCommits), nil
}

func removeControls(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1 // drop it
		}
		return r // keep it
	}, s)
}

func parseCommits(rawCommits []string) []io.Commit {
	commits := make([]io.Commit, 0, len(rawCommits))
	for _, rc := range rawCommits {
		split := strings.Split(rc, string(value))

		if len(split) < 4 {
			slog.Default().Warn("Commit parsing failed. Split length to short.", "split", split)
			continue
		}

		c := io.Commit{
			GitOID:  split[0],
			Message: split[1],
			Date:    split[2],
			Signed:  split[3],
		}
		commits = append(commits, c)
	}

	return commits
}
