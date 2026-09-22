package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveTokenToFile_PreservesCustomMetadata(t *testing.T) {
	tempDir := t.TempDir()
	authFilePath := filepath.Join(tempDir, "codex-test.json")

	storage := &CodexTokenStorage{
		Type:         "codex",
		Email:        "user@example.com",
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		IDToken:      "new-id-token",
		AccountID:    "new-account",
		Expire:       "2026-12-31T23:59:59Z",
		LastRefresh:  "2026-04-14T12:00:00Z",
	}
	storage.SetMetadata(map[string]any{
		"disabled":   false,
		"prefix":     "my-prefix",
		"websockets": false,
		"note":       "my important note",
		"proxy_url":  "http://proxy:8080",
		"weight":     float64(42),
	})

	if errSave := storage.SaveTokenToFile(authFilePath); errSave != nil {
		t.Fatalf("SaveTokenToFile() error = %v", errSave)
	}

	savedRaw, errRead := os.ReadFile(authFilePath)
	if errRead != nil {
		t.Fatalf("os.ReadFile error = %v", errRead)
	}

	var saved map[string]any
	if errUnmarshal := json.Unmarshal(savedRaw, &saved); errUnmarshal != nil {
		t.Fatalf("json.Unmarshal error = %v", errUnmarshal)
	}

	// Verify updated OAuth token fields
	if saved["access_token"] != "new-access-token" {
		t.Errorf("access_token = %v, want new-access-token", saved["access_token"])
	}
	if saved["refresh_token"] != "new-refresh-token" {
		t.Errorf("refresh_token = %v, want new-refresh-token", saved["refresh_token"])
	}
	if saved["id_token"] != "new-id-token" {
		t.Errorf("id_token = %v, want new-id-token", saved["id_token"])
	}
	if saved["account_id"] != "new-account" {
		t.Errorf("account_id = %v, want new-account", saved["account_id"])
	}

	// Verify custom fields in metadata
	if saved["prefix"] != "my-prefix" {
		t.Errorf("prefix = %v, want my-prefix", saved["prefix"])
	}
	if saved["websockets"] != false {
		t.Errorf("websockets = %v, want false", saved["websockets"])
	}
	if saved["note"] != "my important note" {
		t.Errorf("note = %v, want my important note", saved["note"])
	}
	if saved["proxy_url"] != "http://proxy:8080" {
		t.Errorf("proxy_url = %v, want http://proxy:8080", saved["proxy_url"])
	}
	if saved["weight"] != float64(42) {
		t.Errorf("weight = %v, want 42", saved["weight"])
	}
}

func TestSaveTokenToFileTightensExistingFilePermissions(t *testing.T) {
	authFilePath := filepath.Join(t.TempDir(), "codex-test.json")
	if errWrite := os.WriteFile(authFilePath, []byte(`{"access_token":"old"}`), 0o644); errWrite != nil {
		t.Fatalf("write existing token file: %v", errWrite)
	}
	if errChmod := os.Chmod(authFilePath, 0o644); errChmod != nil {
		t.Fatalf("chmod existing token file: %v", errChmod)
	}

	storage := &CodexTokenStorage{AccessToken: "new-access-token"}
	if errSave := storage.SaveTokenToFile(authFilePath); errSave != nil {
		t.Fatalf("SaveTokenToFile() error = %v", errSave)
	}

	info, errStat := os.Stat(authFilePath)
	if errStat != nil {
		t.Fatalf("stat token file: %v", errStat)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token file permissions = %04o, want 0600", got)
	}
}

func TestSaveTokenToFileCreatesPrivateFileAndDirectory(t *testing.T) {
	baseDir := t.TempDir()
	authDir := filepath.Join(baseDir, "auths")
	authFilePath := filepath.Join(authDir, "codex-test.json")

	storage := &CodexTokenStorage{AccessToken: "access-token"}
	if errSave := storage.SaveTokenToFile(authFilePath); errSave != nil {
		t.Fatalf("SaveTokenToFile() error = %v", errSave)
	}

	fileInfo, errStat := os.Stat(authFilePath)
	if errStat != nil {
		t.Fatalf("stat token file: %v", errStat)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Errorf("token file permissions = %04o, want 0600", got)
	}
	dirInfo, errStat := os.Stat(authDir)
	if errStat != nil {
		t.Fatalf("stat auth directory: %v", errStat)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("auth directory permissions = %04o, want 0700", got)
	}
}
