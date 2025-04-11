package vcs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type PatchIdCache struct {
	maxSize int
	cache   map[string]string
}

func NewPatchIdCache(size int) *PatchIdCache {
	c := PatchIdCache{
		cache:   make(map[string]string),
		maxSize: size,
	}
	return &c
}

func (c *PatchIdCache) Add(key string, value string) {
	if len(c.cache) > c.maxSize {
		slog.Default().Info("Max cache size reached")
		deleteTarget := len(c.cache) / 10
		slog.Default().Info("Removing entries", "to remove", deleteTarget)
		deleteCounter := 0
		for h := range c.cache {
			delete(c.cache, h)
			deleteCounter++
			if deleteCounter == deleteTarget {
				break
			}
		}
	}
	slog.Default().Debug("Cache add", "key", key)
	c.cache[key] = value
}

func (c *PatchIdCache) Get(key string) *string {
	r := c.cache[key]
	if r == "" {
		slog.Default().Debug("Cache miss", "key", key)
		return nil
	}
	slog.Default().Debug("Cache hit", "key", key)
	return &r
}

func getPatchContext(ctx context.Context, c *object.Commit) (*object.Patch, error) {
	fromTree, err := c.Tree()
	if err != nil {
		return nil, err
	}

	toTree := &object.Tree{}
	if c.NumParents() != 0 {
		firstParent, err := c.Parents().Next()
		if err != nil {
			return nil, err
		}

		toTree, err = firstParent.Tree()
		if err != nil {
			return nil, err
		}
	}

	return toTree.PatchContext(ctx, fromTree)
}

func InternalGetPatchId(ctx context.Context, c *object.Commit) (*plumbing.Hash, error) {
	patch, err := getPatchContext(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get patch ctx for commit %s: %w", c.Hash, err)
	}

	chunks := ""
	for _, p := range patch.FilePatches() {
		for _, c := range p.Chunks() {
			chunks += c.Content()
		}
	}
	h := plumbing.ComputeHash(plumbing.AnyObject, []byte(chunks))
	return &h, nil
}

var cache = NewPatchIdCache(10_000_000)

func GetPatchId(dir string, c *object.Commit) (string, error) {

	hash := c.Hash.String()

	cacheResult := cache.Get(hash)
	if cacheResult != nil {
		return *cacheResult, nil
	}

	patchId, err := InternalGetPatchId(context.Background(), c)
	if err != nil {
		return "", err
	}

	// showCmd := exec.Command("git", "show", hash)
	// showCmd.Dir = dir
	// out, err := showCmd.StdoutPipe()
	// if err != nil {
	// 	return "", err
	// }

	// patchIdCmd := exec.Command("git", "patch-id", "--stable")
	// patchIdCmd.Dir = dir
	// patchIdCmd.Stdin = out

	// patchOut, err := patchIdCmd.StdoutPipe()
	// if err != nil {
	// 	return "", err
	// }

	// patchIdCmd.Start()
	// showCmd.Start()

	// scanner := bufio.NewScanner(patchOut)

	// output := ""
	// for scanner.Scan() {
	// 	output += scanner.Text()
	// }

	// showCmd.Wait()
	// patchIdCmd.Wait()

	// patchId := strings.Split(output, " ")[0]
	cache.Add(hash, patchId.String())

	return patchId.String(), nil
}
