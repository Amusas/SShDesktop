package ssh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultNATNetworkName nombre predeterminado de la red NAT compartida de Vessel
	DefaultNATNetworkName = "vessel-net"
	// DefaultNATNetworkCIDR rango IPv4 predeterminado de la red NAT compartida de Vessel
	DefaultNATNetworkCIDR = "10.250.0.0/24"
)

var natNetMu sync.Mutex

// NATNetworkInfo describe una red NAT Network registrada en VirtualBox
type NATNetworkInfo struct {
	Name    string `json:"name"`
	CIDR    string `json:"cidr"`
	Gateway string `json:"gateway,omitempty"`
	Enabled bool   `json:"enabled"`
	DHCP    bool   `json:"dhcp"`
	Active  bool   `json:"active,omitempty"`
}

// GuestAddrEntry representa una entrada de direccion IP en la salida de `ip -j addr`
type GuestAddrEntry struct {
	Family    string `json:"family"`
	Local     string `json:"local"`
	PrefixLen int    `json:"prefixlen"`
	Scope     string `json:"scope,omitempty"`
}

// GuestInterfaceInfo representa una interfaz de red Linux en la salida de `ip -j addr`
type GuestInterfaceInfo struct {
	IfIndex   int              `json:"ifindex"`
	IfName    string           `json:"ifname"`
	Flags     []string         `json:"flags"`
	OperState string           `json:"operstate"`
	LinkType  string           `json:"link_type"`
	Address   string           `json:"address"`
	AddrInfo  []GuestAddrEntry `json:"addr_info"`
}

// DiscoveredVMNetwork contiene la informacion de red descubierta por MAC dentro de la VM Linux
type DiscoveredVMNetwork struct {
	PrivateIP     string `json:"privateIp"`
	NetworkName   string `json:"networkName"`
	NetworkCIDR   string `json:"networkCidr"`
	MACAddress1   string `json:"macAddress1"`
	MACAddress2   string `json:"macAddress2"`
	Interface1    string `json:"interface1"`
	Interface2    string `json:"interface2"`
	Adapter1State string `json:"adapter1State"`
	Adapter2State string `json:"adapter2State"`
	Adapter2IP    string `json:"adapter2Ip,omitempty"`
	DiscoveredAt  string `json:"discoveredAt"`
}

// VMConnectivityResult contiene el resultado de una prueba de conectividad entre dos VMs
type VMConnectivityResult struct {
	SourceVM      string  `json:"sourceVm"`
	TargetVM      string  `json:"targetVm,omitempty"`
	TargetIP      string  `json:"targetIp"`
	TargetPort    int     `json:"targetPort,omitempty"`
	Reachable     bool    `json:"reachable"`
	ICMPReachable bool    `json:"icmpReachable"`
	TCPReachable  bool    `json:"tcpReachable"`
	LatencyMs     float64 `json:"latencyMs,omitempty"`
	Method        string  `json:"method"`
	Output        string  `json:"output,omitempty"`
	Error         string  `json:"error,omitempty"`
	CheckedAt     string  `json:"checkedAt"`
}

// NormalizeMAC normaliza una direccion MAC (ej: "080027AABBCC" o "08-00-27-AA-BB-CC")
// al formato estandar en minusculas separado por dos puntos ("08:00:27:aa:bb:cc").
func NormalizeMAC(mac string) string {
	cleaned := strings.ToLower(strings.TrimSpace(mac))
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	if len(cleaned) != 12 {
		return strings.ToLower(strings.TrimSpace(mac))
	}
	for _, c := range cleaned {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return strings.ToLower(strings.TrimSpace(mac))
		}
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		cleaned[0:2], cleaned[2:4], cleaned[4:6],
		cleaned[6:8], cleaned[8:10], cleaned[10:12])
}

// ParseNATNetworkList parsea la salida de `VBoxManage natnetwork list` o `VBoxManage list natnetworks`
func ParseNATNetworkList(output string) []NATNetworkInfo {
	var networks []NATNetworkInfo
	var current *NATNetworkInfo

	flushCurrent := func() {
		if current != nil && strings.TrimSpace(current.Name) != "" {
			networks = append(networks, *current)
		}
		current = nil
	}

	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			flushCurrent()
			continue
		}

		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		keyLower := strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)

		switch keyLower {
		case "name", "networkname", "network name":
			flushCurrent()
			current = &NATNetworkInfo{
				Name:   val,
				Active: true, // Por defecto si está habilitada
			}
		case "network", "ipv4 network", "cidr", "ipv4 prefix":
			if current != nil {
				current.CIDR = val
			}
		case "gateway", "ipv4 gateway":
			if current != nil {
				current.Gateway = val
			}
		case "enabled":
			if current != nil {
				current.Enabled = isAffirmative(val)
			}
		case "dhcp server", "dhcp":
			if current != nil {
				current.DHCP = isAffirmative(val)
			}
		case "active", "running", "state", "status":
			if current != nil {
				current.Active = isAffirmative(val)
			}
		}
	}
	flushCurrent()
	return networks
}

