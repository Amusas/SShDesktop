package ssh

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// 1. Creacion de vessel-net cuando no existe
func TestEnsureNATNetwork_CreateWhenMissing(t *testing.T) {
	var executed []string
	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		executed = append(executed, command)
		switch {
		case strings.Contains(command, "natnetwork list"):
			return "NAT Networks:\n\n0 networks found\n", nil
		case strings.Contains(command, "natnetwork add"):
			return "", nil
		case strings.Contains(command, "natnetwork start"):
			return "", nil
		default:
			return "", fmt.Errorf("comando inesperado: %s", command)
		}
	})
	defer restore()

	cfg := Config{Host: "192.168.101.80"}
	info, err := EnsureNATNetwork(context.Background(), cfg, "vessel-net", "10.250.0.0/24", testLogger())
	if err != nil {
		t.Fatalf("EnsureNATNetwork fallo inesperadamente: %v", err)
	}
	if info.Name != "vessel-net" || info.CIDR != "10.250.0.0/24" || !info.Enabled || !info.DHCP {
		t.Fatalf("informacion de NAT Network inesperada: %+v", info)
	}

	var sawAdd, sawStart bool
	for _, cmd := range executed {
		if strings.Contains(cmd, `VBoxManage natnetwork add --netname "vessel-net" --network "10.250.0.0/24" --enable --dhcp on`) {
			sawAdd = true
		}
		if strings.Contains(cmd, `VBoxManage natnetwork start --netname "vessel-net"`) {
			sawStart = true
		}
	}
	if !sawAdd {
		t.Errorf("esperaba comando 'natnetwork add', comandos ejecutados: %v", executed)
	}
	if !sawStart {
		t.Errorf("esperaba comando 'natnetwork start', comandos ejecutados: %v", executed)
	}
}

// 2. Reutilizacion de vessel-net cuando ya existe y es compatible
func TestEnsureNATNetwork_ReuseExisting(t *testing.T) {
	var executed []string
	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		executed = append(executed, command)
		switch {
		case strings.Contains(command, "natnetwork list"):
			return `NAT Networks:

Name:        vessel-net
Network:     10.250.0.0/24
Gateway:     10.250.0.1
DHCP Server: Yes
IPv6:        No
Enabled:     Yes

1 network found
`, nil
		case strings.Contains(command, "natnetwork start"):
			return "NAT network 'vessel-net' is already running", fmt.Errorf("already running")
		default:
			return "", fmt.Errorf("no debio ejecutar: %s", command)
		}
	})
	defer restore()

	cfg := Config{Host: "192.168.101.80"}
	info, err := EnsureNATNetwork(context.Background(), cfg, "vessel-net", "10.250.0.0/24", testLogger())
	if err != nil {
		t.Fatalf("EnsureNATNetwork fallo al reutilizar red existente: %v", err)
	}
	if info.Name != "vessel-net" || info.CIDR != "10.250.0.0/24" {
		t.Fatalf("info inesperada: %+v", info)
	}
	for _, cmd := range executed {
		if strings.Contains(cmd, "natnetwork add") || strings.Contains(cmd, "natnetwork modify") {
			t.Errorf("no debio recrear ni modificar una red ya habilitada: %s", cmd)
		}
	}
}

// 3. Inicio y habilitacion de vessel-net cuando existe pero esta detenida o deshabilitada
func TestEnsureNATNetwork_StartWhenStoppedOrDisabled(t *testing.T) {
	var executed []string
	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		executed = append(executed, command)
		switch {
		case strings.Contains(command, "natnetwork list"):
			return `NAT Networks:

Name:        vessel-net
Network:     10.250.0.0/24
Gateway:     10.250.0.1
DHCP Server: No
Enabled:     No
`, nil
		case strings.Contains(command, "natnetwork modify"):
			return "", nil
		case strings.Contains(command, "natnetwork start"):
			return "", nil
		default:
			return "", fmt.Errorf("comando inesperado: %s", command)
		}
	})
	defer restore()

	cfg := Config{Host: "192.168.101.80"}
	info, err := EnsureNATNetwork(context.Background(), cfg, "vessel-net", "10.250.0.0/24", testLogger())
	if err != nil {
		t.Fatalf("EnsureNATNetwork fallo al reactivar red detenida: %v", err)
	}
	if !info.Enabled || !info.DHCP || !info.Active {
		t.Fatalf("esperaba red habilitada, con DHCP y activa: %+v", info)
	}

	var sawModify, sawStart bool
	for _, cmd := range executed {
		if strings.Contains(cmd, `VBoxManage natnetwork modify --netname "vessel-net" --enable --dhcp on`) {
			sawModify = true
		}
		if strings.Contains(cmd, `VBoxManage natnetwork start --netname "vessel-net"`) {
			sawStart = true
		}
	}
	if !sawModify || !sawStart {
		t.Errorf("esperaba modify y start, ejecutados: %v", executed)
	}
}

