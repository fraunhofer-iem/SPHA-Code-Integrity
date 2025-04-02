package vcs

import (
	"bufio"
	"log/slog"
	"os/exec"
	"strings"
)

type PatchIdCache struct {
	logger *slog.Logger
	cache  map[string]string
}

func NewPatchIdCache() *PatchIdCache {
	c := PatchIdCache{
		cache:  make(map[string]string),
		logger: slog.Default(),
	}
	return &c
}

func (c *PatchIdCache) Add(key string, value string) {
	c.logger.Debug("Cache add", "key", key)
	c.cache[key] = value
}

func (c *PatchIdCache) Get(key string) *string {
	r := c.cache[key]
	if r == "" {
		c.logger.Debug("Cache miss", "key", key)
		return nil
	}
	c.logger.Debug("Cache hit", "key", key)
	return &r
}

var cache = NewPatchIdCache()

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