func isAffirmative(val string) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	return v == "yes" || v == "true" || v == "on" || v == "1" || v == "enabled" || v == "running" || v == "active"
}

// ListNATNetworks consulta las NAT Networks configuradas en VirtualBox
func ListNATNetworks(ctx context.Context, cfg Config, log *slog.Logger) ([]NATNetworkInfo, error) {
	out, err := runHostCmdContext(ctx, cfg, `VBoxManage natnetwork list`, log)
	if err != nil {
		// Fallback para versiones de VBoxManage que usan `list natnetworks`
		outFallback, errFallback := runHostCmdContext(ctx, cfg, `VBoxManage list natnetworks`, log)
		if errFallback != nil {
			return nil, fmt.Errorf("consultando redes NAT de VirtualBox: %w", err)
		}
		out = outFallback
	}
	return ParseNATNetworkList(out), nil
}

// CIDROverlaps determina si dos rangos CIDR IPv4 se superponen
func CIDROverlaps(cidrA, cidrB string) (bool, error) {
	_, netA, err := net.ParseCIDR(strings.TrimSpace(cidrA))
	if err != nil {
		return false, fmt.Errorf("CIDR invalido '%s': %w", cidrA, err)
	}
	_, netB, err := net.ParseCIDR(strings.TrimSpace(cidrB))
	if err != nil {
		return false, fmt.Errorf("CIDR invalido '%s': %w", cidrB, err)
	}
	return netA.Contains(netB.IP) || netB.Contains(netA.IP), nil
}

// ValidateNATNetworkCIDR valida que el CIDR propuesto sea privado, bien formado y no tenga colisiones
// con la red local del host, las redes NAT individuales de VirtualBox, Docker por defecto ni otras NAT Networks.
func ValidateNATNetworkCIDR(targetName, targetCIDR string, existing []NATNetworkInfo, hostIP string, extraSubnets []string) error {
	targetName = strings.TrimSpace(targetName)
	targetCIDR = strings.TrimSpace(targetCIDR)
	if targetName == "" {
		return fmt.Errorf("el nombre de la NAT Network no puede estar vacio")
	}
	if targetCIDR == "" {
		return fmt.Errorf("el rango CIDR de la NAT Network no puede estar vacio")
	}

	ip, ipNet, err := net.ParseCIDR(targetCIDR)
	if err != nil {
		return fmt.Errorf("rango CIDR '%s' invalido: %w", targetCIDR, err)
	}
	if ip.To4() == nil || ipNet.IP.To4() == nil {
		return fmt.Errorf("el rango CIDR '%s' debe ser una red IPv4 valida", targetCIDR)
	}

	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones < 8 || ones > 29 {
		return fmt.Errorf("el prefijo de red '%s' debe estar entre /8 y /29", targetCIDR)
	}

	if !ipNet.IP.IsPrivate() {
		return fmt.Errorf("el rango CIDR '%s' debe pertenecer a un bloque IPv4 privado (RFC 1918: 10.0.0.0/8, 172.16.0.0/12 o 192.168.0.0/16)", targetCIDR)
	}

	canonicalCIDR := ipNet.String()

	// 1. Verificar contra las redes NAT Networks existentes en VirtualBox
	for _, netInfo := range existing {
		if strings.EqualFold(strings.TrimSpace(netInfo.Name), targetName) {
			if netInfo.CIDR != "" {
				_, existingNet, pErr := net.ParseCIDR(strings.TrimSpace(netInfo.CIDR))
				if pErr == nil && existingNet.String() != canonicalCIDR {
					return fmt.Errorf("la red NAT '%s' ya existe en VirtualBox con un CIDR incompatible (%s != %s); no se modificara automaticamente",
						targetName, existingNet.String(), canonicalCIDR)
				} else if pErr != nil && !strings.EqualFold(strings.TrimSpace(netInfo.CIDR), targetCIDR) {
					return fmt.Errorf("la red NAT '%s' ya existe en VirtualBox con una configuracion de red incompatible (%s != %s)",
						targetName, netInfo.CIDR, targetCIDR)
				}
			}
			continue
		}

		if netInfo.CIDR != "" {
			if overlaps, oErr := CIDROverlaps(canonicalCIDR, netInfo.CIDR); oErr == nil && overlaps {
				return fmt.Errorf("el rango CIDR '%s' para '%s' se superpone con la NAT Network existente '%s' (%s)",
					canonicalCIDR, targetName, netInfo.Name, netInfo.CIDR)
			}
		}
	}

	// 2. Verificar contra subredes reservadas de VirtualBox NAT individual (10.0.2.0/24 para NIC1 NAT, 10.0.3.0/24 para NIC2 NAT),
	// Host-Only predeterminada (192.168.56.0/24) y Docker bridge por defecto (172.17.0.0/16)
	reservedSubnets := []struct {
		cidr   string
		reason string
	}{
		{cidr: "10.0.2.0/24", reason: "rango reservado por VirtualBox NAT individual (NIC 1)"},
		{cidr: "10.0.3.0/24", reason: "rango reservado por VirtualBox NAT individual (NIC 2 SSH)"},
		{cidr: "192.168.56.0/24", reason: "rango predeterminado de VirtualBox Host-Only Adapter"},
		{cidr: "172.17.0.0/16", reason: "red bridge predeterminada de Docker (docker0)"},
	}

	for _, res := range reservedSubnets {
		if overlaps, oErr := CIDROverlaps(canonicalCIDR, res.cidr); oErr == nil && overlaps {
			return fmt.Errorf("el rango CIDR '%s' entra en conflicto con %s (%s)", canonicalCIDR, res.reason, res.cidr)
		}
	}

	// 3. Verificar contra la IP/subred local del host Windows
	hostIP = strings.TrimSpace(hostIP)
	if parsedHostIP := net.ParseIP(hostIP); parsedHostIP != nil && parsedHostIP.To4() != nil && !parsedHostIP.IsLoopback() {
		if ipNet.Contains(parsedHostIP) {
			return fmt.Errorf("el rango CIDR '%s' se superpone con la direccion IP del host (%s)", canonicalCIDR, hostIP)
		}
		hostSubnet := fmt.Sprintf("%s/24", parsedHostIP.Mask(net.CIDRMask(24, 32)).String())
		if overlaps, oErr := CIDROverlaps(canonicalCIDR, hostSubnet); oErr == nil && overlaps {
			return fmt.Errorf("el rango CIDR '%s' se superpone con la subred local del host (%s)", canonicalCIDR, hostSubnet)
		}
	}

	// 4. Verificar contra subredes adicionales reportadas (ej: redes Docker o interfaces extra)
	for _, extra := range extraSubnets {
		extra = strings.TrimSpace(extra)
		if extra == "" {
			continue
		}
		if overlaps, oErr := CIDROverlaps(canonicalCIDR, extra); oErr == nil && overlaps {
			return fmt.Errorf("el rango CIDR '%s' se superpone con la red existente '%s'", canonicalCIDR, extra)
		}
	}

	return nil
}

