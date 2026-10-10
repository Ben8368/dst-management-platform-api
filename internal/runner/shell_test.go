package runner

import (
	"strings"
	"testing"
)

func TestShellRender_Simple(t *testing.T) {
	spec := CommandSpec{
		Binary: "/usr/bin/box64",
		Args:   []string{"./app", "-flag", "value"},
	}

	result := ShellRender(spec)

	// Should contain quoted binary and all args
	if !strings.Contains(result, "'/usr/bin/box64'") {
		t.Errorf("result should contain quoted binary: %s", result)
	}
	if !strings.Contains(result, "'./app'") {
		t.Errorf("result should contain quoted app: %s", result)
	}
	if !strings.Contains(result, "'-flag'") {
		t.Errorf("result should contain quoted flag: %s", result)
	}
	if !strings.Contains(result, "'value'") {
		t.Errorf("result should contain quoted value: %s", result)
	}
}

func TestShellRender_WithSpaces(t *testing.T) {
	spec := CommandSpec{
		Binary: "/usr/bin/box64",
		Args: []string{
			"./dontstarve_x64",
			"-cluster",
			"My Cluster",
			"-shard",
			"Master World",
		},
	}

	result := ShellRender(spec)

	// Spaces should be preserved within quotes
	if !strings.Contains(result, "'My Cluster'") {
		t.Errorf("result should preserve spaces in arguments: %s", result)
	}
	if !strings.Contains(result, "'Master World'") {
		t.Errorf("result should preserve spaces in arguments: %s", result)
	}
}

func TestShellRender_WithSingleQuotes(t *testing.T) {
	spec := CommandSpec{
		Binary: "/bin/sh",
		Args:   []string{"-c", "echo 'hello'"},
	}

	result := ShellRender(spec)

	// Single quotes in arguments should be escaped properly
	// The ShellQuote function escapes ' as '\''
	if !strings.Contains(result, `'\''`) {
		t.Errorf("result should escape single quotes: %s", result)
	}
}

func TestShellRender_WithDollarSign(t *testing.T) {
	spec := CommandSpec{
		Binary: "/bin/sh",
		Args:   []string{"-c", "echo $HOME"},
	}

	result := ShellRender(spec)

	// Dollar signs should be quoted to prevent shell expansion
	if !strings.Contains(result, "'echo $HOME'") {
		t.Errorf("result should quote dollar signs: %s", result)
	}
}

func TestShellRender_WithSemicolon(t *testing.T) {
	spec := CommandSpec{
		Binary: "/bin/sh",
		Args:   []string{"-c", "ls;rm -rf /"},
	}

	result := ShellRender(spec)

	// Semicolons should be quoted to prevent command injection
	if !strings.Contains(result, "'ls;rm -rf /'") {
		t.Errorf("result should quote semicolons: %s", result)
	}
}

func TestShellRender_WithBackticks(t *testing.T) {
	spec := CommandSpec{
		Binary: "/bin/sh",
		Args:   []string{"-c", "`whoami`"},
	}

	result := ShellRender(spec)

	// Backticks should be quoted to prevent command substitution
	if !strings.Contains(result, "'`whoami`'") {
		t.Errorf("result should quote backticks: %s", result)
	}
}

func TestShellRender_WithNewline(t *testing.T) {
	spec := CommandSpec{
		Binary: "/bin/echo",
		Args:   []string{"line1\nline2"},
	}

	result := ShellRender(spec)

	// Newlines should be preserved within quotes
	if !strings.Contains(result, "'line1\nline2'") {
		t.Errorf("result should preserve newlines: %s", result)
	}
}

func TestShellRender_ComplexDSTCommand(t *testing.T) {
	spec := CommandSpec{
		Binary: "/usr/bin/box64",
		Args: []string{
			"./dontstarve_dedicated_server_nullrenderer_x64",
			"-console",
			"-cluster",
			"Cluster_1",
			"-shard",
			"Master",
		},
	}

	result := ShellRender(spec)

	// Verify all parts are present and quoted
	expectedParts := []string{
		"'/usr/bin/box64'",
		"'./dontstarve_dedicated_server_nullrenderer_x64'",
		"'-console'",
		"'-cluster'",
		"'Cluster_1'",
		"'-shard'",
		"'Master'",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("result should contain %s, got: %s", part, result)
		}
	}

	// Should be space-separated
	parts := strings.Split(result, " ")
	if len(parts) != len(expectedParts) {
		t.Errorf("expected %d space-separated parts, got %d", len(expectedParts), len(parts))
	}
}

func TestShellRender_EmptyArgs(t *testing.T) {
	spec := CommandSpec{
		Binary: "/usr/bin/box64",
		Args:   []string{},
	}

	result := ShellRender(spec)

	if result != "'/usr/bin/box64'" {
		t.Errorf("expected quoted binary only, got: %s", result)
	}
}

func TestShellRender_ArgumentOrder(t *testing.T) {
	spec := CommandSpec{
		Binary: "/bin/app",
		Args:   []string{"arg1", "arg2", "arg3"},
	}

	result := ShellRender(spec)

	// Verify order is preserved
	arg1Pos := strings.Index(result, "'arg1'")
	arg2Pos := strings.Index(result, "'arg2'")
	arg3Pos := strings.Index(result, "'arg3'")

	if arg1Pos == -1 || arg2Pos == -1 || arg3Pos == -1 {
		t.Fatalf("all arguments should be present in result: %s", result)
	}

	if !(arg1Pos < arg2Pos && arg2Pos < arg3Pos) {
		t.Errorf("arguments should maintain order, got: %s", result)
	}
}
