package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigBytesUsesSafeLocalDefaults(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte("port: 8317\n"))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Host, "127.0.0.1")
	}
	if !cfg.RemoteManagement.DisableControlPanel {
		t.Error("DisableControlPanel = false, want true")
	}
	if !cfg.RemoteManagement.DisableAutoUpdatePanel {
		t.Error("DisableAutoUpdatePanel = false, want true")
	}
}

func TestParseConfigBytesRespectsExplicitExposureSettings(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`host: "0.0.0.0"
remote-management:
  disable-control-panel: false
  disable-auto-update-panel: false
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", cfg.Host, "0.0.0.0")
	}
	if cfg.RemoteManagement.DisableControlPanel {
		t.Error("DisableControlPanel = true, want false")
	}
	if cfg.RemoteManagement.DisableAutoUpdatePanel {
		t.Error("DisableAutoUpdatePanel = true, want false")
	}
}

func TestLoadConfigUsesSafeLocalDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Host, "127.0.0.1")
	}
	if !cfg.RemoteManagement.DisableControlPanel {
		t.Error("DisableControlPanel = false, want true")
	}
	if !cfg.RemoteManagement.DisableAutoUpdatePanel {
		t.Error("DisableAutoUpdatePanel = false, want true")
	}
}

func TestLoadConfigRespectsExplicitExposureSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := []byte(`host: "0.0.0.0"
remote-management:
  disable-control-panel: false
  disable-auto-update-panel: false
`)
	if err := os.WriteFile(configPath, configYAML, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", cfg.Host, "0.0.0.0")
	}
	if cfg.RemoteManagement.DisableControlPanel {
		t.Error("DisableControlPanel = true, want false")
	}
	if cfg.RemoteManagement.DisableAutoUpdatePanel {
		t.Error("DisableAutoUpdatePanel = true, want false")
	}
}

func TestLoadConfigOptionalMissingUsesSafeLocalDefaults(t *testing.T) {
	cfg, err := LoadConfigOptional(filepath.Join(t.TempDir(), "missing.yaml"), true)
	if err != nil {
		t.Fatalf("LoadConfigOptional() error = %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Host, "127.0.0.1")
	}
	if !cfg.RemoteManagement.DisableControlPanel {
		t.Error("DisableControlPanel = false, want true")
	}
	if !cfg.RemoteManagement.DisableAutoUpdatePanel {
		t.Error("DisableAutoUpdatePanel = false, want true")
	}
}

func TestSaveConfigPreservesExplicitExposureOptIns(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Host = ""
	cfg.RemoteManagement.DisableControlPanel = false
	cfg.RemoteManagement.DisableAutoUpdatePanel = false
	if err = SaveConfigPreserveComments(configPath, cfg); err != nil {
		t.Fatalf("SaveConfigPreserveComments() error = %v", err)
	}

	reloaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() after save error = %v", err)
	}
	if reloaded.Host != "" {
		t.Errorf("Host after save = %q, want explicit empty value", reloaded.Host)
	}
	if reloaded.RemoteManagement.DisableControlPanel {
		t.Error("DisableControlPanel after save = true, want explicit false")
	}
	if reloaded.RemoteManagement.DisableAutoUpdatePanel {
		t.Error("DisableAutoUpdatePanel after save = true, want explicit false")
	}
}