// EnsureNATNetwork garantiza de forma idempotente que la red compartida NAT Network exista,
// tenga una configuracion CIDR compatible sin superposiciones, tenga DHCP habilitado y este iniciada.
func EnsureNATNetwork(ctx context.Context, cfg Config, netName, cidr string, log *slog.Logger) (*NATNetworkInfo, error) {
	natNetMu.Lock()
	defer natNetMu.Unlock()

	netName = strings.TrimSpace(netName)
	if netName == "" {
		netName = DefaultNATNetworkName
	}
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		cidr = DefaultNATNetworkCIDR
	}

	existing, err := ListNATNetworks(ctx, cfg, log)
	if err != nil {
		return nil, err
	}

	if err := ValidateNATNetworkCIDR(netName, cidr, existing, cfg.Host, nil); err != nil {
		return nil, err
	}

	_, parsedNet, _ := net.ParseCIDR(cidr)
	canonicalCIDR := parsedNet.String()

	var found *NATNetworkInfo
	for i := range existing {
		if strings.EqualFold(strings.TrimSpace(existing[i].Name), netName) {
			found = &existing[i]
			break
		}
	}

	if found == nil {
		log.Info("creando red NAT compartida en VirtualBox", "netName", netName, "cidr", canonicalCIDR)
		addCmd := fmt.Sprintf(`VBoxManage natnetwork add --netname "%s" --network "%s" --enable --dhcp on`,
			escapeForDoubleQuote(netName), escapeForDoubleQuote(canonicalCIDR))
		if _, err := runHostCmdContext(ctx, cfg, addCmd, log); err != nil {
			return nil, fmt.Errorf("creando NAT Network '%s' (%s): %w", netName, canonicalCIDR, err)
		}

		startCmd := fmt.Sprintf(`VBoxManage natnetwork start --netname "%s"`, escapeForDoubleQuote(netName))
		if out, sErr := runHostCmdContext(ctx, cfg, startCmd, log); sErr != nil && !isNATNetAlreadyRunningErr(out, sErr) {
			log.Warn("aviso al iniciar NAT Network recien creada", "netName", netName, "error", sErr)
		}

		return &NATNetworkInfo{
			Name:    netName,
			CIDR:    canonicalCIDR,
			Enabled: true,
			DHCP:    true,
			Active:  true,
		}, nil
	}

	// Si ya existe, verificar que este habilitada y con DHCP activo
	if !found.Enabled || !found.DHCP {
		log.Info("habilitando red NAT compartida y servidor DHCP en VirtualBox", "netName", netName)
		modCmd := fmt.Sprintf(`VBoxManage natnetwork modify --netname "%s" --enable --dhcp on`, escapeForDoubleQuote(netName))
		if _, err := runHostCmdContext(ctx, cfg, modCmd, log); err != nil {
			return nil, fmt.Errorf("habilitando NAT Network '%s': %w", netName, err)
		}
		found.Enabled = true
		found.DHCP = true
	}

	// Garantizar que la red este iniciada (especialmente si estaba detenida)
	startCmd := fmt.Sprintf(`VBoxManage natnetwork start --netname "%s"`, escapeForDoubleQuote(netName))
	if out, sErr := runHostCmdContext(ctx, cfg, startCmd, log); sErr != nil {
		if !isNATNetAlreadyRunningErr(out, sErr) {
			return nil, fmt.Errorf("iniciando NAT Network '%s': %w", netName, sErr)
		}
	}
	found.Active = true
	if found.CIDR == "" {
		found.CIDR = canonicalCIDR
	}

	return found, nil
}

