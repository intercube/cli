package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSymlinkFailsWhenOriginIsMissing(t *testing.T) {
	tempDir := t.TempDir()
	origin := filepath.Join(tempDir, "shared", "sitemaps")
	destination := filepath.Join(tempDir, "current", "sitemaps")

	err := symlink(origin, destination, false)
	if err == nil {
		t.Fatalf("expected missing origin to fail")
	}
	if !strings.Contains(err.Error(), "origin file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSymlinkCreatesMissingOriginDirectory(t *testing.T) {
	tempDir := t.TempDir()
	origin := filepath.Join(tempDir, "shared", "sitemaps")
	destination := filepath.Join(tempDir, "current", "sitemaps")

	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatalf("unable to create destination parent: %v", err)
	}

	if err := symlink(origin, destination, true); err != nil {
		t.Fatalf("expected missing origin directory to be created: %v", err)
	}

	originInfo, err := os.Stat(origin)
	if err != nil {
		t.Fatalf("expected origin directory to exist: %v", err)
	}
	if !originInfo.IsDir() {
		t.Fatalf("expected origin to be a directory")
	}

	destinationInfo, err := os.Lstat(destination)
	if err != nil {
		t.Fatalf("expected destination symlink to exist: %v", err)
	}
	if destinationInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected destination to be a symlink")
	}

	target, err := os.Readlink(destination)
	if err != nil {
		t.Fatalf("unable to read destination symlink: %v", err)
	}
	if target != origin {
		t.Fatalf("unexpected symlink target: got %q want %q", target, origin)
	}
}
