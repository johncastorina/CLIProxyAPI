package management

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteConfigRejectsSymlinkDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available on Windows test hosts")
	}
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.yaml")
	linkPath := filepath.Join(dir, "config.yaml")
	original := []byte("secret-key: original\n")
	if errWrite := os.WriteFile(targetPath, original, 0o600); errWrite != nil {
		t.Fatalf("write target: %v", errWrite)
	}
	if errLink := os.Symlink(targetPath, linkPath); errLink != nil {
		t.Fatalf("create symlink: %v", errLink)
	}

	if errWrite := WriteConfig(linkPath, []byte("secret-key: replacement\n")); errWrite == nil {
		t.Fatal("WriteConfig() accepted a symlink destination")
	}
	got, errRead := os.ReadFile(targetPath)
	if errRead != nil {
		t.Fatalf("read target: %v", errRead)
	}
	if string(got) != string(original) {
		t.Fatalf("symlink target changed to %q, want %q", got, original)
	}
}

func TestWriteConfigTightensExistingFilePermissions(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if errWrite := os.WriteFile(configPath, []byte("port: 8317\n"), 0o644); errWrite != nil {
		t.Fatalf("write config: %v", errWrite)
	}
	if errChmod := os.Chmod(configPath, 0o644); errChmod != nil {
		t.Fatalf("chmod config: %v", errChmod)
	}

	if errWrite := WriteConfig(configPath, []byte("port: 8318\n")); errWrite != nil {
		t.Fatalf("WriteConfig() error = %v", errWrite)
	}
	info, errStat := os.Stat(configPath)
	if errStat != nil {
		t.Fatalf("stat config: %v", errStat)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config permissions = %04o, want 0600", got)
	}
}
