package integrationtests

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path"
	"project-integrity-calculator/internal/io"
	"strings"
	"testing"

	"github.com/hashicorp/go-set/v3"
)

type Commits struct {
	Commits []string `json:"commits"`
}

func TestMain(t *testing.T) {
	// run multirepo with TestRepos.json
	// write result data into tmp
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		t.Fatalf("Failed to get token from environment. GH_TOKEN must be set")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd. err:%s", err)
	}
	dir := path.Join(wd, "..")

	out := path.Join(t.TempDir(), "results")
	app := path.Join(dir, "cmd", "multiRepo", "Main.go")
	in := path.Join(dir, "integrationTests", "TestRepos.json")

	cmd := exec.Command("go", "run", app, "--in", in, "--out", out, "--cloneTarget", t.TempDir(), "--token", token)
	cmd.Dir = dir
	o, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Go run failed target dir %s. output %s", dir, o)
	}

	// read TestRepos, get URL, get expectedResults.json from repo
	file, err := os.Open(path.Join(dir, "integrationTests", "TestRepos.json"))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var input io.Input
	if err := decoder.Decode(&input); err != nil {
		t.Fatalf("Couldn't read TestRepos.json. Err %s", err)
	}

	for _, repo := range input.Data.Search.Nodes {
		results := repo.Url
		resp, err := http.DefaultClient.Get(results)
		if err != nil {
			t.Errorf("Couldn't get results for %+v. Err %s", repo, err)
		}
		defer resp.Body.Close()

		var expectedCommits Commits
		decoder = json.NewDecoder(resp.Body)
		if err = decoder.Decode(&expectedCommits); err != nil {
			t.Errorf("Couldn't decode expcted commits for %+v. Err %s", repo, err)
		}

		ownerRepoSplit := strings.Split(repo.NameWithOwner, "/")

		outResultPath := path.Join(out, ownerRepoSplit[0]+ownerRepoSplit[1]+"-result.json")
		resultFile, err := os.Open(outResultPath)
		if err != nil {
			t.Errorf("Read result file %s. Err %s", outResultPath, err)
		}
		defer resultFile.Close()

		decoder = json.NewDecoder(resultFile)
		var result io.Repo
		if err = decoder.Decode(&result); err != nil {
			t.Errorf("Failed to decode result %s. Err %s", outResultPath, err)
		}

		// compare output with expectedResults.json
		if len(result.CommitsWithoutPR) != len(expectedCommits.Commits) {
			t.Errorf("Unexpected number of commits. Repo %+v", repo)
		}

		expectedSet := set.From(expectedCommits.Commits)
		resultSet := set.New[string](len(result.CommitsWithoutPR))
		for _, c := range result.CommitsWithoutPR {
			resultSet.Insert(c.GitOID)
		}

		equal := resultSet.Equal(expectedSet)
		if !equal {
			t.Errorf("Result and expected sets are not equal. Result %+v, Expected %+v. Repo %+v", resultSet, expectedSet, repo)
		}

	}
}
