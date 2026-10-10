package dst

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/internal/runner"
	"dst-management-platform-api/utils"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// createFakeBox 创建一个可执行的空文件，用于 Box64/Box86 路径解析测试
func createFakeBox(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create fake executable: %v", err)
	}
	f.Close()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0755); err != nil {
			t.Fatalf("failed to chmod fake executable: %v", err)
		}
	}
	return path
}

// clearBoxEnv 清理 Box 相关环境变量，避免测试环境干扰路径解析
func clearBoxEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DMP_BOX64_PATH", "")
	t.Setenv("DMP_BOX86_PATH", "")
}

func newTestGame(startType string) *Game {
	return &Game{
		setting:      &models.RoomSetting{StartType: startType},
		roomSaveData: roomSaveData{clusterName: "Cluster_1"},
	}
}

func TestGenerateGameStartCmd_AMD64_64bit(t *testing.T) {
	clearBoxEnv(t)
	g := newTestGame("64-bit")
	config := runner.Config{HostOS: "linux", HostArch: "amd64"}

	cmd, err := g.generateGameStartCmd("DMP_Cluster_1_Master", "", "Master", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "'./dontstarve_dedicated_server_nullrenderer_x64'") {
		t.Errorf("expected x64 binary in command, got: %s", cmd)
	}
	if strings.Contains(cmd, "box64") {
		t.Errorf("AMD64 host should not use Box64: %s", cmd)
	}
	if !strings.Contains(cmd, "'-cluster' 'Cluster_1'") {
		t.Errorf("expected -cluster Cluster_1, got: %s", cmd)
	}
	if !strings.Contains(cmd, "'-shard' 'Master'") {
		t.Errorf("expected -shard Master, got: %s", cmd)
	}
	if !strings.Contains(cmd, "screen -d -h 200 -m -S 'DMP_Cluster_1_Master'") {
		t.Errorf("expected screen command, got: %s", cmd)
	}
}

func TestGenerateGameStartCmd_AMD64_32bit(t *testing.T) {
	clearBoxEnv(t)
	g := newTestGame("32-bit")
	config := runner.Config{HostOS: "linux", HostArch: "amd64"}

	cmd, err := g.generateGameStartCmd("DMP_Cluster_1_Caves", "", "Caves", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "'./dontstarve_dedicated_server_nullrenderer'") {
		t.Errorf("expected 32-bit binary in command, got: %s", cmd)
	}
	if strings.Contains(cmd, "box86") {
		t.Errorf("AMD64 host should not use Box86: %s", cmd)
	}
	if !strings.Contains(cmd, "'-shard' 'Caves'") {
		t.Errorf("expected -shard Caves, got: %s", cmd)
	}
	if !strings.Contains(cmd, "screen -d -h 200 -m -S 'DMP_Cluster_1_Caves'") {
		t.Errorf("expected screen command, got: %s", cmd)
	}
}

func TestGenerateGameStartCmd_AMD64_LuaJIT(t *testing.T) {
	clearBoxEnv(t)
	g := newTestGame("luajit")
	config := runner.Config{HostOS: "linux", HostArch: "amd64"}

	cmd, err := g.generateGameStartCmd("DMP_Cluster_1_Master", "", "Master", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "'./dontstarve_dedicated_server_nullrenderer_x64_luajit'") {
		t.Errorf("expected luajit binary in command, got: %s", cmd)
	}
	if strings.Contains(cmd, "box64") {
		t.Errorf("AMD64 host should not use Box64: %s", cmd)
	}
}

