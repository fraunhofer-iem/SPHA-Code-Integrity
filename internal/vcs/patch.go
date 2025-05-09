package vcs

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
)

type PatchIdCache struct {
	maxSize int
	cache   map[string]string
	rw      sync.RWMutex
}

func NewPatchIdCache(size int) *PatchIdCache {
	c := PatchIdCache{
		cache:   make(map[string]string),
		maxSize: size,
		rw:      sync.RWMutex{},
	}
	return &c
}

func (c *PatchIdCache) Add(key string, value string) {
	c.rw.Lock()
	defer c.rw.Unlock()

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
	c.cache[key] = value
}

func (c *PatchIdCache) get(key string) *string {
	c.rw.RLock()
	defer c.rw.RUnlock()
	r := c.cache[key]
	if r == "" {
		return nil
	}
	return &r
}

func (c *PatchIdCache) GetOrCreatePatchId(dir string, hash string) (string, error) {

	cacheResult := c.get(hash)
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
	c.Add(hash, patchId)

	return patchId, nil
}
