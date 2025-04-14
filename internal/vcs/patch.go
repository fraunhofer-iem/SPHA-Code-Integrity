package vcs

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

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

var cache = NewPatchIdCache(10_000_000)

func GetPatchId(dir string, c *object.Commit) (string, error) {

	hash := c.Hash.String()

	cacheResult := cache.Get(hash)
	if cacheResult != nil {
		return *cacheResult, nil
	}

	cmd := exec.Command("sh", "-c", fmt.Sprintf("git show %s | git patch-id --stable", hash))
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	patchId := strings.Split(string(out), " ")[0]
	cache.Add(hash, patchId)

	return patchId, nil
}
