package runner

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

// MergeEnv merges base environment entries with overrides.
// Override keys replace any existing entries with the same key; other base
// entries are preserved. The result never contains duplicate keys for overrides.
func MergeEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		out := make([]string, len(base))
		copy(out, base)
		return out
	}

	overrideKeys := make(map[string]struct{}, len(overrides))
	for k := range overrides {
		overrideKeys[k] = struct{}{}
	}

	out := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			out = append(out, entry)
			continue
		}
		if _, overridden := overrideKeys[key]; overridden {
			continue
		}
		out = append(out, entry)
	}

	// Stable order for deterministic tests and shell rendering.
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, k+"="+overrides[k])
	}
	return out
}

// Environ returns the process environment for executing spec.
// When spec has no Env overrides, nil is returned so exec.Cmd inherits the
// current process environment (including any user-provided DEBUGGER).
func Environ(spec CommandSpec) []string {
	overrides, err := envSliceToMap(spec.Env)
	if err != nil || len(overrides) == 0 {
		return nil
	}
	return MergeEnv(os.Environ(), overrides)
}

func envSliceToMap(entries []string) (map[string]string, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env entry %q: missing '='", entry)
		}
		if err := validateEnvKey(key); err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, nil
}

func envMapToSlice(overrides map[string]string) ([]string, error) {
	if len(overrides) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		if err := validateEnvKey(k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+overrides[k])
	}
	return out, nil
}

// validateEnvKey ensures key is a legal POSIX shell variable name.
func validateEnvKey(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable name must not be empty")
	}
	for i, r := range key {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return fmt.Errorf("invalid environment variable name %q", key)
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("invalid environment variable name %q", key)
		}
	}
	return nil
}
