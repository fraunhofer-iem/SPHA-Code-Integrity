package processor

import (
	"log/slog"
	"os"
	"path"
	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"
	"project-integrity-calculator/internal/tasks"
	"project-integrity-calculator/internal/vcs"
	"time"
)

type RepoConfig struct {
	Owner, Repo, Branch, Token, ClonePath, Out string
}

func ProcessRepo(config RepoConfig) (*io.Repo, error) {

	logger := slog.Default()
	timer := time.Now()
	logger.Info("Started processing of", "repo with config", config)

	r, err := gh.GetRepoInfo(config.Owner, config.Repo, config.Token)
	if err != nil {
		return nil, err
	}

	var dir string
	if config.ClonePath != "" {
		dir = config.ClonePath
	} else {
		dir = path.Join(os.TempDir(), "repos")
	}
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = vcs.CloneRepo(r.CloneUrl, dir)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	var branch string
	if config.Branch != "" {
		branch = config.Branch
	} else {
		branch = r.DefaultBranch
	}

	methodTimer := time.Now()
	allCommits, err := vcs.GetCommitsFromBrach(dir, branch)
	if err != nil {
		return nil, err
	}
	numberCommits := len(allCommits)
	logger.Info("Number all commits", branch, numberCommits)

	patchIdToCommit := make(map[string]*io.Commit, len(allCommits))
	unsignedCommits := make([]io.Commit, 0, len(allCommits)/3)

	for i := range allCommits {
		c := &allCommits[i]
		pi, err := vcs.GetPatchId(dir, c.GitOID)
		if err != nil || pi == "" {
			slog.Default().Debug("Get patch id failed or is empty. Setting patch id to original commit id", "err", err)
			pi = c.GitOID
		}
		patchIdToCommit[pi] = c
		if c.Signed == "N" || c.Signed == "B" {
			unsignedCommits = append(unsignedCommits, *c)
		}
	}

	elapsed := time.Since(methodTimer)
	logger.Info("query all commits", "time", elapsed)

	methodTimer = time.Now()
	prIter := gh.GetPullRequests(config.Owner, config.Repo, branch, config.Token)

	worker := tasks.Worker[[]gh.PR, []string]{
		Do: func(prs *[]gh.PR) (*[]string, error) {
			commitsFromPrs, err := vcs.GetCommitShaForMergedPr(*prs, dir)
			if err != nil {
				return nil, err
			}
			ids := make([]string, 0, len(*commitsFromPrs))
			for _, cs := range *commitsFromPrs {
				for c := range cs.Items() {
					pi, err := vcs.GetPatchId(dir, c)
					if err != nil || pi == "" {
						slog.Default().Debug("Get patch id failed. Setting patch id to original commit id", "err", err)
						pi = c
					}
					ids = append(ids, pi)
				}
			}
			return &ids, nil
		},
	}

	collect := func(bufferedHashs []*[]string) error {
		logger.Debug("Collecting hashes", "bufferedHashs", bufferedHashs)
		for _, buff := range bufferedHashs {
			for _, h := range *buff {
				delete(patchIdToCommit, h)
			}
		}
		return nil
	}

	collector := tasks.NewBufferedCollector(collect, tasks.BufferedCollectorConfig{})

	noWorker := 3
	dispatcher := tasks.NewDispatcher(worker, prIter, *collector, tasks.DispatcherConfig{NoWorker: &noWorker})
	dispatcher.Dispatch()

	commitsWithoutPr := make([]io.Commit, 0, len(patchIdToCommit))
	for _, c := range patchIdToCommit {
		commitsWithoutPr = append(commitsWithoutPr, *c)
	}

	elapsed = time.Since(methodTimer)
	logger.Info("processed all PRs", "time", elapsed)

	logger.Info("Number commits without PR", "number", len(patchIdToCommit))

	heads, err := vcs.GetCommitsFromHashs(dir, []string{branch})
	head := ""
	if err == nil || len(heads) == 1 {
		head = heads[0].GitOID
	}

	repo := io.Repo{
		Branch:           branch,
		Url:              r.CloneUrl,
		Head:             head,
		CommitsWithoutPR: commitsWithoutPr,
		UnsignedCommits:  unsignedCommits,
		Stats: io.Stats{
			NumberCommits: numberCommits,
			NumberPRs:     0,
			Stars:         r.Stars,
			Languages:     r.Languages,
		},
	}

	timerEnd := time.Since(timer)
	logger.Info("Processing of repo finished", "repo", config.Repo, "time", timerEnd)

	return &repo, nil
}
