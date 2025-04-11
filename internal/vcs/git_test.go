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

func setupGit(t *testing.T) string {
	tmpGit := path.Join(t.TempDir(), "testGit")

	if err := os.Mkdir(tmpGit, os.ModePerm); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	commands := []struct {
		name string
		cmd  *exec.Cmd
	}{
		{"git init", exec.Command("git", "init", ".")},
		{"touch A", exec.Command("touch", "A")},
		{"git add A", exec.Command("git", "add", "A")},
		{"git commit", exec.Command("git", "commit", "-m", "test")},
		{"touch B", exec.Command("touch", "B")},
		{"git add B", exec.Command("git", "add", "B")},
		{"git commit", exec.Command("git", "commit", "-m", "added B")},
	}

	for _, c := range commands {
		c.cmd.Dir = tmpGit
		c.cmd.Stdout = os.Stdout
		c.cmd.Stderr = os.Stderr
		if err := c.cmd.Run(); err != nil {
			t.Fatalf("Failed to execute '%s': %v", c.name, err)
		}
	}

	return tmpGit

}

func TestPatchDiff(t *testing.T) {

	// tmpGit := setupGit(t)
	repo, err := git.PlainOpen("/Users/struewer/git/spha/spha.git")

	if err != nil {
		t.Fatalf("Failed to open git repository: %v", err)
	}

	commitIter, err := repo.CommitObjects()
	if err != nil {
		t.Fatalf("Failed to retrieve commit objects: %v", err)
	}

	err = commitIter.ForEach(func(c *object.Commit) error {
		patchId, err := InternalGetPatchId(t.Context(), c)
		if err != nil {
			t.Fatalf("Patch Id calculation failed")
		}

		t.Logf("Patch ID, %+v", patchId)

		return nil
	})

}

func TestPatchId(t *testing.T) {

	tmpGit := setupGit(t)

	repo, err := git.PlainOpen(tmpGit)
	if err != nil {
		t.Fatalf("Failed to open git repository: %v", err)
	}

	commitIter, err := repo.CommitObjects()
	if err != nil {
		t.Fatalf("Failed to retrieve commit objects: %v", err)
	}

	err = commitIter.ForEach(func(c *object.Commit) error {
		patchId, err := GetPatchId(tmpGit, c.Hash.String())
		if err != nil {
			return fmt.Errorf("failed to get patch ID for commit %s: %w", c.Hash, err)
		}
		fmt.Printf("patch id %s\n", patchId)
		return nil
	})
	if err != nil {
		t.Fatalf("Error iterating over commits: %v", err)
	}
}