func isNATNetAlreadyRunningErr(out string, err error) bool {
	combined := strings.ToLower(out)
	if err != nil {
		combined += " " + strings.ToLower(err.Error())
	}
	return strings.Contains(combined, "already running") ||
		strings.Contains(combined, "already active") ||
		strings.Contains(combined, "already started") ||
		strings.Contains(combined, "already exists")
}

// ParseGuestInterfacesJSON parsea la salida JSON de `ip -j addr` o `ip -j link`
func ParseGuestInterfacesJSON(raw string) ([]GuestInterfaceInfo, error) {
	trimmed := strings.TrimSpace(raw)
	startIdx := strings.Index(trimmed, "[")
	endIdx := strings.LastIndex(trimmed, "]")
	if startIdx >= 0 && endIdx > startIdx {
		trimmed = trimmed[startIdx : endIdx+1]
	}
	var ifaces []GuestInterfaceInfo
	if err := json.Unmarshal([]byte(trimmed), &ifaces); err != nil {
		return nil, err
	}
	for i := range ifaces {
		ifaces[i].Address = NormalizeMAC(ifaces[i].Address)
	}
	return ifaces, nil
}

// ParseGuestInterfacesFallback parsea la salida de `ip -o link` e `ip -o -4 addr` cuando `ip -j` no esta disponible
func ParseGuestInterfacesFallback(linkOut, addrOut string) []GuestInterfaceInfo {
	byName := make(map[string]*GuestInterfaceInfo)
	var order []string

	// Ejemplo `ip -o link`:
	// 2: enp0s3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP mode DEFAULT group default qlen 1000\    link/ether 08:00:27:aa:bb:cc brd ff:ff:ff:ff:ff:ff
	linkRe := regexp.MustCompile(`^\d+:\s+([^:@\s]+)(?:@[^:\s]+)?:\s+<([^>]*)>.*?(?:state\s+(\S+))?.*?(?:link/\S+\s+([0-9a-fA-F:]{17}))?`)
	for _, line := range strings.Split(linkOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := linkRe.FindStringSubmatch(line)
		if len(m) >= 3 {
			ifName := strings.TrimSpace(m[1])
			flags := strings.Split(m[2], ",")
			state := ""
			if len(m) >= 4 {
				state = strings.TrimSpace(m[3])
			}
			mac := ""
			if len(m) >= 5 {
				mac = NormalizeMAC(m[4])
			}
			info := &GuestInterfaceInfo{
				IfName:    ifName,
				Flags:     flags,
				OperState: state,
				Address:   mac,
			}
			byName[ifName] = info
			order = append(order, ifName)
		}
	}

	// Ejemplo `ip -o -4 addr`:
	// 2: enp0s3    inet 10.250.0.11/24 brd 10.250.0.255 scope global dynamic enp0s3
	addrRe := regexp.MustCompile(`^\d+:\s+([^:@\s]+)\s+inet\s+(\d+\.\d+\.\d+\.\d+)/(\d+)`)
	for _, line := range strings.Split(addrOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := addrRe.FindStringSubmatch(line)
		if len(m) == 4 {
			ifName := strings.TrimSpace(m[1])
			ipStr := strings.TrimSpace(m[2])
			prefix, _ := strconv.Atoi(m[3])
			info, exists := byName[ifName]
			if !exists {
				info = &GuestInterfaceInfo{IfName: ifName}
				byName[ifName] = info
				order = append(order, ifName)
			}
			info.AddrInfo = append(info.AddrInfo, GuestAddrEntry{
				Family:    "inet",
				Local:     ipStr,
				PrefixLen: prefix,
			})
		}
	}

	res := make([]GuestInterfaceInfo, 0, len(order))
	for _, name := range order {
		res = append(res, *byName[name])
	}
	return res
}

