package logging

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFileRequestLoggerCreatesPrivateSuccessAndErrorLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not available on Windows")
	}
	logsDir := filepath.Join(t.TempDir(), "logs")
	logger := NewFileRequestLogger(true, logsDir, "", 10)

	for _, tc := range []struct {
		name   string
		status int
	}{
		{name: "success", status: http.StatusOK},
		{name: "error", status: http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errLog := logger.LogRequest(
				"/v1/responses",
				http.MethodPost,
				nil,
				[]byte(`{"input":"secret"}`),
				tc.status,
				nil,
				[]byte(`{"result":"detail"}`),
				nil,
				nil,
				nil,
				nil,
				nil,
				"req-"+tc.name,
				time.Now(),
				time.Now(),
			)
			if errLog != nil {
				t.Fatalf("LogRequest() error = %v", errLog)
			}
		})
	}

	dirInfo, errStat := os.Stat(logsDir)
	if errStat != nil {
		t.Fatalf("stat logs directory: %v", errStat)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("logs directory permissions = %04o, want 0700", got)
	}
	entries, errRead := os.ReadDir(logsDir)
	if errRead != nil {
		t.Fatalf("read logs directory: %v", errRead)
	}
	if len(entries) != 2 {
		t.Fatalf("log entry count = %d, want 2", len(entries))
	}
	for _, entry := range entries {
		info, errInfo := entry.Info()
		if errInfo != nil {
			t.Fatalf("stat %s: %v", entry.Name(), errInfo)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("%s permissions = %04o, want 0600", entry.Name(), got)
		}
	}
}
