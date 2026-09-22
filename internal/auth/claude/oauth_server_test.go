package claude

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestOAuthCallbackOnlyAcceptsLoopbackConnections(t *testing.T) {
	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := reservation.Addr().(*net.TCPAddr).Port
	if err = reservation.Close(); err != nil {
		t.Fatal(err)
	}
	server := NewOAuthServer(port)
	if err = server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if errStop := server.Stop(context.Background()); errStop != nil {
			t.Error(errStop)
		}
	})

	// Exercise the localhost redirect used by the actual OAuth authorization URL.
	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://localhost:%d/callback?code=test-code&state=test-state", port))
	if err != nil {
		t.Fatal(err)
	}
	if err = response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("callback returned HTTP %d", response.StatusCode)
	}
	result, err := server.WaitForCallback(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != "test-code" || result.State != "test-state" || result.Error != "" {
		t.Fatalf("unexpected callback result: %+v", result)
	}

	t.Run("rejects non-loopback interface", func(t *testing.T) {
		addresses, errAddresses := net.InterfaceAddrs()
		if errAddresses != nil {
			t.Fatal(errAddresses)
		}
		checked := false
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
				continue
			}
			checked = true
			connection, errDial := net.DialTimeout("tcp4", net.JoinHostPort(ipNet.IP.String(), strconv.Itoa(port)), time.Second)
			if errDial == nil {
				_ = connection.Close()
				t.Errorf("OAuth callback accepted a connection on non-loopback address %s", ipNet.IP)
			}
		}
		if !checked {
			t.Skip("host has no non-loopback IPv4 interface")
		}
	})
}
