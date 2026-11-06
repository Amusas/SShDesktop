package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/ssh"
	"github.com/feliren/ssh-web/internal/vm"
)

func TestVMCreateSSEValidation(t *testing.T) {
	defer func() {
		_ = os.Remove("hosts.json")
		_ = os.Remove("vms.json")
	}()
	cfg := &config.Config{
		SSHHost: "127.0.0.1",
		SSHPort: 22,
		SSHUser: "testuser",
		SSHRoot: t.TempDir(),
		VMRoot:  t.TempDir(),
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(cfg, log, t.TempDir())

	// 1. Invalid JSON body with Accept: text/event-stream
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/vms", bytes.NewReader([]byte("invalid json")))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.Header.Set("Accept", "text/event-stream")
	rrInvalid := httptest.NewRecorder()

	h.ServeHTTP(rrInvalid, reqInvalid)
	if rrInvalid.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba status 400 en body invalido, obtenido: %d", rrInvalid.Code)
	}

	// 2. Missing template with Accept: text/event-stream (should stream an error event)
	body, _ := json.Marshal(vm.VMCreateTemplateRequest{
		Name:       "test-vm-invalid-tpl",
		TemplateID: "non-existent-template-id",
	})
	reqStream := httptest.NewRequest(http.MethodPost, "/api/vms", bytes.NewReader(body))
	reqStream.Header.Set("Content-Type", "application/json")
	reqStream.Header.Set("Accept", "text/event-stream")
	rrStream := httptest.NewRecorder()

	h.ServeHTTP(rrStream, reqStream)

	if !strings.Contains(rrStream.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("se esperaba Content-Type text/event-stream, obtenido: %s", rrStream.Header().Get("Content-Type"))
	}

	output := rrStream.Body.String()
	if !strings.Contains(output, "data:") {
		t.Fatalf("se esperaba formato SSE con 'data:', obtenido: %s", output)
	}

	if !strings.Contains(output, `"stage":"error"`) && !strings.Contains(output, `"stage":"validating"`) {
		t.Fatalf("se esperaba evento SSE de progreso o error, obtenido: %s", output)
	}
}

func TestVMNetworkInfoAndConnectivityEndpoints(t *testing.T) {
	defer func() {
		_ = os.Remove("hosts.json")
		_ = os.Remove("vms.json")
		_ = os.Remove("keys.json")
	}()

	cfg := &config.Config{
		SSHHost:        "127.0.0.1",
		SSHPort:        22,
		SSHUser:        "testuser",
		SSHRoot:        t.TempDir(),
		VMRoot:         t.TempDir(),
		NATNetworkName: "vessel-net",
		NATNetworkCIDR: "10.250.0.0/24",
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(cfg, log, t.TempDir())

	store := vm.NewStore("vms.json")
	_, _ = store.Add(vm.VMRecord{
		Name:     "vm-api-1",
		OS:       "Debian 13",
		State:    "running",
		Host:     "127.0.0.1",
		HostPort: 22001,
		SSHUser:  "vm_api_1",
	})
	_, _ = store.Add(vm.VMRecord{
		Name:        "vm-api-2",
		OS:          "Debian 13",
		State:       "running",
		Host:        "127.0.0.1",
		HostPort:    22002,
		SSHUser:     "vm_api_2",
		PrivateIP:   "10.250.0.12",
		NetworkName: "vessel-net",
		NetworkCIDR: "10.250.0.0/24",
	})

	restore := ssh.SetVBoxCommandRunner(func(ctx context.Context, sshCfg ssh.Config, command string, l *slog.Logger) (string, error) {
		switch {
		case strings.Contains(command, `showvminfo "vm-api-1"`):
			return "name=\"vm-api-1\"\nVMState=\"running\"\nnic1=\"natnetwork\"\nnatnet1=\"vessel-net\"\nmacaddress1=\"080027111111\"\nnic2=\"nat\"\nmacaddress2=\"080027222222\"\n", nil
		case strings.Contains(command, `showvminfo "vm-api-2"`):
			return "name=\"vm-api-2\"\nVMState=\"running\"\nnic1=\"natnetwork\"\nnatnet1=\"vessel-net\"\nmacaddress1=\"080027333333\"\nnic2=\"nat\"\nmacaddress2=\"080027444444\"\n", nil
		case strings.Contains(command, "ip -j addr"):
			return `[{"ifindex":2,"ifname":"enp0s3","flags":["UP"],"operstate":"UP","address":"08:00:27:11:11:11","addr_info":[{"family":"inet","local":"10.250.0.11","prefixlen":24}]}]`, nil
		case strings.Contains(command, "ping -c 3 -W 2 10.250.0.12"):
			return "3 packets transmitted, 3 received, 0% packet loss\nrtt min/avg/max/mdev = 0.2/0.5/0.8/0.1 ms\n", nil
		case strings.Contains(command, "VESSEL_TCP_OK"):
			return "VESSEL_TCP_OK\n", nil
		default:
			return "", nil
		}
	})
	defer restore()

	// 1. GET /api/vms/vm-api-1/network-info
	reqNet := httptest.NewRequest(http.MethodGet, "/api/vms/vm-api-1/network-info", nil)
	rrNet := httptest.NewRecorder()
	h.ServeHTTP(rrNet, reqNet)
	if rrNet.Code != http.StatusOK {
		t.Fatalf("GET /api/vms/vm-api-1/network-info status=%d, body=%s", rrNet.Code, rrNet.Body.String())
	}
	var netInfo vm.VMNetworkInfoResponse
	if err := json.Unmarshal(rrNet.Body.Bytes(), &netInfo); err != nil {
		t.Fatalf("error decodificando respuesta network-info: %v", err)
	}
	if netInfo.PrivateIP != "10.250.0.11" || netInfo.NetworkName != "vessel-net" {
		t.Fatalf("network-info inesperado: %+v", netInfo)
	}

	// 2. POST /api/vms/vm-api-1/connectivity -> hacia vm-api-2
	connBody, _ := json.Marshal(map[string]interface{}{"target": "vm-api-2"})
	reqConn := httptest.NewRequest(http.MethodPost, "/api/vms/vm-api-1/connectivity", bytes.NewReader(connBody))
	reqConn.Header.Set("Content-Type", "application/json")
	rrConn := httptest.NewRecorder()
	h.ServeHTTP(rrConn, reqConn)
	if rrConn.Code != http.StatusOK {
		t.Fatalf("POST /api/vms/vm-api-1/connectivity status=%d, body=%s", rrConn.Code, rrConn.Body.String())
	}
	var connRes ssh.VMConnectivityResult
	if err := json.Unmarshal(rrConn.Body.Bytes(), &connRes); err != nil {
		t.Fatalf("error decodificando connectivity: %v", err)
	}
	if !connRes.Reachable || connRes.TargetIP != "10.250.0.12" {
		t.Fatalf("resultado de conectividad inesperado: %+v", connRes)
	}
}