func TestGenerateGameStartCmd_ARM64_64bit(t *testing.T) {
	clearBoxEnv(t)
	fakeBox64 := createFakeBox(t, "box64")
	g := newTestGame("64-bit")
	config := runner.Config{HostOS: "linux", HostArch: "arm64", Box64Path: fakeBox64}

	cmd, err := g.generateGameStartCmd("DMP_Cluster_1_Master", "", "Master", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "'"+fakeBox64+"'") {
		t.Errorf("expected Box64 wrapper in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "'./dontstarve_dedicated_server_nullrenderer_x64'") {
		t.Errorf("expected x64 binary as Box64 argument, got: %s", cmd)
	}
	if !strings.Contains(cmd, "'-shard' 'Master'") {
		t.Errorf("expected -shard Master, got: %s", cmd)
	}
	if !strings.Contains(cmd, "screen -d -h 200 -m -S 'DMP_Cluster_1_Master'") {
		t.Errorf("expected screen command, got: %s", cmd)
	}
}

func TestGenerateGameStartCmd_ARM64_32bit(t *testing.T) {
	clearBoxEnv(t)
	fakeBox86 := createFakeBox(t, "box86")
	g := newTestGame("32-bit")
	config := runner.Config{HostOS: "linux", HostArch: "arm64", Box86Path: fakeBox86}

	cmd, err := g.generateGameStartCmd("DMP_Cluster_1_Caves", "", "Caves", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "'"+fakeBox86+"'") {
		t.Errorf("expected Box86 wrapper in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "'./dontstarve_dedicated_server_nullrenderer'") {
		t.Errorf("expected 32-bit binary as Box86 argument, got: %s", cmd)
	}
	if strings.Contains(cmd, "box64") {
		t.Errorf("32-bit world should use Box86, not Box64: %s", cmd)
	}
	if !strings.Contains(cmd, "'-shard' 'Caves'") {
		t.Errorf("expected -shard Caves, got: %s", cmd)
	}
}

func TestGenerateGameStartCmd_ARM64_LuaJIT(t *testing.T) {
	clearBoxEnv(t)
	fakeBox64 := createFakeBox(t, "box64")
	g := newTestGame("luajit")
	config := runner.Config{HostOS: "linux", HostArch: "arm64", Box64Path: fakeBox64}

	cmd, err := g.generateGameStartCmd("DMP_Cluster_1_Master", "", "Master", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "'"+fakeBox64+"'") {
		t.Errorf("expected Box64 wrapper in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "'./dontstarve_dedicated_server_nullrenderer_x64_luajit'") {
		t.Errorf("expected luajit binary as Box64 argument, got: %s", cmd)
	}
}

func TestGenerateGameStartCmd_ARM64_64bit_MissingBox64(t *testing.T) {
	clearBoxEnv(t)
	g := newTestGame("64-bit")
	config := runner.Config{HostOS: "linux", HostArch: "arm64"}

	_, err := g.generateGameStartCmd("DMP_Cluster_1_Master", "", "Master", config)
	if err == nil {
		t.Fatal("expected error when Box64 is missing, got nil")
	}

	if !strings.Contains(err.Error(), "Box64") {
		t.Errorf("error should mention Box64, got: %s", err.Error())
	}
}

func TestGenerateGameStartCmd_UnsupportedStartType(t *testing.T) {
	clearBoxEnv(t)
	g := newTestGame("bogus")
	config := runner.Config{HostOS: "linux", HostArch: "amd64"}

	_, err := g.generateGameStartCmd("DMP_Cluster_1_Master", "", "Master", config)
	if err == nil {
		t.Fatal("expected error for unsupported StartType, got nil")
	}

	if !strings.Contains(err.Error(), "StartType") {
		t.Errorf("error should mention StartType, got: %s", err.Error())
	}
}

func TestGenerateGameStartCmd_Darwin_Native(t *testing.T) {
	g := newTestGame("64-bit")
	config := runner.Config{HostOS: utils.Darwin, HostArch: "arm64"}

	cmd, err := g.generateGameStartCmd("DMP_Cluster_1_Master", "", "Master", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "'./dontstarve_dedicated_server_nullrenderer'") {
		t.Errorf("expected native darwin binary in command, got: %s", cmd)
	}
	if strings.Contains(cmd, "box64") {
		t.Errorf("Darwin should run natively without Box64: %s", cmd)
	}
	if !strings.Contains(cmd, "DYLD_LIBRARY_PATH") {
		t.Errorf("expected DYLD_LIBRARY_PATH prefix, got: %s", cmd)
	}
	if !strings.Contains(cmd, utils.DarwinDstBinDir) {
		t.Errorf("expected darwin bin dir, got: %s", cmd)
	}
}