// 4. Deteccion de red existente con configuracion incompatible o rango IP superpuesto
func TestValidateNATNetworkCIDR_ConflictsAndOverlaps(t *testing.T) {
	existing := []NATNetworkInfo{
		{Name: "NatNetwork", CIDR: "10.0.2.0/24", Enabled: true, DHCP: true},
		{Name: "other-net", CIDR: "10.200.0.0/16", Enabled: true, DHCP: true},
	}

	// Caso A: Superposicion con otra NAT Network de VirtualBox (10.200.5.0/24 dentro de 10.200.0.0/16)
	if err := ValidateNATNetworkCIDR("vessel-net", "10.200.5.0/24", existing, "192.168.101.80", nil); err == nil {
		t.Errorf("esperaba error por superposicion con 'other-net' (10.200.0.0/16)")
	}

	// Caso B: Mismo nombre 'vessel-net' pero CIDR existente incompatible
	incompatibleExisting := []NATNetworkInfo{
		{Name: "vessel-net", CIDR: "10.99.0.0/24", Enabled: true, DHCP: true},
	}
	if err := ValidateNATNetworkCIDR("vessel-net", "10.250.0.0/24", incompatibleExisting, "192.168.101.80", nil); err == nil {
		t.Errorf("esperaba error cuando vessel-net ya existe con un CIDR diferente (10.99.0.0/24)")
	}

	// Caso C: Superposicion con la red local del host Windows (192.168.101.80)
	if err := ValidateNATNetworkCIDR("vessel-net", "192.168.101.0/24", nil, "192.168.101.80", nil); err == nil {
		t.Errorf("esperaba error por superposicion con la subred local del host 192.168.101.80")
	}

	// Caso D: Superposicion con Docker bridge por defecto (172.17.0.0/16)
	if err := ValidateNATNetworkCIDR("vessel-net", "172.17.1.0/24", nil, "192.168.101.80", nil); err == nil {
		t.Errorf("esperaba error por superposicion con Docker bridge (172.17.0.0/16)")
	}

	// Caso E: Superposicion con el rango reservado de VirtualBox NAT individual en NIC 2 (10.0.3.0/24)
	if err := ValidateNATNetworkCIDR("vessel-net", "10.0.3.0/24", nil, "192.168.101.80", nil); err == nil {
		t.Errorf("esperaba error por superposicion con rango NAT individual de NIC 2 (10.0.3.0/24)")
	}
}

// 5 & 7. Creacion de una VM con dos adaptadores de red y port forwarding en --natpf2
func TestCreateVMFromTemplate_DualNICAndNatpf2(t *testing.T) {
	var executed []string
	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		executed = append(executed, command)
		return "", nil
	})
	defer restore()

	req := VMTemplateCreateRequest{
		Name:           "vm-nodo-1",
		TemplateID:     "debian-13",
		OSType:         "Debian_64",
		BaseDiskPath:   `C:\VMManager\Templates\debian\base.vdi`,
		Cores:          2,
		MemoryMB:       2048,
		HostPort:       22001,
		GuestPort:      22,
		NATNetworkName: "vessel-net",
		NATNetworkCIDR: "10.250.0.0/24",
	}

	err := CreateVMFromTemplate(context.Background(), Config{Host: "192.168.101.80"}, req, testLogger())
	if err != nil {
		t.Fatalf("CreateVMFromTemplate fallo: %v", err)
	}

	var modifyCmd string
	for _, cmd := range executed {
		if strings.Contains(cmd, "modifyvm") && strings.Contains(cmd, "--nic1") {
			modifyCmd = cmd
			break
		}
	}
	if modifyCmd == "" {
		t.Fatalf("no se ejecuto modifyvm con configuracion de red; comandos: %v", executed)
	}
	if !strings.Contains(modifyCmd, `--nic1 natnetwork --nat-network1 "vessel-net"`) {
		t.Errorf("modifyvm no configuro NIC 1 en natnetwork 'vessel-net': %s", modifyCmd)
	}
	if !strings.Contains(modifyCmd, `--nic2 nat`) {
		t.Errorf("modifyvm no configuro NIC 2 en modo nat: %s", modifyCmd)
	}
	if !strings.Contains(modifyCmd, `--natpf2 "ssh,tcp,,22001,,22"`) {
		t.Errorf("modifyvm no configuro port forwarding SSH en --natpf2: %s", modifyCmd)
	}
	if strings.Contains(modifyCmd, "--natpf1") {
		t.Errorf("modifyvm NO debe usar --natpf1 en la nueva topologia: %s", modifyCmd)
	}
}

