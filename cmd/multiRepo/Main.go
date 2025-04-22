package main

import (
	"flag"
	"os"
	"path"
	"project-integrity-calculator/internal/logging"
	"strings"
	"time"
)

var (
	ownerAndRepo = flag.String("ownerAndRepo", "", "GitHub repository link (e.g., https://github.com/owner/repo)")
	token        = flag.String("token", "", "GitHub access token")
	targetBranch = flag.String("branch", "", "Target branch to analyze. Defaults to the default branch of the repository")
	mode         = flag.String("mode", "local", "Mode: 'local' or 'clone'")
	localPath    = flag.String("localPath", "", "Path to the local repository (required if mode is 'local')")
	cloneTarget  = flag.String("cloneTarget", "", "Target to clone. Defaults to tmp")
	logLevel     = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
)

func main() {

	start := time.Now()
	// TODO: add input validation
	flag.Parse()

	logger := logging.SetUpLogging(*logLevel)

	if *ownerAndRepo == "" {
		panic("ownerAndRepo is required")
	}

	ownerAndRepoSplit := strings.Split(*ownerAndRepo, "/")
	// TODO: do validation for format

	if *token == "" {
		panic("token is required")
	}

	if *mode == "local" && *localPath == "" {
		panic("localPath is required if mode is 'local'")
	}

	if *mode == "clone" && *cloneTarget == "" {
		*cloneTarget = path.Join(os.TempDir(), "codeintegrity", ownerAndRepoSplit[1])
	}

	elapsed := time.Since(start)
	logger.Info("Execution finished", "time elapsed", elapsed)
}
