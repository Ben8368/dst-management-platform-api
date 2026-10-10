package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// createFakeExecutable creates a temporary fake executable for testing
func createFakeExecutable(t *testing.T, name string) string {
	t.Helper()

	tempDir := t.TempDir()
	exePath := filepath.Join(tempDir, name)

	// Create an empty file
	f, err := os.Create(exePath)
	if err != nil {
		t.Fatalf("failed to create fake executable: %v", err)
	}
	f.Close()

	// Make it executable on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(exePath, 0755); err != nil {
			t.Fatalf("failed to chmod fake executable: %v", err)
		}
	}

	return exePath
}

func TestBuildCompatCommand_AMD64_Native(t *testing.T) {
	config := Config{
		HostOS:   "linux",
		HostArch: "amd64",
	}

	binary := "./myapp"
	args := []string{"-flag", "value"}

	spec, err := BuildCompatCommand(binary, args, Native, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Binary != binary {
		t.Errorf("expected binary %s, got %s", binary, spec.Binary)
	}

	if len(spec.Args) != len(args) {
		t.Fatalf("expected %d args, got %d", len(args), len(spec.Args))
	}

	for i, arg := range args {
		if spec.Args[i] != arg {
			t.Errorf("arg[%d]: expected %s, got %s", i, arg, spec.Args[i])
		}
	}
}

func TestBuildCompatCommand_AMD64_X8664(t *testing.T) {
	config := Config{
		HostOS:   "linux",
		HostArch: "amd64",
	}

	binary := "./dontstarve_dedicated_server_nullrenderer_x64"
	args := []string{"-console", "-cluster", "MyCluster"}

	spec, err := BuildCompatCommand(binary, args, X8664, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AMD64 can run x86_64 natively
	if spec.Binary != binary {
		t.Errorf("expected binary %s, got %s", binary, spec.Binary)
	}

	if len(spec.Args) != len(args) {
		t.Fatalf("expected %d args, got %d", len(args), len(spec.Args))
	}
}

func TestBuildCompatCommand_AMD64_X8632(t *testing.T) {
	config := Config{
		HostOS:   "linux",
		HostArch: "amd64",
	}

	binary := "./dontstarve_dedicated_server_nullrenderer"
	args := []string{"-console"}

	spec, err := BuildCompatCommand(binary, args, X8632, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AMD64 can run x86_32 natively (if multilib is available, which is not runner's concern)
	if spec.Binary != binary {
		t.Errorf("expected binary %s, got %s", binary, spec.Binary)
	}
}

func TestBuildCompatCommand_ARM64_Native(t *testing.T) {
	config := Config{
		HostOS:   "linux",
		HostArch: "arm64",
	}

	binary := "./native_arm64_app"
	args := []string{"-flag"}

	spec, err := BuildCompatCommand(binary, args, Native, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Binary != binary {
		t.Errorf("expected binary %s, got %s", binary, spec.Binary)
	}
}

func TestBuildCompatCommand_ARM64_X8664_WithBox64(t *testing.T) {
	fakeBox64 := createFakeExecutable(t, "box64")

	config := Config{
		HostOS:    "linux",
		HostArch:  "arm64",
		Box64Path: fakeBox64,
	}

	binary := "./dontstarve_dedicated_server_nullrenderer_x64"
	args := []string{"-console", "-cluster", "Cluster 1", "-shard", "Master"}

	spec, err := BuildCompatCommand(binary, args, X8664, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Binary != fakeBox64 {
		t.Errorf("expected binary %s, got %s", fakeBox64, spec.Binary)
	}

	expectedArgs := []string{
		"./dontstarve_dedicated_server_nullrenderer_x64",
		"-console",
		"-cluster",
		"Cluster 1",
		"-shard",
		"Master",
	}

	if len(spec.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d", len(expectedArgs), len(spec.Args))
	}

	for i, expected := range expectedArgs {
		if spec.Args[i] != expected {
			t.Errorf("arg[%d]: expected %s, got %s", i, expected, spec.Args[i])
		}
	}
}

func TestBuildCompatCommand_ARM64_X8632_WithBox86(t *testing.T) {
	fakeBox86 := createFakeExecutable(t, "box86")

	config := Config{
		HostOS:    "linux",
		HostArch:  "arm64",
		Box86Path: fakeBox86,
	}

	binary := "./dontstarve_dedicated_server_nullrenderer"
	args := []string{"-console", "-shard", "Caves"}

	spec, err := BuildCompatCommand(binary, args, X8632, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Binary != fakeBox86 {
		t.Errorf("expected binary %s, got %s", fakeBox86, spec.Binary)
	}

	expectedArgs := []string{
		"./dontstarve_dedicated_server_nullrenderer",
		"-console",
		"-shard",
		"Caves",
	}

	if len(spec.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d", len(expectedArgs), len(spec.Args))
	}

	for i, expected := range expectedArgs {
		if spec.Args[i] != expected {
			t.Errorf("arg[%d]: expected %s, got %s", i, expected, spec.Args[i])
		}
	}
}

func TestBuildCompatCommand_ARM64_X8664_MissingBox64(t *testing.T) {
	// Clear environment variable
	originalEnv := os.Getenv("DMP_BOX64_PATH")
	os.Unsetenv("DMP_BOX64_PATH")
	defer func() {
		if originalEnv != "" {
			os.Setenv("DMP_BOX64_PATH", originalEnv)
		}
	}()

	config := Config{
		HostOS:   "linux",
		HostArch: "arm64",
		// No Box64Path set, and /usr/bin/box64 doesn't exist in test environment
	}

	binary := "./dontstarve_x64"
	args := []string{"-console"}

	_, err := BuildCompatCommand(binary, args, X8664, config)
	if err == nil {
		t.Fatal("expected error when Box64 is missing, got nil")
	}

	// Error should mention Box64 requirement
	errMsg := err.Error()
	if !contains(errMsg, "Box64") {
		t.Errorf("error should mention Box64, got: %s", errMsg)
	}
	if !contains(errMsg, "ARM64") {
		t.Errorf("error should mention ARM64 host, got: %s", errMsg)
	}
}

func TestBuildCompatCommand_ARM64_X8632_MissingBox86(t *testing.T) {
	originalEnv := os.Getenv("DMP_BOX86_PATH")
	os.Unsetenv("DMP_BOX86_PATH")
	defer func() {
		if originalEnv != "" {
			os.Setenv("DMP_BOX86_PATH", originalEnv)
		}
	}()

	config := Config{
		HostOS:   "linux",
		HostArch: "arm64",
	}

	binary := "./dontstarve_32"
	args := []string{"-console"}

	_, err := BuildCompatCommand(binary, args, X8632, config)
	if err == nil {
		t.Fatal("expected error when Box86 is missing, got nil")
	}

	errMsg := err.Error()
	if !contains(errMsg, "Box86") {
		t.Errorf("error should mention Box86, got: %s", errMsg)
	}
	if !contains(errMsg, "ARM64") {
		t.Errorf("error should mention ARM64 host, got: %s", errMsg)
	}
}

func TestBuildCompatCommand_NonLinux(t *testing.T) {
	config := Config{
		HostOS:   "windows",
		HostArch: "amd64",
	}

	_, err := BuildCompatCommand("./app", []string{}, X8664, config)
	if err == nil {
		t.Fatal("expected error for non-Linux cross-arch execution, got nil")
	}
}

func TestBuildCompatCommand_UnsupportedHostArch(t *testing.T) {
	config := Config{
		HostOS:   "linux",
		HostArch: "riscv64",
	}

	_, err := BuildCompatCommand("./app", []string{}, X8664, config)
	if err == nil {
		t.Fatal("expected error for unsupported host architecture, got nil")
	}
}

func TestResolveBox64Path_EnvVar(t *testing.T) {
	fakeBox64 := createFakeExecutable(t, "box64")

	originalEnv := os.Getenv("DMP_BOX64_PATH")
	os.Setenv("DMP_BOX64_PATH", fakeBox64)
	defer func() {
		if originalEnv != "" {
			os.Setenv("DMP_BOX64_PATH", originalEnv)
		} else {
			os.Unsetenv("DMP_BOX64_PATH")
		}
	}()

	config := Config{}
	path, err := resolveBox64Path(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != fakeBox64 {
		t.Errorf("expected path %s, got %s", fakeBox64, path)
	}
}

func TestResolveBox64Path_InvalidEnvVar(t *testing.T) {
	originalEnv := os.Getenv("DMP_BOX64_PATH")
	os.Setenv("DMP_BOX64_PATH", "/nonexistent/box64")
	defer func() {
		if originalEnv != "" {
			os.Setenv("DMP_BOX64_PATH", originalEnv)
		} else {
			os.Unsetenv("DMP_BOX64_PATH")
		}
	}()

	config := Config{}
	_, err := resolveBox64Path(config)
	if err == nil {
		t.Fatal("expected error for invalid DMP_BOX64_PATH, got nil")
	}

	errMsg := err.Error()
	if !contains(errMsg, "DMP_BOX64_PATH") {
		t.Errorf("error should mention DMP_BOX64_PATH, got: %s", errMsg)
	}
}

func TestResolveBox86Path_Config(t *testing.T) {
	fakeBox86 := createFakeExecutable(t, "box86")

	config := Config{
		Box86Path: fakeBox86,
	}

	path, err := resolveBox86Path(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != fakeBox86 {
		t.Errorf("expected path %s, got %s", fakeBox86, path)
	}
}

func TestValidateExecutable_Directory(t *testing.T) {
	tempDir := t.TempDir()

	err := validateExecutable(tempDir, "test")
	if err == nil {
		t.Fatal("expected error for directory, got nil")
	}

	if !contains(err.Error(), "directory") {
		t.Errorf("error should mention directory, got: %s", err.Error())
	}
}

func TestValidateExecutable_NonExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping non-executable test on Windows")
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nonexec")

	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	f.Close()

	// Don't set executable bit
	if err := os.Chmod(filePath, 0644); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}

	err = validateExecutable(filePath, "test")
	if err == nil {
		t.Fatal("expected error for non-executable file, got nil")
	}

	if !contains(err.Error(), "not executable") {
		t.Errorf("error should mention not executable, got: %s", err.Error())
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.HostOS != runtime.GOOS {
		t.Errorf("expected HostOS %s, got %s", runtime.GOOS, config.HostOS)
	}

	if config.HostArch != runtime.GOARCH {
		t.Errorf("expected HostArch %s, got %s", runtime.GOARCH, config.HostArch)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
