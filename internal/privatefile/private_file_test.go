package privatefile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenCreatesPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.json")
	file, errOpen := Open(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if errOpen != nil {
		t.Fatalf("Open() error = %v", errOpen)
	}
	if _, errWrite := file.WriteString("secret"); errWrite != nil {
		_ = file.Close()
		t.Fatalf("write private file: %v", errWrite)
	}
	if errClose := file.Close(); errClose != nil {
		t.Fatalf("close private file: %v", errClose)
	}
	if runtime.GOOS == "windows" {
		return
	}
	info, errStat := os.Stat(path)
	if errStat != nil {
		t.Fatalf("stat private file: %v", errStat)
	}
	if got := info.Mode().Perm(); got != PrivateFileMode {
		t.Fatalf("private file permissions = %04o, want %04o", got, PrivateFileMode)
	}
}

func TestOpenRejectsFinalSymlinkWithoutModifyingTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available on Windows test hosts")
	}
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target")
	linkPath := filepath.Join(dir, "secret")
	if errWrite := os.WriteFile(targetPath, []byte("original"), 0o600); errWrite != nil {
		t.Fatalf("write target: %v", errWrite)
	}
	if errLink := os.Symlink(targetPath, linkPath); errLink != nil {
		t.Fatalf("create symlink: %v", errLink)
	}

	if file, errOpen := Open(linkPath, os.O_WRONLY|os.O_TRUNC); errOpen == nil {
		_ = file.Close()
		t.Fatal("Open() accepted a symlink destination")
	}
	got, errRead := os.ReadFile(targetPath)
	if errRead != nil {
		t.Fatalf("read target: %v", errRead)
	}
	if string(got) != "original" {
		t.Fatalf("symlink target changed to %q", got)
	}
}