// ValidatePrivateIPInCIDR verifica que una direccion IP sea IPv4 valida, no sea loopback ni link-local,
// y pertenezca al rango CIDR configurado para la red compartida.
func ValidatePrivateIPInCIDR(ipStr, cidrStr string) error {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return fmt.Errorf("direccion IP vacia")
	}
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("'%s' no es una direccion IPv4 valida", ipStr)
	}
	if ip.IsLoopback() {
		return fmt.Errorf("'%s' es una direccion loopback (127.0.0.0/8)", ipStr)
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("'%s' es una direccion link-local (169.254.0.0/16)", ipStr)
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("'%s' es una direccion no especificada (0.0.0.0)", ipStr)
	}

	cidrStr = strings.TrimSpace(cidrStr)
	if cidrStr != "" {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return fmt.Errorf("CIDR esperado '%s' invalido: %w", cidrStr, err)
		}
		if !ipNet.Contains(ip) {
			return fmt.Errorf("la IP '%s' no pertenece al rango de la red compartida '%s'", ipStr, ipNet.String())
		}
		// Verificar que no sea la direccion de red base
		if ip.Equal(ipNet.IP) {
			return fmt.Errorf("la IP '%s' coincide con la direccion de red base de '%s'", ipStr, ipNet.String())
		}
	}
	return nil
}

// DiscoverVMNetworkByMAC identifica las interfaces de red dentro de Linux mediante sus direcciones MAC,
// activa DHCP y ajusta rutas si la interfaz de la red compartida aun no tiene IP, y valida el rango CIDR.
func DiscoverVMNetworkByMAC(ctx context.Context, vmCfg Config, mac1, mac2, netName, expectedCIDR string, log *slog.Logger) (*DiscoveredVMNetwork, error) {
	normMAC1 := NormalizeMAC(mac1)
	normMAC2 := NormalizeMAC(mac2)
	if normMAC1 == "" {
		return nil, fmt.Errorf("la direccion MAC del adaptador 1 (NAT Network) es obligatoria para el descubrimiento de red")
	}
	if strings.TrimSpace(netName) == "" {
		netName = DefaultNATNetworkName
	}
	if strings.TrimSpace(expectedCIDR) == "" {
		expectedCIDR = DefaultNATNetworkCIDR
	}

	queryInterfaces := func() ([]GuestInterfaceInfo, error) {
		out, err := runHostCmdContext(ctx, vmCfg, `ip -j addr`, log)
		if err == nil && strings.TrimSpace(out) != "" {
			if parsed, jErr := ParseGuestInterfacesJSON(out); jErr == nil && len(parsed) > 0 {
				return parsed, nil
			}
		}
		linkOut, lErr := runHostCmdContext(ctx, vmCfg, `ip -o link`, log)
		if lErr != nil {
			if err != nil {
				return nil, fmt.Errorf("consultando interfaces de red por SSH: %w", err)
			}
			return nil, fmt.Errorf("consultando enlaces de red por SSH: %w", lErr)
		}
		addrOut, _ := runHostCmdContext(ctx, vmCfg, `ip -o -4 addr`, log)
		parsed := ParseGuestInterfacesFallback(linkOut, addrOut)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("no se pudieron obtener interfaces de red desde la VM")
		}
		return parsed, nil
	}

	ifaces, err := queryInterfaces()
	if err != nil {
		return nil, err
	}

	matchAndExtract := func(list []GuestInterfaceInfo) (if1 *GuestInterfaceInfo, ip1 string, if2 *GuestInterfaceInfo, ip2 string) {
		for i := range list {
			mac := NormalizeMAC(list[i].Address)
			if mac != "" && strings.EqualFold(mac, normMAC1) {
				if1 = &list[i]
				ip1 = selectBestIPv4(list[i].AddrInfo, expectedCIDR)
			} else if normMAC2 != "" && mac != "" && strings.EqualFold(mac, normMAC2) {
				if2 = &list[i]
				ip2 = selectBestIPv4(list[i].AddrInfo, "")
			}
		}
		return
	}

	iface1, ip1, iface2, ip2 := matchAndExtract(ifaces)
	if iface1 == nil {
		return nil, fmt.Errorf("no se encontro ninguna interfaz en Linux con la direccion MAC '%s' correspondiente al Adaptador 1 (%s)", normMAC1, netName)
	}

	// Si la interfaz 1 no tiene IP valida dentro del CIDR de vessel-net, activar la interfaz, solicitar DHCP y ajustar metricas de ruta
	if ValidatePrivateIPInCIDR(ip1, expectedCIDR) != nil {
		log.Info("interfaz de red compartida sin IP valida en CIDR; solicitando concesion DHCP en Linux",
			"interface1", iface1.IfName, "mac1", normMAC1, "currentIP", ip1, "expectedCIDR", expectedCIDR)

		if2Name := ""
		if iface2 != nil {
			if2Name = iface2.IfName
		}
		remedyCmd := buildLinuxDualNICRemedyScript(iface1.IfName, if2Name, expectedCIDR)
		if _, rErr := runHostCmdContext(ctx, vmCfg, remedyCmd, log); rErr != nil {
			log.Warn("aviso ejecutando configuracion DHCP en la VM", "interface1", iface1.IfName, "error", rErr)
		}

		// Volver a consultar las interfaces tras solicitar DHCP
		if updatedIfaces, qErr := queryInterfaces(); qErr == nil {
			iface1, ip1, iface2, ip2 = matchAndExtract(updatedIfaces)
		}
	} else if iface2 != nil {
		// Asegurar que las rutas por defecto no entren en conflicto manteniendo conectividad SSH en NIC 2
		remedyRoutes := buildRouteMetricAdjustmentScript(iface1.IfName, iface2.IfName)
		_, _ = runHostCmdContext(ctx, vmCfg, remedyRoutes, log)
	}

	state1 := resolveInterfaceState(iface1, ip1)
	state2 := "unknown"
	if2Name := ""
	if iface2 != nil {
		if2Name = iface2.IfName
		state2 = resolveInterfaceState(iface2, ip2)
	}

	result := &DiscoveredVMNetwork{
		PrivateIP:     ip1,
		NetworkName:   netName,
		NetworkCIDR:   expectedCIDR,
		MACAddress1:   normMAC1,
		MACAddress2:   normMAC2,
		Interface1:    iface1.IfName,
		Interface2:    if2Name,
		Adapter1State: state1,
		Adapter2State: state2,
		Adapter2IP:    ip2,
		DiscoveredAt:  time.Now().Format(time.RFC3339),
	}

	if err := ValidatePrivateIPInCIDR(ip1, expectedCIDR); err != nil {
		result.PrivateIP = ""
		return result, fmt.Errorf("la interfaz '%s' (MAC %s) no obtuvo una IP valida en '%s': %w", iface1.IfName, normMAC1, expectedCIDR, err)
	}

	return result, nil
}

