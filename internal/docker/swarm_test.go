package docker

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

type mockSwarmVMProvider struct {
	vms          map[string]SwarmVMInfo
	connectivity map[string]bool
}

func newMockSwarmVMProvider(vms ...SwarmVMInfo) *mockSwarmVMProvider {
	m := &mockSwarmVMProvider{
		vms:          make(map[string]SwarmVMInfo),
		connectivity: make(map[string]bool),
	}
	for _, v := range vms {
		m.vms[v.Name] = v
	}
	return m
}

func (m *mockSwarmVMProvider) ListVMs() ([]SwarmVMInfo, error) {
	out := make([]SwarmVMInfo, 0, len(m.vms))
	for _, v := range m.vms {
		out = append(out, v)
	}
	return out, nil
}

func (m *mockSwarmVMProvider) GetVM(nameOrID string) (SwarmVMInfo, error) {
	for _, v := range m.vms {
		if v.ID == nameOrID || strings.EqualFold(v.Name, nameOrID) {
			return v, nil
		}
	}
	return SwarmVMInfo{}, fmt.Errorf("VM %q no encontrada", nameOrID)
}

func (m *mockSwarmVMProvider) EnsureVMNetwork(_ context.Context, name string) (SwarmVMInfo, error) {
	return m.GetVM(name)
}

func (m *mockSwarmVMProvider) CheckVMConnectivity(_ context.Context, sourceVM, targetIP string, port int) (bool, string, error) {
	key := fmt.Sprintf("%s->%s:%d", sourceVM, targetIP, port)
	if reachable, exists := m.connectivity[key]; exists {
		if !reachable {
			return false, "connection timed out", nil
		}
		return true, "ok", nil
	}
	return true, "ok", nil
}

func TestGetVMEligibility(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clusters.json")
	store := NewClusterStore(tmpFile)

	// Registrar un cluster previo con vm-in-cluster
	_ = store.Add(SwarmCluster{
		ID:            "cluster-existing",
		Name:          "prod-swarm",
		Status:        ClusterStatusActive,
		ManagerVMID:   "id-in-cluster",
		ManagerVMName: "vm-in-cluster",
		Nodes: []SwarmNode{
			{VMID: "id-in-cluster", VMName: "vm-in-cluster", Role: NodeRoleManager, SwarmStatus: NodeSwarmReady, VMState: "running"},
		},
	})

	provider := newMockSwarmVMProvider(
		SwarmVMInfo{ID: "id-ok", Name: "vm-ok", State: "running", PrivateIP: "10.250.0.10", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "id-in-cluster", Name: "vm-in-cluster", State: "running", PrivateIP: "10.250.0.11", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "id-off", Name: "vm-off", State: "poweroff", PrivateIP: "10.250.0.12", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "id-nat", Name: "vm-nat-ip", State: "running", PrivateIP: "10.0.3.15", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "id-ext", Name: "vm-ext-swarm", State: "running", PrivateIP: "10.250.0.14", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "id-nodocker", Name: "vm-no-docker", State: "running", PrivateIP: "10.250.0.15", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
	)

	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		switch vmName {
		case "vm-ok":
			return "26.1.0|inactive|||vm-ok\n", nil
		case "vm-ext-swarm":
			return "26.1.0|active|ext-cluster-id|ext-node-id|vm-ext-swarm\n", nil
		case "vm-no-docker":
			return "", errors.New("Cannot connect to the Docker daemon")
		default:
			return "26.1.0|inactive|||" + vmName + "\n", nil
		}
	}

	svc := NewSwarmService(runner, provider, store, nil)
	items, err := svc.GetVMEligibility(context.Background())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	byName := make(map[string]VMSwarmEligibility)
	for _, item := range items {
		byName[item.VMName] = item
	}

	if !byName["vm-ok"].Eligible {
		t.Errorf("se esperaba vm-ok elegible, obtenido: %+v", byName["vm-ok"])
	}
	if byName["vm-in-cluster"].Eligible || !strings.Contains(byName["vm-in-cluster"].Reason, "prod-swarm") {
		t.Errorf("se esperaba vm-in-cluster no elegible por pertenecer a prod-swarm, obtenido: %+v", byName["vm-in-cluster"])
	}
	if byName["vm-off"].Eligible || !strings.Contains(byName["vm-off"].Reason, "apagada") {
		t.Errorf("se esperaba vm-off no elegible por estar apagada, obtenido: %+v", byName["vm-off"])
	}
	if byName["vm-nat-ip"].Eligible || !strings.Contains(byName["vm-nat-ip"].Reason, "NAT") {
		t.Errorf("se esperaba vm-nat-ip rechazada por usar IP NAT 10.0.3.x, obtenido: %+v", byName["vm-nat-ip"])
	}
	if byName["vm-ext-swarm"].Eligible || !byName["vm-ext-swarm"].ExternalSwarm {
		t.Errorf("se esperaba vm-ext-swarm no elegible por Swarm externo, obtenido: %+v", byName["vm-ext-swarm"])
	}
	if byName["vm-no-docker"].Eligible {
		t.Errorf("se esperaba vm-no-docker no elegible por fallo de Docker, obtenido: %+v", byName["vm-no-docker"])
	}
}

