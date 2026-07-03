package contextconfig

import (
	"os"
	"path/filepath"
	"strings"
)

type Kind string

const (
	ContextPipeline   Kind = "pipeline"
	ContextServer     Kind = "server"
	ContextRepository Kind = "repository"
	ContextGlobal     Kind = "global"
)

// configFileNames lists accepted config basenames in priority order.
// .yaml is canonical; .yml is accepted as an equivalent alias.
var configFileNames = []string{".intercube.yaml", ".intercube.yml"}

// ResolveConfigPath returns the path to an existing config file in dir,
// trying each accepted extension in priority order. If none exist it returns
// the canonical (.yaml) path so new files are created canonically.
func ResolveConfigPath(dir string) string {
	for _, name := range configFileNames {
		candidate := filepath.Join(dir, name)
		if fileExists(candidate) {
			return candidate
		}
	}
	return filepath.Join(dir, configFileNames[0])
}

type Runtime struct {
	Kind             Kind
	Explicit         bool
	WorkingDir       string
	UserConfigPath   string
	ActiveConfigPath string
	RepositoryRoot   string
	NonInteractive   bool
}

func DetectRuntime(explicitContext string, workingDir string) Runtime {
	trimmedExplicit := strings.ToLower(strings.TrimSpace(explicitContext))
	kind, explicit := parseKind(trimmedExplicit)

	home, _ := os.UserHomeDir()
	runtime := Runtime{
		Kind:           ContextGlobal,
		Explicit:       explicit,
		WorkingDir:     strings.TrimSpace(workingDir),
		UserConfigPath: ResolveConfigPath(home),
	}

	if explicit {
		runtime.Kind = kind
		populatePaths(&runtime)
		runtime.NonInteractive = computeNonInteractive(runtime.Kind)
		return runtime
	}

	if isTruthy(os.Getenv("CI")) {
		runtime.Kind = ContextPipeline
		populatePaths(&runtime)
		runtime.NonInteractive = computeNonInteractive(runtime.Kind)
		return runtime
	}

	repoRoot := findRepositoryConfigRoot(runtime.WorkingDir)
	if repoRoot != "" {
		runtime.Kind = ContextRepository
		runtime.RepositoryRoot = repoRoot
		populatePaths(&runtime)
		runtime.NonInteractive = computeNonInteractive(runtime.Kind)
		return runtime
	}

	populatePaths(&runtime)
	runtime.NonInteractive = computeNonInteractive(runtime.Kind)
	return runtime
}

func parseKind(value string) (Kind, bool) {
	switch value {
	case string(ContextPipeline):
		return ContextPipeline, true
	case string(ContextServer):
		return ContextServer, true
	case string(ContextRepository):
		return ContextRepository, true
	case string(ContextGlobal):
		return ContextGlobal, true
	default:
		return ContextGlobal, false
	}
}

func populatePaths(runtime *Runtime) {
	switch runtime.Kind {
	case ContextServer:
		runtime.ActiveConfigPath = runtime.UserConfigPath
	case ContextRepository:
		repoRoot := runtime.RepositoryRoot
		if repoRoot == "" {
			repoRoot = findRepositoryConfigRoot(runtime.WorkingDir)
			runtime.RepositoryRoot = repoRoot
		}
		if repoRoot != "" {
			runtime.ActiveConfigPath = ResolveConfigPath(repoRoot)
		}
	default:
		runtime.ActiveConfigPath = ""
	}
}

func computeNonInteractive(kind Kind) bool {
	if kind == ContextPipeline {
		return true
	}

	if isTruthy(os.Getenv("INTERCUBE_NON_INTERACTIVE")) {
		return true
	}

	return false
}

func findRepositoryConfigRoot(start string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}

	current := start
	for {
		for _, name := range configFileNames {
			if fileExists(filepath.Join(current, name)) {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}

		current = parent
	}
}

func isTruthy(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
