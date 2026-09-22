package cliproxy

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func TestEnsureAuthDirCreatesPrivateDirectoryWithoutChangingParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not available on Windows")
	}
	parent := t.TempDir()
	if errChmod := os.Chmod(parent, 0o755); errChmod != nil {
		t.Fatalf("chmod parent directory: %v", errChmod)
	}
	authDir := filepath.Join(parent, "auths")
	service := &Service{cfg: &config.Config{AuthDir: authDir}}

	if errEnsure := service.ensureAuthDir(); errEnsure != nil {
		t.Fatalf("ensureAuthDir() error = %v", errEnsure)
	}
	authInfo, errStat := os.Stat(authDir)
	if errStat != nil {
		t.Fatalf("stat auth directory: %v", errStat)
	}
	if got := authInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("auth directory permissions = %04o, want 0700", got)
	}
	parentInfo, errStat := os.Stat(parent)
	if errStat != nil {
		t.Fatalf("stat parent directory: %v", errStat)
	}
	if got := parentInfo.Mode().Perm(); got != 0o755 {
		t.Errorf("existing parent permissions = %04o, want unchanged 0755", got)
	}
}
