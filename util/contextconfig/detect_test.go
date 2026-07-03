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
	if err := os.WriteFile(filepath.Join(repoRoot, configFileNames[0]), []byte("context:\n  site_id: '1'\n"), 0644); err != nil {
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

func TestDetectRuntimeRepositoryYmlExtension(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("INTERCUBE_NON_INTERACTIVE", "")

	repoRoot := t.TempDir()
	ymlPath := filepath.Join(repoRoot, ".intercube.yml")
	if err := os.WriteFile(ymlPath, []byte("context:\n  site_id: '1'\n"), 0644); err != nil {
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
	if runtime.ActiveConfigPath != ymlPath {
		t.Fatalf("expected active config path %s, got %s", ymlPath, runtime.ActiveConfigPath)
	}
}

func TestResolveConfigPath(t *testing.T) {
	t.Run("only yml exists", func(t *testing.T) {
		dir := t.TempDir()
		ymlPath := filepath.Join(dir, ".intercube.yml")
		if err := os.WriteFile(ymlPath, []byte("{}\n"), 0644); err != nil {
			t.Fatalf("unable to write temp config: %v", err)
		}
		if got := ResolveConfigPath(dir); got != ymlPath {
			t.Fatalf("expected %s, got %s", ymlPath, got)
		}
	})

	t.Run("only yaml exists", func(t *testing.T) {
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, ".intercube.yaml")
		if err := os.WriteFile(yamlPath, []byte("{}\n"), 0644); err != nil {
			t.Fatalf("unable to write temp config: %v", err)
		}
		if got := ResolveConfigPath(dir); got != yamlPath {
			t.Fatalf("expected %s, got %s", yamlPath, got)
		}
	})

	t.Run("neither exists defaults to yaml", func(t *testing.T) {
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, ".intercube.yaml")
		if got := ResolveConfigPath(dir); got != yamlPath {
			t.Fatalf("expected canonical %s, got %s", yamlPath, got)
		}
	})

	t.Run("both exist prefers yaml", func(t *testing.T) {
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, ".intercube.yaml")
		if err := os.WriteFile(yamlPath, []byte("{}\n"), 0644); err != nil {
			t.Fatalf("unable to write temp config: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".intercube.yml"), []byte("{}\n"), 0644); err != nil {
			t.Fatalf("unable to write temp config: %v", err)
		}
		if got := ResolveConfigPath(dir); got != yamlPath {
			t.Fatalf("expected %s, got %s", yamlPath, got)
		}
	})
}
