package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveForGitRoot(t *testing.T) {
	t.Run("returns root from nested dir", func(t *testing.T) {
		repo := newTempRepo(t)
		nested := filepath.Join(repo, "nested", "workspace")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}

		root, err := ResolveForGitRoot(nested)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != repo {
			t.Fatalf("expected %s, got %s", repo, root)
		}
	})

	t.Run("returns error when git dir missing", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := ResolveForGitRoot(dir); err == nil {
			t.Fatalf("expected error but got nil")
		}
	})

	t.Run("follows symlinked start path", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation requires admin rights on Windows")
		}

		repo := newTempRepo(t)
		nested := filepath.Join(repo, "nested")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}

		link := filepath.Join(repo, "nested-link")
		if err := os.Symlink(nested, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		root, err := ResolveForGitRoot(link)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != repo {
			t.Fatalf("expected %s, got %s", repo, root)
		}
	})
}

func newTempRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	return repo
}