func selectBestIPv4(addrs []GuestAddrEntry, preferredCIDR string) string {
	var preferredNet *net.IPNet
	if strings.TrimSpace(preferredCIDR) != "" {
		_, preferredNet, _ = net.ParseCIDR(strings.TrimSpace(preferredCIDR))
	}

	var fallbackIP string
	for _, a := range addrs {
		if a.Family != "" && a.Family != "inet" {
			continue
		}
		ip := net.ParseIP(strings.TrimSpace(a.Local))
		if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		if preferredNet != nil && preferredNet.Contains(ip) {
			return ip.String()
		}
		if fallbackIP == "" {
			fallbackIP = ip.String()
		}
	}
	return fallbackIP
}

func resolveInterfaceState(iface *GuestInterfaceInfo, ip string) string {
	if iface == nil {
		return "missing"
	}
	oper := strings.ToLower(strings.TrimSpace(iface.OperState))
	hasUpFlag := false
	for _, f := range iface.Flags {
		if strings.EqualFold(strings.TrimSpace(f), "UP") || strings.EqualFold(strings.TrimSpace(f), "LOWER_UP") {
			hasUpFlag = true
			break
		}
	}
	if oper == "up" || (oper == "unknown" && hasUpFlag) || (hasUpFlag && ip != "") {
		if ip != "" {
			return "connected"
		}
		return "up (no-ip)"
	}
	return "down"
}

func buildLinuxDualNICRemedyScript(if1, if2, expectedCIDR string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "sudo ip link set %s up 2>/dev/null || true; ", if1)
	if if2 != "" {
		fmt.Fprintf(&b, "sudo ip link set %s up 2>/dev/null || true; ", if2)
	}
	// Solicitar DHCP en la interfaz 1 (NAT Network)
	fmt.Fprintf(&b, "(sudo dhclient %s 2>/dev/null || sudo udhcpc -i %s -n -q 2>/dev/null || sudo dhcpcd %s 2>/dev/null || sudo networkctl reconfigure %s 2>/dev/null || true); ",
		if1, if1, if1, if1)
	if if2 != "" {
		fmt.Fprintf(&b, "if ! ip -4 addr show dev %s | grep -q 'inet '; then (sudo dhclient %s 2>/dev/null || sudo udhcpc -i %s -n -q 2>/dev/null || sudo networkctl reconfigure %s 2>/dev/null || true); fi; ",
			if2, if2, if2, if2)
		b.WriteString(buildRouteMetricAdjustmentScript(if1, if2))
	}
	return b.String()
}

