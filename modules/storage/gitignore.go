package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureGitignoreEntry adds the data directory to the project's .gitignore so
// generated database files aren't committed by accident. It creates the
// .gitignore if none exists and is a no-op if the entry is already present.
// This is best-effort: it never prevents the module from registering, but
// genuine failures (as opposed to "no .gitignore yet") are reported on
// os.Stderr so the omission isn't silently hidden from the developer.
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

// warnGitignore reports a failure to update the project's .gitignore for the
// given data directory without aborting module registration.
func warnGitignore(path string, err error) {
	fmt.Fprintf(os.Stderr, "store: could not add %q to .gitignore: %v\n", path, err)
}

// findProjectRoot walks up from start looking for a .git directory or file
// (worktrees use a file). Returns "" if none is found.
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
