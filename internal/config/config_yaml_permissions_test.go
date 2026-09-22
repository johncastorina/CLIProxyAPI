package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveConfigPreserveCommentsTightensExistingFilePermissions(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if errWrite := os.WriteFile(configPath, []byte("port: 8317\n"), 0o644); errWrite != nil {
		t.Fatalf("write config: %v", errWrite)
	}
	if errChmod := os.Chmod(configPath, 0o644); errChmod != nil {
		t.Fatalf("chmod config: %v", errChmod)
	}

	cfg, errLoad := LoadConfig(configPath)
	if errLoad != nil {
		t.Fatalf("LoadConfig() error = %v", errLoad)
	}
	if errSave := SaveConfigPreserveComments(configPath, cfg); errSave != nil {
		t.Fatalf("SaveConfigPreserveComments() error = %v", errSave)
	}

	info, errStat := os.Stat(configPath)
	if errStat != nil {
		t.Fatalf("stat config: %v", errStat)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config permissions = %04o, want 0600", got)
	}
}

func TestSaveConfigPreserveCommentsRejectsSymlinkDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available on Windows test hosts")
	}
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.yaml")
	linkPath := filepath.Join(dir, "config.yaml")
	original := []byte("port: 8317\n")
	if errWrite := os.WriteFile(targetPath, original, 0o600); errWrite != nil {
		t.Fatalf("write target: %v", errWrite)
	}
	if errLink := os.Symlink(targetPath, linkPath); errLink != nil {
		t.Fatalf("create symlink: %v", errLink)
	}
	cfg, errLoad := LoadConfig(targetPath)
	if errLoad != nil {
		t.Fatalf("LoadConfig() error = %v", errLoad)
	}

	if errSave := SaveConfigPreserveComments(linkPath, cfg); errSave == nil {
		t.Fatal("SaveConfigPreserveComments() accepted a symlink destination")
	}
	got, errRead := os.ReadFile(targetPath)
	if errRead != nil {
		t.Fatalf("read target: %v", errRead)
	}
	if string(got) != string(original) {
		t.Fatalf("symlink target changed to %q, want %q", got, original)
	}
}