func buildRouteMetricAdjustmentScript(if1, if2 string) string {
	if if1 == "" || if2 == "" {
		return "true"
	}
	// Garantiza que si ambos adaptadores inyectaron una ruta default sin metrica (o igual metrica),
	// la interfaz 1 (vessel-net) sea la ruta principal (metric 100) y la interfaz 2 (NAT SSH) tenga metric 200,
	// conservando la ruta de subred conectada de NIC 2 (10.0.3.0/24) por donde ingresa el port-forwarding SSH desde 10.0.3.2.
	return fmt.Sprintf(
		`GW1=$(ip -4 route show dev %s | awk '/default via/ {print $3; exit}'); `+
			`GW2=$(ip -4 route show dev %s | awk '/default via/ {print $3; exit}'); `+
			`if [ -n "$GW1" ] && [ -n "$GW2" ]; then `+
			`sudo ip route replace default via "$GW1" dev %s metric 100 2>/dev/null || true; `+
			`sudo ip route replace default via "$GW2" dev %s metric 200 2>/dev/null || true; `+
			`fi`,
		if1, if2, if1, if2,
	)
}

// NeedsDualNICMigration determina si una VM existente en VirtualBox aun tiene la topologia antigua
// (un solo adaptador NAT con --natpf1) y requiere migracion a doble adaptador (NIC1 NAT Network + NIC2 NAT SSH).
func NeedsDualNICMigration(info VMInfo, expectedNetName string) bool {
	if strings.TrimSpace(expectedNetName) == "" {
		expectedNetName = DefaultNATNetworkName
	}
	nic1Ok := strings.EqualFold(info.NIC1, "natnetwork") && strings.EqualFold(info.NATNet1, expectedNetName)
	nic2Ok := strings.EqualFold(info.NIC2, "nat")
	return !nic1Ok || !nic2Ok
}

// MigrateVMToDualNIC migra de forma segura una VM existente (que debe estar apagada)
// desde la topologia antigua (NIC 1 NAT individual con --natpf1) hacia la nueva arquitectura
// (NIC 1 NAT Network compartida + NIC 2 NAT individual con --natpf2 para SSH).
func MigrateVMToDualNIC(ctx context.Context, cfg Config, vmName, netName, cidr string, hostPort, guestPort int, log *slog.Logger) (bool, error) {
	vmName = strings.TrimSpace(vmName)
	if vmName == "" {
		return false, fmt.Errorf("nombre de VM obligatorio para migracion de red")
	}
	if strings.TrimSpace(netName) == "" {
		netName = DefaultNATNetworkName
	}
	if strings.TrimSpace(cidr) == "" {
		cidr = DefaultNATNetworkCIDR
	}
	if guestPort <= 0 {
		guestPort = 22
	}

	info, err := ShowVMInfo(cfg, vmName, log)
	if err != nil {
		return false, fmt.Errorf("consultando estado de VM '%s' para migracion de red: %w", vmName, err)
	}

	if !NeedsDualNICMigration(info, netName) {
		return false, nil
	}

	statusLower := strings.ToLower(strings.TrimSpace(info.Status))
	if statusLower == "running" {
		return false, fmt.Errorf("la VM '%s' esta encendida; debe apagarse antes de migrar sus adaptadores de red a '%s'", vmName, netName)
	}

	if _, err := EnsureNATNetwork(ctx, cfg, netName, cidr, log); err != nil {
		return false, fmt.Errorf("asegurando red compartida '%s' durante migracion de '%s': %w", netName, vmName, err)
	}

	log.Info("migrando VM existente a topologia de doble adaptador (NIC1 NAT Network + NIC2 NAT SSH)",
		"vm", vmName, "netName", netName, "hostPort", hostPort)

	// 1. Limpiar reglas previas de port forwarding "ssh" en NIC 1 y NIC 2 (ignorar error si no existían)
	_, _ = runHostCmdContext(ctx, cfg, fmt.Sprintf(`VBoxManage modifyvm "%s" --natpf1 delete "ssh"`, escapeForDoubleQuote(vmName)), log)
	_, _ = runHostCmdContext(ctx, cfg, fmt.Sprintf(`VBoxManage modifyvm "%s" --natpf2 delete "ssh"`, escapeForDoubleQuote(vmName)), log)

	// 2. Configurar NIC 1 en NAT Network compartida y NIC 2 en NAT individual con --natpf2
	modifyCmd := fmt.Sprintf(
		`VBoxManage modifyvm "%s" --nic1 natnetwork --nat-network1 "%s" --cable-connected1 on --nic2 nat --cable-connected2 on --natpf2 "ssh,tcp,,%d,,%d"`,
		escapeForDoubleQuote(vmName), escapeForDoubleQuote(netName), hostPort, guestPort,
	)
	if _, err := runHostCmdContext(ctx, cfg, modifyCmd, log); err != nil {
		modifyCmdFallback := fmt.Sprintf(
			`VBoxManage modifyvm "%s" --nic1 natnetwork --nat-network1 "%s" --cableconnected1 on --nic2 nat --cableconnected2 on --natpf2 "ssh,tcp,,%d,,%d"`,
			escapeForDoubleQuote(vmName), escapeForDoubleQuote(netName), hostPort, guestPort,
		)
		if _, fErr := runHostCmdContext(ctx, cfg, modifyCmdFallback, log); fErr != nil {
			return false, fmt.Errorf("migrando adaptadores de red de VM '%s': %w", vmName, err)
		}
	}

	return true, nil
}

