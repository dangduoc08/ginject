package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensureGitignoreEntry(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		warnGitignore(path, err)
		return
	}

	root := findProjectRoot(abs)
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			warnGitignore(path, err)
			return
		}
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = abs
	}
	entry := strings.TrimSuffix(filepath.ToSlash(rel), "/")

	gitignorePath := filepath.Join(root, ".gitignore")
	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		warnGitignore(path, err)
		return
	}

	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSuffix(strings.TrimSpace(line), "/") == entry {
			return
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		warnGitignore(path, err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			warnGitignore(path, err)
		}
	}()

	line := entry + "/\n"
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		line = "\n" + line
	}
	if _, err := f.WriteString(line); err != nil {
		warnGitignore(path, err)
	}
}

func warnGitignore(path string, err error) {
	fmt.Fprintf(os.Stderr, "store: could not add %q to .gitignore: %v\n", path, err)
}

func findProjectRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