// 8. Descubrimiento de la IP privada mediante coincidencia de direccion MAC (sin asumir nombre de interfaz)
func TestDiscoverVMNetworkByMAC_SuccessWithUnusualInterfaceNames(t *testing.T) {
	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		if strings.Contains(command, "ip -j addr") {
			// Simulamos nombres de interfaz no estandar (ens192 y enx080027ddeeff) en orden inverso
			return `[
				{
					"ifindex": 1,
					"ifname": "lo",
					"flags": ["LOOPBACK","UP","LOWER_UP"],
					"operstate": "UNKNOWN",
					"address": "00:00:00:00:00:00",
					"addr_info": [{"family": "inet", "local": "127.0.0.1", "prefixlen": 8}]
				},
				{
					"ifindex": 2,
					"ifname": "enx080027ddeeff",
					"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
					"operstate": "UP",
					"address": "08:00:27:dd:ee:ff",
					"addr_info": [{"family": "inet", "local": "10.0.3.15", "prefixlen": 24}]
				},
				{
					"ifindex": 3,
					"ifname": "ens192",
					"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
					"operstate": "UP",
					"address": "08:00:27:aa:bb:cc",
					"addr_info": [{"family": "inet", "local": "10.250.0.14", "prefixlen": 24}]
				}
			]`, nil
		}
		return "", nil
	})
	defer restore()

	// Pasamos las MAC en formato VirtualBox sin dos puntos ("080027AABBCC" y "080027DDEEFF")
	disc, err := DiscoverVMNetworkByMAC(
		context.Background(),
		Config{Host: "127.0.0.1", Port: 22001},
		"080027AABBCC",
		"080027DDEEFF",
		"vessel-net",
		"10.250.0.0/24",
		testLogger(),
	)
	if err != nil {
		t.Fatalf("DiscoverVMNetworkByMAC fallo: %v", err)
	}
	if disc.PrivateIP != "10.250.0.14" {
		t.Errorf("esperaba PrivateIP 10.250.0.14 (de ens192), obtuve %s", disc.PrivateIP)
	}
	if disc.Interface1 != "ens192" || disc.Interface2 != "enx080027ddeeff" {
		t.Errorf("interfaces emparejadas incorrectamente: if1=%s, if2=%s", disc.Interface1, disc.Interface2)
	}
	if disc.Adapter1State != "connected" || disc.Adapter2State != "connected" {
		t.Errorf("estados de adaptador inesperados: a1=%s, a2=%s", disc.Adapter1State, disc.Adapter2State)
	}
}