// VerifyInterVMConnectivity comprueba desde una VM origen si otra VM destino es alcanzable dentro de la red compartida.
// Ejecuta primero prueba ICMP (ping) y complementa con verificacion de puerto TCP (por defecto 22) en caso de que ICMP este bloqueado.
func VerifyInterVMConnectivity(ctx context.Context, sourceVMCfg Config, sourceVM, targetVM, targetIP string, targetPort int, log *slog.Logger) (*VMConnectivityResult, error) {
	targetIP = strings.TrimSpace(targetIP)
	ip := net.ParseIP(targetIP)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("direccion IP destino invalida: '%s'", targetIP)
	}
	if targetPort <= 0 {
		targetPort = 22
	}

	res := &VMConnectivityResult{
		SourceVM:   sourceVM,
		TargetVM:   targetVM,
		TargetIP:   targetIP,
		TargetPort: targetPort,
		CheckedAt:  time.Now().Format(time.RFC3339),
	}

	var outBuf bytes.Buffer

	// 1. Prueba ICMP (ping)
	pingCmd := fmt.Sprintf(`ping -c 3 -W 2 %s`, targetIP)
	pingOut, pingErr := runHostCmdContext(ctx, sourceVMCfg, pingCmd, log)
	if strings.TrimSpace(pingOut) != "" {
		outBuf.WriteString(strings.TrimSpace(pingOut))
	}
	if pingErr == nil && !strings.Contains(strings.ToLower(pingOut), "100% packet loss") {
		res.ICMPReachable = true
		res.LatencyMs = parsePingAvgLatencyMs(pingOut)
	}

	// 2. Prueba TCP (puerto SSH o servicio en destino) como verificacion adicional o fallback si ICMP esta filtrado
	tcpCmd := fmt.Sprintf(`timeout 3 bash -c "</dev/tcp/%s/%d" 2>/dev/null && echo "VESSEL_TCP_OK" || (nc -z -w 3 %s %d 2>/dev/null && echo "VESSEL_TCP_OK")`,
		targetIP, targetPort, targetIP, targetPort)
	tcpOut, tcpErr := runHostCmdContext(ctx, sourceVMCfg, tcpCmd, log)
	if tcpErr == nil && strings.Contains(tcpOut, "VESSEL_TCP_OK") {
		res.TCPReachable = true
	}

	res.Reachable = res.ICMPReachable || res.TCPReachable
	switch {
	case res.ICMPReachable && res.TCPReachable:
		res.Method = fmt.Sprintf("ICMP + TCP:%d", targetPort)
	case res.ICMPReachable:
		res.Method = "ICMP (ping)"
	case res.TCPReachable:
		res.Method = fmt.Sprintf("TCP:%d (ICMP bloqueado/sin respuesta)", targetPort)
	default:
		res.Method = "none"
		if pingErr != nil {
			res.Error = fmt.Sprintf("sin respuesta ICMP ni TCP:%d hacia %s (%v)", targetPort, targetIP, pingErr)
		} else {
			res.Error = fmt.Sprintf("sin respuesta ICMP ni TCP:%d hacia %s", targetPort, targetIP)
		}
	}

	res.Output = outBuf.String()
	return res, nil
}

func parsePingAvgLatencyMs(pingOutput string) float64 {
	// Formato estandar iputils-ping: rtt min/avg/max/mdev = 0.312/0.456/0.612/0.098 ms
	re := regexp.MustCompile(`=\s*[0-9.]+/([0-9.]+)/[0-9.]+/[0-9.]+\s*ms`)
	if m := re.FindStringSubmatch(pingOutput); len(m) == 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			return v
		}
	}
	return 0
}
