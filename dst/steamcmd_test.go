package dst

import (
	"reflect"
	"testing"
)

func TestGameUpdateSteamCMDArgsForceInstallDirBeforeLogin(t *testing.T) {
	const dstDir = "/root/dst"
	args := gameUpdateSteamCMDArgs(dstDir)

	expected := []string{
		"+force_install_dir",
		dstDir,
		"+login",
		"anonymous",
		"+app_update",
		"343050",
		"validate",
		"+quit",
	}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected SteamCMD args:\n got: %#v\nwant: %#v", args, expected)
	}

	forceInstallDirIndex := -1
	loginIndex := -1
	for index, arg := range args {
		switch arg {
		case "+force_install_dir":
			forceInstallDirIndex = index
		case "+login":
			loginIndex = index
		}
	}
	if forceInstallDirIndex < 0 || loginIndex < 0 || forceInstallDirIndex >= loginIndex {
		t.Fatalf("+force_install_dir must precede +login, got args: %#v", args)
	}
}
