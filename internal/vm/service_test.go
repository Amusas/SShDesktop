package vm

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/keys"
	"github.com/feliren/ssh-web/internal/ssh"
)

func TestFindOSByISO(t *testing.T) {
	got, err := FindOSByISO("ubuntu-24.04.5-live-server-amd64.iso")
	if err != nil {
		t.Fatalf("esperaba ISO conocida: %v", err)
	}
	if !strings.HasPrefix(got.Name, "Ubuntu 24.04") {
		t.Fatalf("OS soportada derivada incorrecta: %q", got.Name)
	}
}

func TestVMStore(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "vms.json")
	store := NewStore(storePath)

	rec := VMRecord{
		Name:     "Test-VM-1",
		OS:       "Ubuntu 24.04 LTS",
		ISO:      "ubuntu-24.04.5-live-server-amd64.iso",
		Cores:    4,
		MemoryMB: 4096,
		DiskGB:   60,
		State:    "Creada",
	}

	added, err := store.Add(rec)
	if err != nil {
		t.Fatalf("error agregando VM: %v", err)
	}
	if added.Name != "Test-VM-1" || added.Cores != 4 || added.MemoryMB != 4096 || added.DiskGB != 60 {
		t.Fatalf("valores de VM incorrectos: %+v", added)
	}

	found, exists, err := store.Find("Test-VM-1")
	if err != nil || !exists {
		t.Fatalf("error buscando VM: %v", err)
	}
	if found.DiskGB != 60 {
		t.Fatalf("esperaba 60 GB de disco, obtuve %d", found.DiskGB)
	}

	list, err := store.Load()
	if err != nil || len(list) != 1 {
		t.Fatalf("error listando VMs: %v", err)
	}

	if err := store.Delete("Test-VM-1"); err != nil {
		t.Fatalf("error eliminando VM: %v", err)
	}

	listAfter, _ := store.Load()
	if len(listAfter) != 0 {
		t.Fatalf("esperaba 0 VMs, obtuve %d", len(listAfter))
	}
}

// 6. Asignacion de puertos SSH unicos para multiples VMs
func TestPortAllocator_UniqueSSHDPorts(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(filepath.Join(tempDir, "vms.json"))
	alloc := NewPortAllocator(22001, store)

	p1, err := alloc.AllocatePort(context.Background())
	if err != nil {
		t.Fatalf("fallo al asignar puerto 1: %v", err)
	}
	_, _ = store.Add(VMRecord{Name: "vm1", HostPort: p1, State: "running"})

	p2, err := alloc.AllocatePort(context.Background())
	if err != nil {
		t.Fatalf("fallo al asignar puerto 2: %v", err)
	}
	_, _ = store.Add(VMRecord{Name: "vm2", HostPort: p2, State: "running"})

	p3, err := alloc.AllocatePort(context.Background())
	if err != nil {
		t.Fatalf("fallo al asignar puerto 3: %v", err)
	}

	if p1 == p2 || p2 == p3 || p1 == p3 {
		t.Fatalf("los puertos SSH asignados deben ser unicos, obtuve: p1=%d, p2=%d, p3=%d", p1, p2, p3)
	}
}

