package contextconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRuntimeExplicitContext(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("INTERCUBE_NON_INTERACTIVE", "")

	runtime := DetectRuntime("repository", t.TempDir())
	if runtime.Kind != ContextRepository {
		t.Fatalf("expected repository context, got %s", runtime.Kind)
	}
	if !runtime.Explicit {
		t.Fatalf("expected explicit context to be true")
	}
}

func TestDetectRuntimeCIPipeline(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("INTERCUBE_NON_INTERACTIVE", "")

	runtime := DetectRuntime("", t.TempDir())
	if runtime.Kind != ContextPipeline {
		t.Fatalf("expected pipeline context, got %s", runtime.Kind)
	}
	if !runtime.NonInteractive {
		t.Fatalf("expected pipeline context to be non-interactive")
	}
}

func TestDetectRuntimeRepository(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("INTERCUBE_NON_INTERACTIVE", "")

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, defaultUserConfigName), []byte("context:\n  site_id: '1'\n"), 0644); err != nil {
		t.Fatalf("unable to write temp config: %v", err)
	}

	nested := filepath.Join(repoRoot, "sub", "dir")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("unable to create nested dir: %v", err)
	}

	runtime := DetectRuntime("", nested)
	if runtime.Kind != ContextRepository {
		t.Fatalf("expected repository context, got %s", runtime.Kind)
	}
	if runtime.RepositoryRoot != repoRoot {
		t.Fatalf("expected repository root %s, got %s", repoRoot, runtime.RepositoryRoot)
	}
}