func TestCreateClusterWithProgress_SuccessAndTokenPrivacy(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clusters.json")
	store := NewClusterStore(tmpFile)

	provider := newMockSwarmVMProvider(
		SwarmVMInfo{ID: "vm-1", Name: "mgr-1", State: "running", PrivateIP: "10.250.0.21", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "vm-2", Name: "wrk-1", State: "running", PrivateIP: "10.250.0.22", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "vm-3", Name: "wrk-2", State: "running", PrivateIP: "10.250.0.23", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
	)

	const secretToken = "SWMTKN-1-49nj1cmql0jkz5s954yi3oex3nedyz0fb0xx123456789"
	swarmActive := map[string]bool{}
	var executedCmds []string

	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		executedCmds = append(executedCmds, vmName+": "+cmd)

		if strings.Contains(cmd, "docker info") {
			if swarmActive[vmName] {
				return fmt.Sprintf("26.1.0|active|swarm-cluster-xyz|node-%s|%s\n", vmName, vmName), nil
			}
			return fmt.Sprintf("26.1.0|inactive|||%s\n", vmName), nil
		}
		if strings.Contains(cmd, "PORTS_OK") {
			return "PORTS_OK\n", nil
		}
		if strings.Contains(cmd, "docker swarm init") {
			if !strings.Contains(cmd, "--advertise-addr '10.250.0.21'") || !strings.Contains(cmd, "--data-path-addr '10.250.0.21'") {
				return "", fmt.Errorf("comando init sin IPs explícitas de vessel-net: %s", cmd)
			}
			swarmActive[vmName] = true
			return "Swarm initialized\n", nil
		}
		if strings.Contains(cmd, "docker swarm join-token -q worker") {
			return secretToken + "\n", nil
		}
		if strings.Contains(cmd, "docker swarm join ") {
			swarmActive[vmName] = true
			return "This node joined a swarm as a worker.\n", nil
		}
		if strings.Contains(cmd, "docker node ls") {
			return strings.Join([]string{
				`{"ID":"node-mgr-1","Hostname":"mgr-1","Status":"Ready","Availability":"Active","ManagerStatus":"Leader","EngineVersion":"26.1.0"}`,
				`{"ID":"node-wrk-1","Hostname":"wrk-1","Status":"Ready","Availability":"Active","ManagerStatus":"","EngineVersion":"26.1.0"}`,
				`{"ID":"node-wrk-2","Hostname":"wrk-2","Status":"Ready","Availability":"Active","ManagerStatus":"","EngineVersion":"26.1.0"}`,
			}, "\n"), nil
		}
		return "", nil
	}

	svc := NewSwarmService(runner, provider, store, nil)

	var events []ClusterProgressEvent
	cluster, err := svc.CreateClusterWithProgress(context.Background(), CreateClusterRequest{
		Name:      "cluster-alpha",
		ManagerVM: "mgr-1",
		WorkerVMs: []string{"wrk-1", "wrk-2"},
	}, func(ev ClusterProgressEvent) {
		if strings.Contains(ev.Message, secretToken) || strings.Contains(ev.Error, secretToken) {
			t.Errorf("el evento de progreso expuso el token secreto de Swarm: %+v", ev)
		}
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("error inesperado creando cluster: %v", err)
	}

	if cluster.Status != ClusterStatusActive {
		t.Errorf("se esperaba estado Active, obtenido %s (reason: %s)", cluster.Status, cluster.StatusReason)
	}
	if cluster.WorkersConnected != 2 || cluster.TotalNodes != 3 {
		t.Errorf("contadores inesperados: WorkersConnected=%d, TotalNodes=%d", cluster.WorkersConnected, cluster.TotalNodes)
	}
	if len(events) < 11 {
		t.Errorf("se esperaban al menos 11 eventos de progreso, recibidos %d", len(events))
	}
}

func TestCreateClusterWithProgress_PartialWorkerFailureAndRetry(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clusters.json")
	store := NewClusterStore(tmpFile)

	provider := newMockSwarmVMProvider(
		SwarmVMInfo{ID: "vm-1", Name: "mgr-1", State: "running", PrivateIP: "10.250.0.31", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "vm-2", Name: "wrk-ok", State: "running", PrivateIP: "10.250.0.32", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "vm-3", Name: "wrk-fail", State: "running", PrivateIP: "10.250.0.33", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
	)

	const secretToken = "SWMTKN-1-secret-token-value"
	swarmActive := map[string]bool{}
	failWrk2 := true

	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		if strings.Contains(cmd, "docker info") {
			if swarmActive[vmName] {
				return fmt.Sprintf("26.1.0|active|swarm-cid|node-%s|%s\n", vmName, vmName), nil
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
			return secretToken + "\n", nil
		}
		if strings.Contains(cmd, "docker swarm join ") {
			if vmName == "wrk-fail" && failWrk2 {
				return "Error with token " + secretToken, errors.New("join timeout")
			}
			swarmActive[vmName] = true
			return "Joined\n", nil
		}
		if strings.Contains(cmd, "docker node ls") {
			lines := []string{
				`{"ID":"node-mgr-1","Hostname":"mgr-1","Status":"Ready","Availability":"Active","ManagerStatus":"Leader","EngineVersion":"26.1.0"}`,
				`{"ID":"node-wrk-ok","Hostname":"wrk-ok","Status":"Ready","Availability":"Active","ManagerStatus":"","EngineVersion":"26.1.0"}`,
			}
			if !failWrk2 {
				lines = append(lines, `{"ID":"node-wrk-fail","Hostname":"wrk-fail","Status":"Ready","Availability":"Active","ManagerStatus":"","EngineVersion":"26.1.0"}`)
			}
			return strings.Join(lines, "\n"), nil
		}
		return "", nil
	}

	svc := NewSwarmService(runner, provider, store, nil)
	cluster, err := svc.CreateClusterWithProgress(context.Background(), CreateClusterRequest{
		Name:      "cluster-degraded",
		ManagerVM: "mgr-1",
		WorkerVMs: []string{"wrk-ok", "wrk-fail"},
	}, nil)
	if err != nil {
		t.Fatalf("no debía fallar destructivamente ante un fallo parcial de Worker: %v", err)
	}

	if cluster.Status != ClusterStatusDegraded {
		t.Errorf("se esperaba estado Degraded, obtenido %s", cluster.Status)
	}
	if cluster.WorkersConnected != 1 || cluster.WorkersTotal != 2 || cluster.TotalNodes != 3 {
		t.Errorf("contadores inesperados en fallo parcial: connected=%d, totalWorkers=%d, totalNodes=%d",
			cluster.WorkersConnected, cluster.WorkersTotal, cluster.TotalNodes)
	}
	for _, n := range cluster.Nodes {
		if strings.Contains(n.JoinError, secretToken) {
			t.Errorf("el error de unión del nodo filtró el token secreto: %s", n.JoinError)
		}
	}

	// Ahora reintentar incorporar wrk-fail y verificar que pasa a Active
	failWrk2 = false
	updated, retryErr := svc.RetryOrAddWorker(context.Background(), cluster.ID, "wrk-fail")
	if retryErr != nil {
		t.Fatalf("error al reintentar incorporación de Worker: %v", retryErr)
	}
	if updated.Status != ClusterStatusActive || updated.WorkersConnected != 2 {
		t.Errorf("tras reintentar Worker se esperaba Active con 2 Workers conectados, obtenido: status=%s, connected=%d",
			updated.Status, updated.WorkersConnected)
	}
}

func TestRemoveNodeFromCluster_WorkerAndManagerRules(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clusters.json")
	store := NewClusterStore(tmpFile)

	provider := newMockSwarmVMProvider(
		SwarmVMInfo{ID: "vm-1", Name: "mgr-1", State: "running", PrivateIP: "10.250.0.41", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "vm-2", Name: "wrk-1", State: "running", PrivateIP: "10.250.0.42", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
	)

	_ = store.Add(SwarmCluster{
		ID:            "cluster-rm",
		Name:          "rm-cluster",
		Status:        ClusterStatusActive,
		ManagerVMID:   "vm-1",
		ManagerVMName: "mgr-1",
		ManagerIP:     "10.250.0.41",
		Nodes: []SwarmNode{
			{VMID: "vm-1", VMName: "mgr-1", NodeID: "node-mgr-1", Role: NodeRoleManager, IsLeader: true, InternalIP: "10.250.0.41", VMState: "running", SwarmStatus: NodeSwarmReady},
			{VMID: "vm-2", VMName: "wrk-1", NodeID: "node-wrk-1", Role: NodeRoleWorker, InternalIP: "10.250.0.42", VMState: "running", SwarmStatus: NodeSwarmReady},
		},
	})

	workerRemoved := false
	managerLeft := false

	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		if strings.Contains(cmd, "docker info") {
			if vmName == "mgr-1" && managerLeft {
				return "26.1.0|inactive|||mgr-1\n", nil
			}
			return fmt.Sprintf("26.1.0|active|swarm-cid|node-%s|%s\n", vmName, vmName), nil
		}
		if strings.Contains(cmd, "docker node ls --format '{{json .}}'") {
			if workerRemoved {
				return `{"ID":"node-mgr-1","Hostname":"mgr-1","Status":"Ready","Availability":"Active","ManagerStatus":"Leader","EngineVersion":"26.1.0"}`, nil
			}
			return strings.Join([]string{
				`{"ID":"node-mgr-1","Hostname":"mgr-1","Status":"Ready","Availability":"Active","ManagerStatus":"Leader","EngineVersion":"26.1.0"}`,
				`{"ID":"node-wrk-1","Hostname":"wrk-1","Status":"Ready","Availability":"Active","ManagerStatus":"","EngineVersion":"26.1.0"}`,
			}, "\n"), nil
		}
		if strings.Contains(cmd, "docker node update --availability drain") {
			return "node-wrk-1\n", nil
		}
		if strings.Contains(cmd, "docker swarm leave") {
			if vmName == "mgr-1" {
				managerLeft = true
			}
			return "Node left the swarm.\n", nil
		}
		if strings.Contains(cmd, "docker node rm") {
			workerRemoved = true
			return "node-wrk-1\n", nil
		}
		if strings.Contains(cmd, "docker node ls --format '{{.ID}}'") {
			if workerRemoved {
				return "node-mgr-1\n", nil
			}
			return "node-mgr-1\nnode-wrk-1\n", nil
		}
		return "", nil
	}

	svc := NewSwarmService(runner, provider, store, nil)

	// 1. Intentar eliminar el Manager mientras existe wrk-1 debe fallar
	if _, err := svc.RemoveNodeFromCluster(context.Background(), "cluster-rm", "mgr-1", false); err == nil {
		t.Fatalf("debía bloquearse la eliminación del único Manager mientras existan Workers")
	}

	// 2. Si wrk-1 se apaga e intentamos eliminarlo sin force, debe devolver ErrWorkerUnreachable
	wrkVM := provider.vms["wrk-1"]
	wrkVM.State = "poweroff"
	provider.vms["wrk-1"] = wrkVM
	resOffline, errOffline := svc.RemoveNodeFromCluster(context.Background(), "cluster-rm", "wrk-1", false)
	if !errors.Is(errOffline, ErrWorkerUnreachable) || resOffline == nil || !resOffline.WorkerUnreachable {
		t.Fatalf("se esperaba ErrWorkerUnreachable para Worker apagado sin force, obtenido: res=%+v, err=%v", resOffline, errOffline)
	}

	// 3. Encender wrk-1 y retirarlo de forma limpia
	wrkVM.State = "running"
	provider.vms["wrk-1"] = wrkVM
	resWrk, errWrk := svc.RemoveNodeFromCluster(context.Background(), "cluster-rm", "wrk-1", false)
	if errWrk != nil {
		t.Fatalf("error retirando Worker wrk-1: %v", errWrk)
	}
	if resWrk.ClusterDeleted || resWrk.Cluster == nil || len(resWrk.Cluster.Nodes) != 1 {
		t.Fatalf("resultado inesperado tras retirar Worker: %+v", resWrk)
	}

	// 4. Ahora que no quedan Workers, retirar el Manager debe eliminar el cluster
	resMgr, errMgr := svc.RemoveNodeFromCluster(context.Background(), "cluster-rm", "mgr-1", false)
	if errMgr != nil {
		t.Fatalf("error retirando el último Manager: %v", errMgr)
	}
	if !resMgr.ClusterDeleted {
		t.Errorf("se esperaba ClusterDeleted=true al retirar el último Manager")
	}
	if _, stillExists := store.Find("cluster-rm"); stillExists {
		t.Errorf("el cluster debía haberse eliminado de clusters.json")
	}
}
