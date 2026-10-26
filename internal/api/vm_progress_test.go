package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/feliren/ssh-web/internal/config"
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
