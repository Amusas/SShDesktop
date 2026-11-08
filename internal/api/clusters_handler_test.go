package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/docker"
)

type testSwarmVMProvider struct {
	vms []docker.SwarmVMInfo
}

func (p *testSwarmVMProvider) ListVMs() ([]docker.SwarmVMInfo, error) {
	return p.vms, nil
}

func (p *testSwarmVMProvider) GetVM(nameOrID string) (docker.SwarmVMInfo, error) {
	for _, v := range p.vms {
		if v.ID == nameOrID || strings.EqualFold(v.Name, nameOrID) {
			return v, nil
		}
	}
	return docker.SwarmVMInfo{}, fmt.Errorf("no encontrada: %s", nameOrID)
}

func (p *testSwarmVMProvider) EnsureVMNetwork(_ context.Context, name string) (docker.SwarmVMInfo, error) {
	return p.GetVM(name)
}

func (p *testSwarmVMProvider) CheckVMConnectivity(_ context.Context, _, _ string, _ int) (bool, string, error) {
	return true, "ok", nil
}

func TestClustersAPIEndpointsAndSSE(t *testing.T) {
	cfg := &config.Config{SSHHost: "127.0.0.1", SSHPort: 22, SSHUser: "test"}
	h := NewHandler(cfg, slog.Default(), t.TempDir())

	store := docker.NewClusterStore(filepath.Join(t.TempDir(), "clusters.json"))
	provider := &testSwarmVMProvider{
		vms: []docker.SwarmVMInfo{
			{ID: "vm-1", Name: "node-mgr", State: "running", PrivateIP: "10.250.0.51", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
			{ID: "vm-2", Name: "node-wrk", State: "running", PrivateIP: "10.250.0.52", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		},
	}

	swarmActive := map[string]bool{}
	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		if strings.Contains(cmd, "docker info") {
			if swarmActive[vmName] {
				return fmt.Sprintf("26.1.0|active|cid-123|nid-%s|%s\n", vmName, vmName), nil
			}
			return fmt.Sprintf("26.1.0|inactive|||%s\n", vmName), nil
		}
		if strings.Contains(cmd, "PORTS_OK") {
			return "PORTS_OK\n", nil
		}
		if strings.Contains(cmd, "docker swarm init") {
			swarmActive[vmName] = true
			return "Swarm initialized\n", nil
		}
		if strings.Contains(cmd, "docker swarm join-token -q worker") {
			return "SWMTKN-1-secret\n", nil
		}
		if strings.Contains(cmd, "docker swarm join ") {
			swarmActive[vmName] = true
			return "Joined\n", nil
		}
		if strings.Contains(cmd, "docker node ls") {
			return strings.Join([]string{
				`{"ID":"nid-node-mgr","Hostname":"node-mgr","Status":"Ready","Availability":"Active","ManagerStatus":"Leader","EngineVersion":"26.1.0"}`,
				`{"ID":"nid-node-wrk","Hostname":"node-wrk","Status":"Ready","Availability":"Active","ManagerStatus":"","EngineVersion":"26.1.0"}`,
			}, "\n"), nil
		}
		return "", nil
	}

	h.swarmSvc = docker.NewSwarmService(runner, provider, store, slog.Default())

	// 1. GET /api/clusters/eligible-vms
	reqElig := httptest.NewRequest(http.MethodGet, "/api/clusters/eligible-vms", nil)
	recElig := httptest.NewRecorder()
	h.ServeHTTP(recElig, reqElig)
	if recElig.Code != http.StatusOK {
		t.Fatalf("status inesperado en eligible-vms: %d (%s)", recElig.Code, recElig.Body.String())
	}
	var eligList []docker.VMSwarmEligibility
	if err := json.Unmarshal(recElig.Body.Bytes(), &eligList); err != nil || len(eligList) != 2 {
		t.Fatalf("respuesta inesperada en eligible-vms: %v (%+v)", err, eligList)
	}

	// 2. POST /api/clusters con SSE (Accept: text/event-stream)
	bodyJSON, _ := json.Marshal(docker.CreateClusterRequest{
		Name:      "sse-cluster",
		ManagerVM: "node-mgr",
		WorkerVMs: []string{"node-wrk"},
	})
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(bodyJSON))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Accept", "text/event-stream")
	recCreate := httptest.NewRecorder()
	h.ServeHTTP(recCreate, reqCreate)

	if recCreate.Code != http.StatusOK {
		t.Fatalf("status inesperado en POST /api/clusters SSE: %d (%s)", recCreate.Code, recCreate.Body.String())
	}
	sseOut := recCreate.Body.String()
	if !strings.Contains(sseOut, `"stage":"done"`) {
		t.Fatalf("la respuesta SSE no incluyó el evento final 'done': %s", sseOut)
	}
	if strings.Contains(sseOut, "SWMTKN-1-secret") {
		t.Fatalf("la respuesta SSE expuso el token de Swarm: %s", sseOut)
	}

	// 3. GET /api/clusters
	reqList := httptest.NewRequest(http.MethodGet, "/api/clusters", nil)
	recList := httptest.NewRecorder()
	h.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("status inesperado en GET /api/clusters: %d", recList.Code)
	}
	var clusters []docker.SwarmCluster
	if err := json.Unmarshal(recList.Body.Bytes(), &clusters); err != nil || len(clusters) != 1 {
		t.Fatalf("se esperaba 1 cluster creado, obtenido: %+v (%v)", clusters, err)
	}
}
