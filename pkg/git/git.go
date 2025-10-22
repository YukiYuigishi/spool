package git

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

var gitdirs = []string{
	".git/branches",
	".git/hooks",
	".git/info",
	".git/logs/refs/heads",
	".git/objects/info",
	".git/objects/pack",
	".git/refs/heads",
	".git/refs/tags",
}

var defaultConfig = `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
`

var defaultDescription = "Unnamed repository; edit this file 'description' to name the repository.\n"

var defaultBranch = "master"
var defaultHead = fmt.Sprintf("ref: refs/heads/%s\n", defaultBranch)
var defaultExclude = ""

var defaultFiles = []struct {
	Name     string
	Content  string
	FileMode os.FileMode
}{
	{".git/config", defaultConfig, 0644},
	{".git/description", defaultDescription, 0644},
	{".git/HEAD", defaultHead, 0644},
	{".git/info/exclude", defaultExclude, 0644},
}

func InitGitRepository() {
	cwd, err := os.Getwd()

	if err != nil {
		slog.Debug("init", slog.String("error", err.Error()))
		os.Exit(1)
	}

	for _, dir := range gitdirs {
		slog.Debug("init", slog.String("dir", filepath.Join(cwd, dir)))
		os.MkdirAll(dir, 0755)
	}

	for _, file := range defaultFiles {
		slog.Debug("init", slog.String("file", filepath.Join(cwd, file.Name)))
		os.WriteFile(file.Name, []byte(file.Content), file.FileMode)
	}
}