// 10. Consulta de una VM recien creada que aun no tiene IP asignada -> solicita DHCP y obtiene IP
func TestDiscoverVMNetworkByMAC_TriggersDHCPWhenMissingIP(t *testing.T) {
	callCount := 0
	var dhcpTriggered bool

	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		if strings.Contains(command, "ip -j addr") {
			callCount++
			if callCount == 1 {
				// Primera consulta: la interfaz eth_shared esta DOWN y sin IP
				return `[
					{
						"ifindex": 2,
						"ifname": "eth_shared",
						"flags": ["BROADCAST","MULTICAST"],
						"operstate": "DOWN",
						"address": "08:00:27:11:22:33",
						"addr_info": []
					},
					{
						"ifindex": 3,
						"ifname": "eth_ssh",
						"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
						"operstate": "UP",
						"address": "08:00:27:44:55:66",
						"addr_info": [{"family": "inet", "local": "10.0.3.15", "prefixlen": 24}]
					}
				]`, nil
			}
			// Segunda consulta tras ejecutar remedio DHCP: ya tiene 10.250.0.21
			return `[
				{
					"ifindex": 2,
					"ifname": "eth_shared",
					"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
					"operstate": "UP",
					"address": "08:00:27:11:22:33",
					"addr_info": [{"family": "inet", "local": "10.250.0.21", "prefixlen": 24}]
				},
				{
					"ifindex": 3,
					"ifname": "eth_ssh",
					"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
					"operstate": "UP",
					"address": "08:00:27:44:55:66",
					"addr_info": [{"family": "inet", "local": "10.0.3.15", "prefixlen": 24}]
				}
			]`, nil
		}
		if strings.Contains(command, "dhclient eth_shared") {
			dhcpTriggered = true
			return "", nil
		}
		return "", nil
	})
	defer restore()

	disc, err := DiscoverVMNetworkByMAC(
		context.Background(),
		Config{Host: "127.0.0.1", Port: 22001},
		"080027112233",
		"080027445566",
		"vessel-net",
		"10.250.0.0/24",
		testLogger(),
	)
	if err != nil {
		t.Fatalf("esperaba exito tras remedio DHCP, obtuve error: %v", err)
	}
	if !dhcpTriggered {
		t.Errorf("esperaba que se ejecutara solicitud DHCP sobre eth_shared")
	}
	if disc.PrivateIP != "10.250.0.21" {
		t.Errorf("esperaba IP 10.250.0.21, obtuve %s", disc.PrivateIP)
	}
}

// 12. Migracion de VMs existentes creadas con la configuracion anterior (--nic1 nat --natpf1)
func TestMigrateVMToDualNIC_LegacySingleNIC(t *testing.T) {
	var executed []string
	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		executed = append(executed, command)
		switch {
		case strings.Contains(command, "showvminfo"):
			return `name="legacy-vm"
VMState="poweroff"
nic1="nat"
macaddress1="080027AABBCC"
nic2="none"
Forwarding(0)="ssh,tcp,,22000,,22"
`, nil
		case strings.Contains(command, "natnetwork list"):
			return `Name: vessel-net
Network: 10.250.0.0/24
Enabled: Yes
DHCP Server: Yes
`, nil
		default:
			return "", nil
		}
	})
	defer restore()

	migrated, err := MigrateVMToDualNIC(
		context.Background(),
		Config{Host: "192.168.101.80"},
		"legacy-vm",
		"vessel-net",
		"10.250.0.0/24",
		22000,
		22,
		testLogger(),
	)
	if err != nil {
		t.Fatalf("MigrateVMToDualNIC fallo: %v", err)
	}
	if !migrated {
		t.Fatalf("esperaba migrated=true para VM con nic1=nat")
	}

	var sawDeleteNatpf1, sawDualModify bool
	for _, cmd := range executed {
		if strings.Contains(cmd, `--natpf1 delete "ssh"`) {
			sawDeleteNatpf1 = true
		}
		if strings.Contains(cmd, `--nic1 natnetwork --nat-network1 "vessel-net"`) && strings.Contains(cmd, `--nic2 nat`) && strings.Contains(cmd, `--natpf2 "ssh,tcp,,22000,,22"`) {
			sawDualModify = true
		}
	}
	if !sawDeleteNatpf1 || !sawDualModify {
		t.Errorf("comandos de migracion incompletos: %v", executed)
	}
}