// 9, 11 & 15. Persistencia de IP privada en vms.json, consulta de VM encendida/apagada y conectividad entre dos VMs
func TestService_NetworkPersistencePoweredOffAndConnectivity(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "vms.json")
	keyPath := filepath.Join(tempDir, "keys.json")
	store := NewStore(storePath)

	_, err := store.Add(VMRecord{
		Name:     "vm-alpha",
		OS:       "Debian 13",
		Cores:    2,
		MemoryMB: 2048,
		State:    "running",
		Host:     "192.168.101.80",
		HostPort: 22001,
		SSHUser:  "vm_alpha_user",
	})
	if err != nil {
		t.Fatalf("error registrando vm-alpha: %v", err)
	}

	_, err = store.Add(VMRecord{
		Name:        "vm-beta",
		OS:          "Debian 13",
		Cores:       2,
		MemoryMB:    2048,
		State:       "poweroff",
		Host:        "192.168.101.80",
		HostPort:    22002,
		SSHUser:     "vm_beta_user",
		PrivateIP:   "10.250.0.12",
		NetworkName: "vessel-net",
		NetworkCIDR: "10.250.0.0/24",
		MACAddress1: "08:00:27:bb:bb:01",
		MACAddress2: "08:00:27:bb:bb:02",
	})
	if err != nil {
		t.Fatalf("error registrando vm-beta: %v", err)
	}

	cfg := &config.Config{
		SSHHost:        "192.168.101.80",
		SSHPort:        22,
		SSHUser:        "admin",
		SSHRoot:        tempDir,
		BasePort:       22000,
		NATNetworkName: "vessel-net",
		NATNetworkCIDR: "10.250.0.0/24",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := &Service{
		cfg:           cfg,
		log:           logger,
		vmStore:       store,
		keyStore:      keys.NewStore(keyPath),
		portAlloc:     NewPortAllocator(cfg.BasePort, store),
		knownHostsMgr: ssh.NewKnownHostsManager(filepath.Join(tempDir, "known_hosts")),
	}

	restore := ssh.SetVBoxCommandRunner(func(ctx context.Context, sshCfg ssh.Config, command string, log *slog.Logger) (string, error) {
		switch {
		case strings.Contains(command, `showvminfo "vm-alpha"`):
			return `name="vm-alpha"
VMState="running"
nic1="natnetwork"
natnet1="vessel-net"
macaddress1="080027AAAA01"
nic2="nat"
macaddress2="080027AAAA02"
`, nil
		case strings.Contains(command, `showvminfo "vm-beta"`):
			return `name="vm-beta"
VMState="poweroff"
nic1="natnetwork"
natnet1="vessel-net"
macaddress1="080027BBBB01"
nic2="nat"
macaddress2="080027BBBB02"
`, nil
		case strings.Contains(command, "ip -j addr"):
			return `[
				{
					"ifindex": 2,
					"ifname": "enp0s3",
					"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
					"operstate": "UP",
					"address": "08:00:27:aa:aa:01",
					"addr_info": [{"family": "inet", "local": "10.250.0.11", "prefixlen": 24}]
				},
				{
					"ifindex": 3,
					"ifname": "enp0s8",
					"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
					"operstate": "UP",
					"address": "08:00:27:aa:aa:02",
					"addr_info": [{"family": "inet", "local": "10.0.3.15", "prefixlen": 24}]
				}
			]`, nil
		case strings.Contains(command, "ping -c 3 -W 2 10.250.0.12"):
			return "3 packets transmitted, 3 received, 0% packet loss\nrtt min/avg/max/mdev = 0.200/0.350/0.500/0.050 ms\n", nil
		case strings.Contains(command, "VESSEL_TCP_OK"):
			return "VESSEL_TCP_OK\n", nil
		default:
			return "", nil
		}
	})
	defer restore()

	// Caso 9: Descubrimiento de red de vm-alpha y persistencia en vms.json
	alphaNet, err := svc.GetNetworkInfo(context.Background(), "vm-alpha", false)
	if err != nil {
		t.Fatalf("GetNetworkInfo para vm-alpha fallo: %v", err)
	}
	if alphaNet.PrivateIP != "10.250.0.11" || alphaNet.NetworkName != "vessel-net" {
		t.Fatalf("datos de red de vm-alpha inesperados: %+v", alphaNet)
	}

	persistedAlpha, ok, _ := store.Find("vm-alpha")
	if !ok || persistedAlpha.PrivateIP != "10.250.0.11" || persistedAlpha.MACAddress1 != "08:00:27:aa:aa:01" {
		t.Fatalf("no se persistieron correctamente los datos de red de vm-alpha en vms.json: %+v", persistedAlpha)
	}

	// Caso 11: Consulta de VM apagada (vm-beta) -> estado disconnected pero conserva ultima IP conocida
	betaNet, err := svc.GetNetworkInfo(context.Background(), "vm-beta", false)
	if err != nil {
		t.Fatalf("GetNetworkInfo para vm-beta (apagada) fallo: %v", err)
	}
	if betaNet.Adapter1State != "disconnected" || betaNet.Adapter2State != "disconnected" {
		t.Errorf("esperaba adaptadores disconnected para VM apagada, obtuve: %+v", betaNet)
	}
	if betaNet.PrivateIP != "10.250.0.12" {
		t.Errorf("esperaba conservar la ultima IP privada conocida 10.250.0.12 en VM apagada, obtuve: %s", betaNet.PrivateIP)
	}

	// Caso 15: Verificacion de conectividad desde vm-alpha hacia la IP de vm-beta (10.250.0.12)
	connRes, err := svc.CheckConnectivity(context.Background(), "vm-alpha", "10.250.0.12", 22)
	if err != nil {
		t.Fatalf("CheckConnectivity fallo: %v", err)
	}
	if !connRes.Reachable || !connRes.ICMPReachable || !connRes.TCPReachable {
		t.Errorf("esperaba conectividad exitosa entre vm-alpha y 10.250.0.12: %+v", connRes)
	}
}
