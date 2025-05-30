package processor

import (
	"log/slog"
	"os"
	"path"
	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"
	"project-integrity-calculator/internal/vcs"
	"time"

	"github.com/janniclas/beehive"
)

type RepoConfig struct {
	Owner, Repo, Branch, Token, ClonePath, Out string
	IgnoreFirstCommits                         bool
}

func ProcessRepo(config RepoConfig) (*io.Repo, error) {

	logger := slog.Default()
	timer := time.Now()
	logger.Info("Started processing of", "repo with config", config)

	r, err := gh.GetRepoInfo(config.Owner, config.Repo, config.Token)
	if err != nil {
		return nil, err
	}

	// setup clone path and clone repo
	dir := config.ClonePath
	if dir == "" {
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

	branch := config.Branch
	if config.Branch == "" {
		branch = r.DefaultBranch
	}

	cache := vcs.NewPatchIdCache(10_000_000)
	methodTimer := time.Now()
	patchIdToCommit, unsignedCommits, err := vcs.GetPatchIdAndUnsignedCommits(dir, branch, cache)
	numberCommits := len(*patchIdToCommit)

	elapsed := time.Since(methodTimer)
	logger.Info("query all commits", "time", elapsed)

	methodTimer = time.Now()
	prIter := gh.GetPullRequests(config.Owner, config.Repo, branch, config.Token)
	var work func(p *[]gh.PR) (*WorkerResult, error)
	if config.IgnoreFirstCommits {
		work = WorkerWithNewestPr(dir, cache)
	} else {
		work = WorkerWithoutNewestPr(dir, cache)
	}

	noOfForcePushes, err := gh.GetForcePushInfo(config.Owner, config.Repo, config.Token, branch)

	worker := beehive.Worker[[]gh.PR, WorkerResult]{
		Work: work,
	}

	var firstPR *gh.PR = nil
	// this implementation relys on the fact that there is only one collector at all times so no
	// race conditions can happen
	collect := func(workerResults []*WorkerResult) error {
		logger.Debug("Collecting hashes", "bufferedHashs", workerResults)
		for i := range workerResults {
			res := workerResults[i]
			for _, h := range *&res.PatchIds {
				delete(*patchIdToCommit, h)
			}
			if config.IgnoreFirstCommits && (firstPR == nil || (res.NewestPr != nil && res.NewestPr.MergedAt < firstPR.MergedAt)) {
				firstPR = res.NewestPr
			}
		}
		return nil
	}

	collector := beehive.NewBufferedCollector(collect, beehive.BufferedCollectorConfig{})

	// memory profiling for the Benchmark showed some larger memory spikes so we limit the number of worker to 6
	// which works for our 256GB RAM VM
	numWorker := 6
	dispatcher := beehive.NewDispatcher(worker, prIter, *collector, beehive.DispatcherConfig{NumWorker: &numWorker})
	dispatcher.Dispatch()

	if config.IgnoreFirstCommits && firstPR != nil {
		logger.Info("First PR", "pr", *firstPR)
		// identify commits before newest PRs and whitelist them
		// newestPr.HeadRefOid
		for h, k := range *patchIdToCommit {
			if k.Date < firstPR.MergedAt {
				delete(*patchIdToCommit, h)
			}
		}
	}

	commitsWithoutPr := make([]io.Commit, 0, len(*patchIdToCommit))
	for _, c := range *patchIdToCommit {
		commitsWithoutPr = append(commitsWithoutPr, *c)
	}

	elapsed = time.Since(methodTimer)
	logger.Info("processed all PRs", "time", elapsed)

	logger.Info("Number commits without PR", "number", len(*patchIdToCommit))

	heads, err := vcs.GetCommitsFromHashs(dir, []string{branch})
	head := ""
	if err == nil && len(heads) == 1 {
		head = heads[0].GitOID
	}

	repo := io.Repo{
		Branch:            branch,
		Url:               r.CloneUrl,
		NumberForcePushes: noOfForcePushes,
		Head:              head,
		CommitsWithoutPR:  commitsWithoutPr,
		UnsignedCommits:   *unsignedCommits,
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

type WorkerResult struct {
	PatchIds []string
	NewestPr *gh.PR
}

func WorkerWithoutNewestPr(dir string, cache *vcs.PatchIdCache) func(p *[]gh.PR) (*WorkerResult, error) {
	return func(p *[]gh.PR) (*WorkerResult, error) {
		if len(*p) == 0 {
			return &WorkerResult{}, nil
		}

		prs := *p
		firstPR := prs[0]

		commitsFromPrs, err := vcs.GetCommitShaForMergedPr(prs, dir)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(*commitsFromPrs))
		for _, cs := range *commitsFromPrs {
			for c := range cs.Items() {
				pi, err := cache.GetOrCreatePatchId(dir, c)
				if err != nil || pi == "" {
					slog.Default().Debug("Get patch id failed. Setting patch id to original commit id", "err", err)
					pi = c
				}
				ids = append(ids, pi)
			}
		}
		return &WorkerResult{
			PatchIds: ids,
			NewestPr: &firstPR,
		}, nil
	}
}

func WorkerWithNewestPr(dir string, cache *vcs.PatchIdCache) func(p *[]gh.PR) (*WorkerResult, error) {
	return func(p *[]gh.PR) (*WorkerResult, error) {
		if len(*p) == 0 {
			return &WorkerResult{}, nil
		}

		prs := *p
		newestPr := prs[0]

		for i, pr := range prs {
			if pr.MergedAt < newestPr.MergedAt {
				newestPr = prs[i]
			}
		}
		commitsFromPrs, err := vcs.GetCommitShaForMergedPr(prs, dir)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(*commitsFromPrs))
		for _, cs := range *commitsFromPrs {
			for c := range cs.Items() {
				pi, err := cache.GetOrCreatePatchId(dir, c)
				if err != nil || pi == "" {
					slog.Default().Debug("Get patch id failed. Setting patch id to original commit id", "err", err)
					pi = c
				}
				ids = append(ids, pi)
			}
		}
		return &WorkerResult{
			PatchIds: ids,
			NewestPr: &newestPr,
		}, nil
	}
}