// 13 & 14. Manejo de fallos en comandos SSH y cuando la interfaz Linux no recibe IP por DHCP
func TestDiscoverVMNetworkByMAC_SSHFailureAndDHCPFailure(t *testing.T) {
	// Caso 13: Fallo de comando SSH
	restoreSSH := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		return "", fmt.Errorf("ssh: handshake failed: connection refused")
	})
	_, err := DiscoverVMNetworkByMAC(context.Background(), Config{Host: "127.0.0.1", Port: 22001}, "08:00:27:aa:bb:cc", "08:00:27:dd:ee:ff", "vessel-net", "10.250.0.0/24", testLogger())
	restoreSSH()
	if err == nil {
		t.Fatalf("esperaba error cuando falla la conexion SSH")
	}

	// Caso 14: La interfaz existe pero el servidor DHCP no entrega IP (permanece sin IP o con link-local 169.254.x.x)
	restoreDHCP := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		if strings.Contains(command, "ip -j addr") {
			return `[
				{
					"ifindex": 2,
					"ifname": "enp0s3",
					"flags": ["BROADCAST","MULTICAST","UP","LOWER_UP"],
					"operstate": "UP",
					"address": "08:00:27:aa:bb:cc",
					"addr_info": [{"family": "inet", "local": "169.254.10.20", "prefixlen": 16}]
				}
			]`, nil
		}
		return "", nil
	})
	defer restoreDHCP()

	disc, err := DiscoverVMNetworkByMAC(context.Background(), Config{Host: "127.0.0.1", Port: 22001}, "08:00:27:aa:bb:cc", "", "vessel-net", "10.250.0.0/24", testLogger())
	if err == nil {
		t.Fatalf("esperaba error cuando la interfaz solo tiene IP link-local (169.254.10.20) y no recibe DHCP en 10.250.0.0/24")
	}
	if disc == nil || disc.PrivateIP != "" {
		t.Errorf("no debio aceptar una IP link-local como PrivateIP valida: %+v", disc)
	}
}

// 15. Verificacion de conectividad entre dos VMs en vessel-net (ICMP y fallback TCP)
func TestVerifyInterVMConnectivity_ICMPAndTCPFallback(t *testing.T) {
	// Subcaso A: ICMP y TCP responden
	restoreA := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		if strings.Contains(command, "ping -c 3") {
			return "PING 10.250.0.12 (10.250.0.12) 56(84) bytes of data.\n3 packets transmitted, 3 received, 0% packet loss\nrtt min/avg/max/mdev = 0.210/0.420/0.610/0.050 ms\n", nil
		}
		if strings.Contains(command, "VESSEL_TCP_OK") {
			return "VESSEL_TCP_OK\n", nil
		}
		return "", nil
	})
	resA, err := VerifyInterVMConnectivity(context.Background(), Config{}, "vm1", "vm2", "10.250.0.12", 22, testLogger())
	restoreA()
	if err != nil || !resA.Reachable || !resA.ICMPReachable || !resA.TCPReachable {
		t.Fatalf("esperaba conectividad completa ICMP+TCP: err=%v, res=%+v", err, resA)
	}
	if resA.LatencyMs != 0.42 {
		t.Errorf("esperaba latencia 0.42 ms, obtuve %v", resA.LatencyMs)
	}

	// Subcaso B: ICMP bloqueado por firewall pero puerto TCP 22 alcanzable (fallback TCP)
	restoreB := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		if strings.Contains(command, "ping -c 3") {
			return "3 packets transmitted, 0 received, 100% packet loss", fmt.Errorf("exit status 1")
		}
		if strings.Contains(command, "VESSEL_TCP_OK") {
			return "VESSEL_TCP_OK\n", nil
		}
		return "", nil
	})
	resB, err := VerifyInterVMConnectivity(context.Background(), Config{}, "vm1", "vm2", "10.250.0.12", 22, testLogger())
	restoreB()
	if err != nil || !resB.Reachable || resB.ICMPReachable || !resB.TCPReachable {
		t.Fatalf("esperaba Reachable=true via fallback TCP cuando ICMP falla: err=%v, res=%+v", err, resB)
	}
}

// 16. Eliminacion de una VM sin eliminar vessel-net
func TestDeleteVM_NeverDeletesNATNetwork(t *testing.T) {
	var executed []string
	restore := SetVBoxCommandRunner(func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
		executed = append(executed, command)
		if strings.Contains(command, "list runningvms") {
			return "", nil
		}
		return "", nil
	})
	defer restore()

	if err := DeleteVM(Config{Host: "192.168.101.80"}, "vm-nodo-1", testLogger()); err != nil {
		t.Fatalf("DeleteVM fallo: %v", err)
	}
	for _, cmd := range executed {
		if strings.Contains(strings.ToLower(cmd), "natnetwork") {
			t.Fatalf("DeleteVM NUNCA debe eliminar ni detener la NAT Network compartida: %s", cmd)
		}
	}
}
