package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func PatchId(t *testing.T) {
	fmt.Println("hello world")
	tmpGit := path.Join(t.TempDir(), "testGit")

	err := os.Mkdir(tmpGit, os.ModePerm)
	if err != nil {
		t.FailNow()
	}

	gitInit := exec.Command("git", "init", ".")
	gitInit.Dir = tmpGit
	gitInit.Run()

	touchFile := exec.Command("touch", "A")
	gitInit.Dir = tmpGit
	touchFile.Run()

	add := exec.Command("git", "add", "A")
	add.Dir = tmpGit

	commit := exec.Command("git", "commit", "-m", "test")
	commit.Dir = tmpGit

	repo, err := git.PlainOpen(tmpGit)

	commitIter, err := repo.CommitObjects()
	commitIter.ForEach(func(c *object.Commit) error {
		fmt.Printf("%+v\n", c)
		patchId, err := GetPatchId(tmpGit, c.Hash.String())
		if err != nil {
			return err
		}
		fmt.Printf("patch id %s\n", patchId)

		return nil
	})

}
