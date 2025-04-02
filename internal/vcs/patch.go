package vcs

import (
	"bufio"
	"log/slog"
	"os/exec"
	"strings"
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

func GetPatchId(dir string, hash string) (string, error) {

	cacheResult := cache.Get(hash)
	if cacheResult != nil {
		return *cacheResult, nil
	}

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
	cache.Add(hash, patchId)

	return patchId, nil
}
