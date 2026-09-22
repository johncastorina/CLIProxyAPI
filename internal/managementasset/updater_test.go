package managementasset

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchLatestAssetRejectsInvalidDigest(t *testing.T) {
	for _, digestField := range []string{
		``,
		`,"digest":"sha256:not-hex"`,
		`,"digest":"sha256:abc123"`,
		`,"digest":"md5:` + strings.Repeat("a", 64) + `"`,
	} {
		digestField := digestField
		t.Run(digestField, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"assets":[{"name":"management.html","browser_download_url":"https://example.com/management.html"` + digestField + `}]}`))
			}))
			defer server.Close()

			_, _, err := fetchLatestAsset(t.Context(), server.Client(), server.URL)
			if err == nil || !strings.Contains(err.Error(), "digest") {
				t.Fatalf("fetchLatestAsset() error = %v, want digest validation error", err)
			}
		})
	}
}

func TestEnsureLatestManagementHTMLDoesNotInstallFallbackAfterReleaseFailure(t *testing.T) {
	staticDir := t.TempDir()
	localPath := filepath.Join(staticDir, managementAssetName)

	var requests atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		if req.URL.Host == "cpamc.router-for.me" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("unverified-fallback")),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("release unavailable")),
			Request:    req,
		}, nil
	})}
	oldFactory := httpClientFactory
	httpClientFactory = func(string) *http.Client { return client }
	t.Cleanup(func() { httpClientFactory = oldFactory })
	lastUpdateCheckMu.Lock()
	lastUpdateCheckTime = time.Time{}
	lastUpdateCheckMu.Unlock()

	if EnsureLatestManagementHTML(t.Context(), staticDir, "", "") {
		t.Fatal("unverified fallback management panel should not be installed")
	}
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Fatalf("management panel stat error = %v, want file absent", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("HTTP requests = %d, want only the primary release request", got)
	}
}

func TestEnsureLatestManagementHTMLInstallsVerifiedPanel(t *testing.T) {
	staticDir := t.TempDir()
	localPath := filepath.Join(staticDir, managementAssetName)
	panel := []byte("verified-panel")
	sum := sha256.Sum256(panel)
	digest := hex.EncodeToString(sum[:])

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := panel
		if req.URL.Host == "api.github.com" {
			body = []byte(`{"assets":[{"name":"management.html","browser_download_url":"https://downloads.example/management.html","digest":"sha256:` + digest + `"}]}`)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Request:    req,
		}, nil
	})}
	oldFactory := httpClientFactory
	httpClientFactory = func(string) *http.Client { return client }
	t.Cleanup(func() { httpClientFactory = oldFactory })
	lastUpdateCheckMu.Lock()
	lastUpdateCheckTime = time.Time{}
	lastUpdateCheckMu.Unlock()

	if !EnsureLatestManagementHTML(t.Context(), staticDir, "", "") {
		t.Fatal("verified management panel should be installed")
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != string(panel) {
		t.Fatalf("management panel = %q, want %q", got, panel)
	}
}

func TestEnsureLatestManagementHTMLKeepsExistingPanelOnDigestMismatch(t *testing.T) {
	staticDir := t.TempDir()
	localPath := filepath.Join(staticDir, managementAssetName)
	if err := os.WriteFile(localPath, []byte("existing-panel"), 0o644); err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := "different-panel"
		if req.URL.Host == "api.github.com" {
			body = `{"assets":[{"name":"management.html","browser_download_url":"https://downloads.example/management.html","digest":"sha256:` + digest + `"}]}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
	oldFactory := httpClientFactory
	httpClientFactory = func(string) *http.Client { return client }
	t.Cleanup(func() { httpClientFactory = oldFactory })
	lastUpdateCheckMu.Lock()
	lastUpdateCheckTime = time.Time{}
	lastUpdateCheckMu.Unlock()

	if !EnsureLatestManagementHTML(t.Context(), staticDir, "", "") {
		t.Fatal("existing management panel should remain available")
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "existing-panel" {
		t.Fatalf("management panel = %q, want existing panel unchanged", got)
	}
}

func TestFetchLatestAssetSetsGitHubAuthorization(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "asset-token")
	t.Setenv("GITSTORE_GIT_TOKEN", "")
	t.Setenv("GITSTORE_GIT_URL", "")

	var authorization string
	validDigest := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authorization = req.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets":[{"name":"management.html","browser_download_url":"https://example.com/management.html","digest":"sha256:` + validDigest + `"}]}`))
	}))
	defer server.Close()

	asset, remoteHash, err := fetchLatestAsset(t.Context(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("fetchLatestAsset() error = %v", err)
	}
	if authorization != "Bearer asset-token" {
		t.Fatalf("Authorization = %q, want %q", authorization, "Bearer asset-token")
	}
	if asset == nil || asset.Name != managementAssetName {
		t.Fatalf("asset = %#v, want %q", asset, managementAssetName)
	}
	if remoteHash != validDigest {
		t.Fatalf("remoteHash = %q, want %q", remoteHash, validDigest)
	}
}

func TestFetchLatestAssetOmitsAuthorizationWithoutToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("github_token", "")
	t.Setenv("GITSTORE_GIT_TOKEN", "")
	t.Setenv("GITSTORE_GIT_URL", "")

	var authorization string
	validDigest := strings.Repeat("b", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authorization = req.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets":[{"name":"management.html","browser_download_url":"https://example.com/management.html","digest":"sha256:` + validDigest + `"}]}`))
	}))
	defer server.Close()

	asset, remoteHash, err := fetchLatestAsset(t.Context(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("fetchLatestAsset() error = %v", err)
	}
	if authorization != "" {
		t.Fatalf("Authorization = %q, want empty", authorization)
	}
	if asset == nil || asset.Name != managementAssetName {
		t.Fatalf("asset = %#v, want %q", asset, managementAssetName)
	}
	if remoteHash != validDigest {
		t.Fatalf("remoteHash = %q, want %q", remoteHash, validDigest)
	}
}

func TestAutoUpdateSkipReason(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		wantReason string
		wantSkip   bool
	}{
		{
			name:       "nil config",
			cfg:        nil,
			wantReason: "config not yet available",
			wantSkip:   true,
		},
		{
			name: "cluster mode",
			cfg: &config.Config{
				Home: config.HomeConfig{Enabled: true},
			},
			wantReason: "cluster mode enabled",
			wantSkip:   true,
		},
		{
			name: "control panel disabled",
			cfg: &config.Config{
				RemoteManagement: config.RemoteManagement{DisableControlPanel: true},
			},
			wantReason: "control panel disabled",
			wantSkip:   true,
		},
		{
			name: "auto update disabled",
			cfg: &config.Config{
				RemoteManagement: config.RemoteManagement{DisableAutoUpdatePanel: true},
			},
			wantReason: "disable-auto-update-panel is enabled",
			wantSkip:   true,
		},
		{
			name:       "enabled",
			cfg:        &config.Config{},
			wantReason: "",
			wantSkip:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReason, gotSkip := autoUpdateSkipReason(tt.cfg)
			if gotReason != tt.wantReason || gotSkip != tt.wantSkip {
				t.Fatalf("autoUpdateSkipReason() = (%q, %t), want (%q, %t)", gotReason, gotSkip, tt.wantReason, tt.wantSkip)
			}
		})
	}
}
