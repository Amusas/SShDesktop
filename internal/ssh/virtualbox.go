package ssh

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// HostCommandRunner define la funcion de ejecucion de comandos en el host remoto (inyectable en tests)
type HostCommandRunner func(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error)

var vboxCommandRunner HostCommandRunner = ExecuteCommandContext

// SetVBoxCommandRunner permite reemplazar temporalmente el ejecutor de comandos en pruebas unitarias
func SetVBoxCommandRunner(runner HostCommandRunner) func() {
	prev := vboxCommandRunner
	if runner == nil {
		vboxCommandRunner = ExecuteCommandContext
	} else {
		vboxCommandRunner = runner
	}
	return func() {
		vboxCommandRunner = prev
	}
}

func runHostCmd(cfg Config, command string, log *slog.Logger) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return vboxCommandRunner(ctx, cfg, command, log)
}

func runHostCmdContext(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
	return vboxCommandRunner(ctx, cfg, command, log)
}

// VMInfo contiene los datos en vivo consultados desde VBoxManage
type VMInfo struct {
	Name            string            `json:"name"`
	OS              string            `json:"os"`
	Status          string            `json:"state"`
	CPU             int               `json:"cores"`
	MemoryMB        int               `json:"memoryMB"`
	DiskGB          int               `json:"diskGB"`
	Network         string            `json:"network"`
	IP              string            `json:"ip,omitempty"`
	Firmware        string            `json:"firmware,omitempty"`
	DiskPath        string            `json:"diskPath,omitempty"`
	CreatedAt       string            `json:"createdAt,omitempty"`
	NIC1            string            `json:"nic1,omitempty"`
	NATNet1         string            `json:"natNet1,omitempty"`
	MACAddress1     string            `json:"macAddress1,omitempty"`
	CableConnected1 string            `json:"cableConnected1,omitempty"`
	NIC2            string            `json:"nic2,omitempty"`
	MACAddress2     string            `json:"macAddress2,omitempty"`
	CableConnected2 string            `json:"cableConnected2,omitempty"`
	ForwardingRules map[string]string `json:"forwardingRules,omitempty"`
}

// VMTemplateCreateRequest parametros para crear una VM a partir de plantilla
type VMTemplateCreateRequest struct {
	Name           string `json:"name"`
	TemplateID     string `json:"templateId"`
	TemplateName   string `json:"templateName,omitempty"`
	OSType         string `json:"osType"`
	BaseDiskPath   string `json:"baseDiskPath"`   // Ruta completa al base.vdi en el host remoto
	VMRoot         string `json:"vmRoot,omitempty"` // Ruta base para almacenar VMs (ej: C:\VMManager\VMs)
	Cores          int    `json:"cores"`
	MemoryMB       int    `json:"memoryMB"`
	HostPort       int    `json:"hostPort"` // Puerto NAT asignado (ej: 22001)
	GuestPort      int    `json:"guestPort"`
	NATNetworkName string `json:"natNetworkName,omitempty"` // Red NAT compartida (ej: vessel-net)
	NATNetworkCIDR string `json:"natNetworkCidr,omitempty"` // CIDR de red compartida (ej: 10.250.0.0/24)
}

// ListVMs lista todas las maquinas registradas en VirtualBox y determina su estado (running / poweroff)
func ListVMs(cfg Config, log *slog.Logger) ([]VMInfo, error) {
	out, err := runHostCmd(cfg, `VBoxManage list vms`, log)
	if err != nil {
		return nil, err
	}

	runningOut, _ := runHostCmd(cfg, `VBoxManage list runningvms`, log)
	runningMap := make(map[string]bool)
	for _, line := range strings.Split(runningOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := regexp.MustCompile(`^"([^"]+)"\s*\{`).FindStringSubmatch(line)
		if len(m) == 2 {
			runningMap[strings.ToLower(m[1])] = true
		}
	}

	vms := []VMInfo{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := regexp.MustCompile(`^"([^"]+)"\s*\{`).FindStringSubmatch(line)
		if len(m) == 2 {
			name := m[1]
			status := "poweroff"
			if runningMap[strings.ToLower(name)] {
				status = "running"
			}
			vms = append(vms, VMInfo{Name: name, Status: status})
		}
	}
	return vms, nil
}

// ShowVMInfo consulta informacion detallada y estado de la maquina virtual mediante --machinereadable
func ShowVMInfo(cfg Config, name string, log *slog.Logger) (VMInfo, error) {
	cmd := fmt.Sprintf(`VBoxManage showvminfo "%s" --machinereadable`, escapeForDoubleQuote(name))
	out, err := runHostCmd(cfg, cmd, log)
	if err != nil {
		return VMInfo{}, err
	}

	return ParseShowVMInfo(name, out), nil
}

// ParseShowVMInfo parsea la salida --machinereadable de VBoxManage showvminfo
func ParseShowVMInfo(defaultName, out string) VMInfo {
	info := VMInfo{
		Name:            defaultName,
		ForwardingRules: make(map[string]string),
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, rawVal, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val := strings.Trim(strings.TrimSpace(rawVal), `"`)

		switch key {
		case "VMState":
			info.Status = val
		case "name":
			info.Name = val
		case "ostype":
			info.OS = val
		case "cpus":
			if n, err := strconv.Atoi(val); err == nil {
				info.CPU = n
			}
		case "memory":
			if n, err := strconv.Atoi(val); err == nil {
				info.MemoryMB = n
			}
		case "nic1":
			info.NIC1 = val
		case "natnet1", "nat-network1":
			info.NATNet1 = val
		case "macaddress1":
			info.MACAddress1 = NormalizeMAC(val)
		case "cableconnected1":
			info.CableConnected1 = val
		case "nic2":
			info.NIC2 = val
		case "macaddress2":
			info.MACAddress2 = NormalizeMAC(val)
		case "cableconnected2":
			info.CableConnected2 = val
		default:
			if strings.HasPrefix(key, "Forwarding(") {
				info.ForwardingRules[key] = val
			}
		}
	}

	if strings.EqualFold(info.NIC1, "natnetwork") && info.NATNet1 != "" {
		info.Network = info.NATNet1
	} else if info.NIC1 != "" && !strings.EqualFold(info.NIC1, "none") {
		info.Network = info.NIC1
	} else if info.NIC2 != "" && !strings.EqualFold(info.NIC2, "none") {
		info.Network = info.NIC2
	}
	return info
}

// IsVMRunning verifica si la VM esta en estado "running"
func IsVMRunning(cfg Config, name string, log *slog.Logger) (bool, error) {
	out, err := runHostCmd(cfg, `VBoxManage list runningvms`, log)
	if err != nil {
		return false, err
	}
	quotedName := `"` + name + `"`
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, quotedName+" ") || line == quotedName {
			return true, nil
		}
	}
	return false, nil
}

// CreateVMFromTemplate crea una VM basada en plantilla conectando su disco en modo multiattach
func CreateVMFromTemplate(ctx context.Context, cfg Config, req VMTemplateCreateRequest, log *slog.Logger) error {
	name := sanitizeVMName(req.Name)
	if name == "" || !regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`).MatchString(name) {
		return fmt.Errorf("nombre de VM invalido: '%s'", req.Name)
	}
	if req.Cores < 1 || req.Cores > 64 {
		return fmt.Errorf("cores de CPU debe estar entre 1 y 64")
	}
	if req.MemoryMB < 256 || req.MemoryMB > 131072 {
		return fmt.Errorf("memoria RAM debe estar entre 256 MB y 128 GB")
	}
	if req.HostPort <= 0 {
		return fmt.Errorf("puerto host forwarding invalido: %d", req.HostPort)
	}
	if strings.TrimSpace(req.BaseDiskPath) == "" {
		return fmt.Errorf("ruta de disco base es obligatoria")
	}

	// 1. Verificar si ya existe en VirtualBox
	if out, err := runHostCmdContext(ctx, cfg, `VBoxManage list vms`, log); err == nil {
		if strings.Contains(out, `"`+name+`"`) {
			return fmt.Errorf("la maquina virtual '%s' ya existe en VirtualBox", name)
		}
	}

	natNetName := strings.TrimSpace(req.NATNetworkName)
	if natNetName == "" {
		natNetName = DefaultNATNetworkName
	}
	natNetCIDR := strings.TrimSpace(req.NATNetworkCIDR)
	if natNetCIDR == "" {
		natNetCIDR = DefaultNATNetworkCIDR
	}

	// 2. Garantizar que la red compartida NAT Network exista, sea valida y este activa
	if _, err := EnsureNATNetwork(ctx, cfg, natNetName, natNetCIDR, log); err != nil {
		return fmt.Errorf("preparando red compartida '%s' (%s): %w", natNetName, natNetCIDR, err)
	}

	guestPort := req.GuestPort
	if guestPort <= 0 {
		guestPort = 22
	}

	osType := strings.TrimSpace(req.OSType)
	if osType == "" {
		osType = "Debian_64"
	}

	// 3. Crear VM en VirtualBox
	createCmd := fmt.Sprintf(`VBoxManage createvm --name "%s" --ostype "%s" --register`, escapeForDoubleQuote(name), escapeForDoubleQuote(osType))
	if _, err := runHostCmdContext(ctx, cfg, createCmd, log); err != nil {
		log.Error("fallo comando VBoxManage createvm", "error", err)
		return fmt.Errorf("ejecutando createvm '%s': %w", name, err)
	}

	// 4. Crear controlador SATA
	storageCtlCmd := fmt.Sprintf(`VBoxManage storagectl "%s" --name "SATA" --add sata --controller IntelAhci`, escapeForDoubleQuote(name))
	if _, err := runHostCmdContext(ctx, cfg, storageCtlCmd, log); err != nil {
		log.Error("fallo comando storagectl, ejecutando rollback", "error", err)
		_ = DeleteVM(cfg, name, log)
		return fmt.Errorf("ejecutando storagectl '%s': %w", name, err)
	}

	// 5. Conectar disco base a la controladora SATA
	attachCmdWithMtype := fmt.Sprintf(`VBoxManage storageattach "%s" --storagectl "SATA" --port 0 --device 0 --type hdd --medium "%s" --mtype multiattach`, escapeForDoubleQuote(name), escapeForDoubleQuote(req.BaseDiskPath))
	attachCmdWithoutMtype := fmt.Sprintf(`VBoxManage storageattach "%s" --storagectl "SATA" --port 0 --device 0 --type hdd --medium "%s"`, escapeForDoubleQuote(name), escapeForDoubleQuote(req.BaseDiskPath))

	var attachErr error
	if _, err := runHostCmdContext(ctx, cfg, attachCmdWithMtype, log); err == nil {
		attachErr = nil
	} else if _, err := runHostCmdContext(ctx, cfg, attachCmdWithoutMtype, log); err == nil {
		attachErr = nil
	} else {
		// VirtualBox CLI en Windows 7.2 puede arrojar el error histórico de estado:
		// "the media type 'MultiAttach' can only be attached to machines that were created with VirtualBox 4.0 or later"
		// cuando un medio ya fue registrado en el contexto de otra definición de VM.
		// En este caso aprovisionamos explícitamente el disco diferencial hijo con createmedium disk --diffparent.
		// Esto garantiza que el disco base de la plantilla permanezca 100% intacto y de sólo lectura,
		// y que la nueva VM tenga su propio medio hijo sin conflictos de registro.
		log.Info("aprovisionando disco diferencial hijo para la VM a partir de plantilla base", "template", req.BaseDiskPath)

		vmFolder := ""
		if infoOut, iErr := runHostCmdContext(ctx, cfg, fmt.Sprintf(`VBoxManage showvminfo "%s" --machinereadable`, escapeForDoubleQuote(name)), log); iErr == nil {
			for _, line := range strings.Split(infoOut, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "CfgFile=") {
					cfgFile := strings.Trim(strings.TrimPrefix(line, "CfgFile="), `"`)
					vmFolder = filepath.Dir(cfgFile)
					break
				}
			}
		}
		if vmFolder == "" {
			vmFolder = filepath.Join(cfg.StartPath, "VirtualBox VMs", name)
		}

		diffDiskPath := filepath.Join(vmFolder, fmt.Sprintf("%s.vdi", name))
		createDiffCmd := fmt.Sprintf(`VBoxManage createmedium disk --filename "%s" --diffparent "%s"`, escapeForDoubleQuote(diffDiskPath), escapeForDoubleQuote(req.BaseDiskPath))
		if _, dErr := runHostCmdContext(ctx, cfg, createDiffCmd, log); dErr == nil {
			attachDiffCmd := fmt.Sprintf(`VBoxManage storageattach "%s" --storagectl "SATA" --port 0 --device 0 --type hdd --medium "%s"`, escapeForDoubleQuote(name), escapeForDoubleQuote(diffDiskPath))
			if _, aErr := runHostCmdContext(ctx, cfg, attachDiffCmd, log); aErr == nil {
				attachErr = nil
				log.Info("disco diferencial aprovisionado y conectado exitosamente", "diffDisk", diffDiskPath)
			} else {
				attachErr = fmt.Errorf("asociando disco diferencial a la VM: %w", aErr)
			}
		} else {
			attachErr = fmt.Errorf("creando disco diferencial (%s): %w", dErr, err)
		}
	}

	if attachErr != nil {
		log.Error("fallo conectando almacenamiento a la VM, ejecutando rollback", "error", attachErr)
		_ = DeleteVM(cfg, name, log)
		return fmt.Errorf("conectando almacenamiento a la VM '%s': %w", name, attachErr)
	}

	// 6. Configurar hardware y topologia de doble adaptador:
	// - Adaptador 1: NAT Network compartida (vessel-net) para comunicacion entre VMs y salida externa
	// - Adaptador 2: NAT individual con port forwarding SSH (--natpf2) para administracion desde Vessel
	modifyCmd := fmt.Sprintf(
		`VBoxManage modifyvm "%s" --memory %d --cpus %d --nic1 natnetwork --nat-network1 "%s" --cable-connected1 on --nic2 nat --cable-connected2 on --natpf2 "ssh,tcp,,%d,,%d"`,
		escapeForDoubleQuote(name), req.MemoryMB, req.Cores, escapeForDoubleQuote(natNetName), req.HostPort, guestPort,
	)
	if _, err := runHostCmdContext(ctx, cfg, modifyCmd, log); err != nil {
		// Fallback compatible con sintaxis sin guion (--cableconnected1 / --cableconnected2)
		modifyCmdFallback := fmt.Sprintf(
			`VBoxManage modifyvm "%s" --memory %d --cpus %d --nic1 natnetwork --nat-network1 "%s" --cableconnected1 on --nic2 nat --cableconnected2 on --natpf2 "ssh,tcp,,%d,,%d"`,
			escapeForDoubleQuote(name), req.MemoryMB, req.Cores, escapeForDoubleQuote(natNetName), req.HostPort, guestPort,
		)
		if _, fErr := runHostCmdContext(ctx, cfg, modifyCmdFallback, log); fErr != nil {
			log.Error("fallo comando modifyvm, ejecutando rollback", "error", err)
			_ = DeleteVM(cfg, name, log)
			return fmt.Errorf("ejecutando modifyvm '%s': %w", name, err)
		}
	}

	// 7. Configurar orden de arranque
	_, _ = runHostCmdContext(ctx, cfg, fmt.Sprintf(`VBoxManage modifyvm "%s" --boot1 disk --boot2 none --boot3 none --boot4 none`, escapeForDoubleQuote(name)), log)

	return nil
}

// StartVM inicia la maquina virtual en modo headless
func StartVM(cfg Config, name string, log *slog.Logger) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("nombre de VM obligatorio")
	}
	if running, err := IsVMRunning(cfg, name, log); err == nil && running {
		return nil
	}

	cmd := fmt.Sprintf(`VBoxManage startvm "%s" --type headless`, escapeForDoubleQuote(name))
	_, err := runHostCmd(cfg, cmd, log)
	if err != nil {
		return fmt.Errorf("iniciando VM '%s': %w", name, err)
	}
	return nil
}

// BootstrapVMHostKeys ejecuta los comandos para generar las llaves de host (ssh-keygen -A),
// levantar todas las interfaces de red (NIC 1 NAT Network y NIC 2 NAT SSH) con DHCP,
// y reiniciar el servicio sshd en la VM sin requerir conexion SSH previa.
// Intenta primero mediante 'VBoxManage guestcontrol' (VirtualBox Guest Additions) y como fallback
// inyecta la secuencia de teclas en la consola virtual (TTY2).
func BootstrapVMHostKeys(ctx context.Context, cfg Config, vmName, user, password string, log *slog.Logger) error {
	log.Info("inicializando interfaces de red y host keys de SSH en consola de la VM", "vm", vmName, "user", user)

	netInitSnippet := `for d in $(ip -o link show | awk -F': ' '{print $2}' | cut -d'@' -f1 | grep -v '^lo$'); do ip link set "$d" up 2>/dev/null || true; if ! ip -4 addr show dev "$d" | grep -q 'inet '; then (dhclient "$d" || udhcpc -i "$d" -n -q || dhcpcd "$d" || networkctl reconfigure "$d") >/dev/null 2>&1 & fi; done`

	// 1. Intentar primero con VBoxManage guestcontrol (ejecución directa dentro del guest sin depender de la UI)
	guestCmd := fmt.Sprintf(`VBoxManage guestcontrol "%s" run --exe "/bin/sh" --username "%s" --password "%s" -- /bin/sh -c "echo %s | sudo -S sh -c '%s && ssh-keygen -A && systemctl restart ssh'"`,
		escapeForDoubleQuote(vmName), escapeForDoubleQuote(user), escapeForDoubleQuote(password), escapeForDoubleQuote(password), netInitSnippet)
	if out, err := runHostCmdContext(ctx, cfg, guestCmd, log); err == nil {
		log.Info("red y host keys inicializadas via guestcontrol exitosamente", "vm", vmName, "output", out)
		return nil
	}

	// 2. Fallback: secuencia de consola combinada en una sola llamada remota para rapidez
	cmdStr := fmt.Sprintf("echo %s | sudo -S sh -c '%s && ssh-keygen -A && systemctl restart ssh'", password, netInitSnippet)
	cmdChain := fmt.Sprintf(
		`VBoxManage controlvm "%s" keyboardputscancode 1c 9c & ping -n 2 127.0.0.1 >nul & `+
			`VBoxManage controlvm "%s" keyboardputscancode 1d 38 3c bc b8 9d & ping -n 2 127.0.0.1 >nul & `+
			`VBoxManage controlvm "%s" keyboardputstring "%s" & VBoxManage controlvm "%s" keyboardputscancode 1c 9c & ping -n 2 127.0.0.1 >nul & `+
			`VBoxManage controlvm "%s" keyboardputstring "%s" & VBoxManage controlvm "%s" keyboardputscancode 1c 9c & ping -n 3 127.0.0.1 >nul & `+
			`VBoxManage controlvm "%s" keyboardputstring "%s" & VBoxManage controlvm "%s" keyboardputscancode 1c 9c`,
		escapeForDoubleQuote(vmName),
		escapeForDoubleQuote(vmName),
		escapeForDoubleQuote(vmName), escapeForDoubleQuote(user), escapeForDoubleQuote(vmName),
		escapeForDoubleQuote(vmName), escapeForDoubleQuote(password), escapeForDoubleQuote(vmName),
		escapeForDoubleQuote(vmName), escapeForDoubleQuote(cmdStr), escapeForDoubleQuote(vmName),
	)

	_, err := runHostCmdContext(ctx, cfg, cmdChain, log)
	return err
}

// StopVM realiza un apagado limpio enviando senal ACPI
func StopVM(cfg Config, name string, log *slog.Logger) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("nombre de VM obligatorio")
	}

	running, err := IsVMRunning(cfg, name, log)
	if err != nil || !running {
		return nil
	}

	cmd := fmt.Sprintf(`VBoxManage controlvm "%s" acpipowerbutton`, escapeForDoubleQuote(name))
	if _, err := runHostCmd(cfg, cmd, log); err != nil {
		return fmt.Errorf("enviando senal ACPI a VM '%s': %w", name, err)
	}

	return WaitForVMState(cfg, name, "poweroff", 30*time.Second, log)
}

// ForcePoweroffVM realiza un apagado forzado inmediato
func ForcePoweroffVM(cfg Config, name string, log *slog.Logger) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("nombre de VM obligatorio")
	}

	cmd := fmt.Sprintf(`VBoxManage controlvm "%s" poweroff`, escapeForDoubleQuote(name))
	if _, err := runHostCmd(cfg, cmd, log); err != nil {
		return fmt.Errorf("apagando forzadamente VM '%s': %w", name, err)
	}

	return WaitForVMState(cfg, name, "poweroff", 15*time.Second, log)
}

// DeleteVM detiene la VM si esta corriendo y la desregistra de VirtualBox sin eliminar discos base de plantillas
// ni eliminar la red compartida NAT Network (vessel-net).
func DeleteVM(cfg Config, name string, log *slog.Logger) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("nombre de VM obligatorio")
	}

	if running, err := IsVMRunning(cfg, name, log); err == nil && running {
		_ = StopVM(cfg, name, log)
		_ = ForcePoweroffVM(cfg, name, log)
	}

	cmd := fmt.Sprintf(`VBoxManage unregistervm "%s" --delete`, escapeForDoubleQuote(name))
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := runHostCmd(cfg, cmd, log); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(1 * time.Second)
	}
	return lastErr
}

// WaitForVMState espera hasta que la VM alcance un estado especifico o se agote el timeout
func WaitForVMState(cfg Config, name, expectedState string, timeout time.Duration, log *slog.Logger) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		info, err := ShowVMInfo(cfg, name, log)
		if err == nil {
			statusLower := strings.ToLower(info.Status)
			if expectedState == "poweroff" && (statusLower == "poweroff" || statusLower == "powered off" || statusLower == "aborted" || statusLower == "") {
				return nil
			}
			if expectedState == "running" && statusLower == "running" {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timeout esperando que VM '%s' alcance estado '%s'", name, expectedState)
}

// WaitForSSH espera activamente hasta que el puerto SSH de la VM este respondiendo.
// Intenta primero mediante tunel SSH a traves del host remoto (127.0.0.1:port) verificando el banner SSH,
// evitando bloqueos por el Firewall de Windows en la red local.
func WaitForSSH(ctx context.Context, hostCfg Config, port int, timeout time.Duration, log *slog.Logger) error {
	deadline := time.Now().Add(timeout)
	localTarget := fmt.Sprintf("127.0.0.1:%d", port)
	directTarget := net.JoinHostPort(hostCfg.Host, fmt.Sprintf("%d", port))

	log.Info("esperando disponibilidad de SSH en VM", "localTarget", localTarget, "directTarget", directTarget)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 1. Intentar mediante tunel SSH con el host Windows
		hClient, err := DialHost(hostCfg)
		if err == nil {
			conn, dErr := hClient.Dial("tcp", localTarget)
			if dErr == nil {
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				buf := make([]byte, 256)
				n, rErr := conn.Read(buf)
				_ = conn.Close()
				hClient.Close()
				if rErr == nil && strings.HasPrefix(string(buf[:n]), "SSH-") {
					log.Info("puerto SSH de la VM activo y respondiendo banner via tunel", "puerto", port, "banner", strings.TrimSpace(string(buf[:n])))
					return nil
				}
				log.Debug("puerto abierto pero SSH daemon aun iniciando en VM...", "puerto", port)
			} else {
				hClient.Close()
			}
		}

		// 2. Fallback: intentar conexion directa al puerto
		if conn, err := net.DialTimeout("tcp", directTarget, 2*time.Second); err == nil {
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 256)
			n, rErr := conn.Read(buf)
			_ = conn.Close()
			if rErr == nil && strings.HasPrefix(string(buf[:n]), "SSH-") {
				log.Info("puerto SSH de la VM activo y respondiendo banner via conexion directa", "direccion", directTarget)
				return nil
			}
		}

		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("tiempo de espera agotado esperando puerto SSH en VM (puerto %d)", port)
}

// ProvisionVMUser conecta a la VM con la clave bootstrap, crea el usuario unico, instala la clave publica,
// actualiza el hostname del sistema operativo al nombre real de la VM, remueve la clave de bootstrap autorizada y configura permisos
func ProvisionVMUser(ctx context.Context, hostCfg Config, port int, bootstrapUser, bootstrapKeyPath, targetUser, targetPublicKey string, log *slog.Logger, vmName ...string) error {
	authMethods, err := buildAuthMethodsFromKeyPath(bootstrapKeyPath)
	if err != nil {
		log.Warn("no se pudo cargar llave bootstrap, se usara contraseña", "error", err)
		authMethods = []gossh.AuthMethod{}
	}
	// Soporte de contrasena para el usuario bootstrap (ej: "admin")
	authMethods = append(authMethods, gossh.Password("admin"))

	clientConfig := &gossh.ClientConfig{
		User:            bootstrapUser,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), // Temporal durante el bootstrap inicial
		Timeout:         10 * time.Second,
	}

	targetAddr := fmt.Sprintf("127.0.0.1:%d", port)
	directAddr := net.JoinHostPort(hostCfg.Host, fmt.Sprintf("%d", port))

	var guestClient *gossh.Client
	var hostClient *gossh.Client

	// Reintentos de conexion SSH con la clave o contrasena bootstrap (tunel o directo)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 1. Intentar mediante tunel SSH al host Windows
		hClient, err := DialHost(hostCfg)
		if err == nil {
			netConn, dialErr := hClient.Dial("tcp", targetAddr)
			if dialErr == nil {
				sshConn, chans, reqs, cErr := gossh.NewClientConn(netConn, targetAddr, clientConfig)
				if cErr == nil {
					guestClient = gossh.NewClient(sshConn, chans, reqs)
					hostClient = hClient
					break
				}
				netConn.Close()
			}
			hClient.Close()
		}

		// 2. Fallback: conexion directa
		gClient, dErr := gossh.Dial("tcp", directAddr, clientConfig)
		if dErr == nil {
			guestClient = gClient
			break
		}

		time.Sleep(2 * time.Second)
	}

	if guestClient == nil {
		return fmt.Errorf("no se pudo autenticar SSH con usuario bootstrap '%s' en el puerto %d (clave: %s o pass: admin)", bootstrapUser, port, bootstrapKeyPath)
	}
	defer guestClient.Close()
	if hostClient != nil {
		defer hostClient.Close()
	}

	escapedPubKey := strings.ReplaceAll(targetPublicKey, "'", `'\''`)
	desiredHost := ""
	if len(vmName) > 0 {
		desiredHost = sanitizeLinuxHostname(vmName[0])
	}

	// Script Linux seguro y determinista:
	// 1. Garantizar privilegios sudo con password si se requiere
	// 2. Actualizar el hostname de la VM para no heredar "plantilla" de la imagen base
	// 3. Crear usuario unico si no existe
	// 4. Dar permisos sudo al nuevo usuario
	// 5. Crear ~/.ssh y autorizar la llave publica generada para esta VM
	// 6. Configurar multi-user.target para arranque rapido de servidor sin GUI pesada
	// 7. Eliminar las llaves y privilegios de bootstrap de la VM
	provisionScript := fmt.Sprintf(`
set -e
TARGET_USER='%s'
PUB_KEY='%s'
BOOTSTRAP_USER='%s'
NEW_HOSTNAME='%s'

# Asegurar permisos de sudo para bootstrap usando contraseña si se requiere
if ! sudo -n true 2>/dev/null; then
    echo "admin" | sudo -S sh -c 'mkdir -p /etc/sudoers.d && echo "'"$BOOTSTRAP_USER"' ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/99-bootstrap && chmod 440 /etc/sudoers.d/99-bootstrap' 2>/dev/null || true
fi

# Actualizar hostname del sistema operativo para reemplazar el hostname heredado de la plantilla
if [ -n "$NEW_HOSTNAME" ]; then
    sudo hostnamectl set-hostname "$NEW_HOSTNAME" 2>/dev/null || sudo hostname "$NEW_HOSTNAME" 2>/dev/null || true
    echo "$NEW_HOSTNAME" | sudo tee /etc/hostname >/dev/null 2>&1 || true
    if [ -f /etc/hosts ]; then
        if grep -q '^127\.0\.1\.1' /etc/hosts 2>/dev/null; then
            sudo sed -i "s/^127\.0\.1\.1.*/127.0.1.1\t$NEW_HOSTNAME/" /etc/hosts 2>/dev/null || true
        else
            printf "127.0.1.1\t%%s\n" "$NEW_HOSTNAME" | sudo tee -a /etc/hosts >/dev/null 2>&1 || true
        fi
    fi
fi

if ! id -u "$TARGET_USER" >/dev/null 2>&1; then
    sudo useradd -m -s /bin/bash "$TARGET_USER" || sudo adduser --disabled-password --gecos "" "$TARGET_USER"
fi

sudo usermod -aG sudo "$TARGET_USER" 2>/dev/null || sudo usermod -aG wheel "$TARGET_USER" 2>/dev/null || true
sudo usermod -aG docker "$TARGET_USER" 2>/dev/null || true

if [ -d /etc/sudoers.d ]; then
    echo "$TARGET_USER ALL=(ALL) NOPASSWD:ALL" | sudo tee "/etc/sudoers.d/99-$TARGET_USER" >/dev/null
    sudo chmod 440 "/etc/sudoers.d/99-$TARGET_USER"
fi

USER_HOME=$(eval echo "~$TARGET_USER")
sudo mkdir -p "$USER_HOME/.ssh"
sudo chmod 700 "$USER_HOME/.ssh"
echo "$PUB_KEY" | sudo tee "$USER_HOME/.ssh/authorized_keys" >/dev/null
sudo chmod 600 "$USER_HOME/.ssh/authorized_keys"
sudo chown -R "$TARGET_USER:$TARGET_USER" "$USER_HOME/.ssh"

# Acelerar arranque cambiando a multi-user.target (modo consola/servidor sin entorno grafico pesado)
sudo systemctl set-default multi-user.target 2>/dev/null || true

# Persistir configuracion DHCP para todas las interfaces Ethernet (NIC 1 NAT Network + NIC 2 NAT SSH)
if [ -f /etc/network/interfaces ]; then
    sudo mkdir -p /etc/network/interfaces.d
    if ! grep -q "interfaces.d" /etc/network/interfaces 2>/dev/null; then
        echo "source /etc/network/interfaces.d/*" | sudo tee -a /etc/network/interfaces >/dev/null
    fi
    sudo rm -f /etc/network/interfaces.d/50-vessel-dual-nic.cfg
    for dev in $(ip -o link show | awk -F': ' '{print $2}' | cut -d'@' -f1 | grep -v '^lo$' | grep -v '^docker' | grep -v '^br-' | grep -v '^veth'); do
        sudo ip link set "$dev" up 2>/dev/null || true
        if ! grep -rq "iface $dev" /etc/network/interfaces /etc/network/interfaces.d/ 2>/dev/null; then
            printf "allow-hotplug %%s\niface %%s inet dhcp\n\n" "$dev" "$dev" | sudo tee -a /etc/network/interfaces.d/50-vessel-dual-nic.cfg >/dev/null
        fi
        if ! ip -4 addr show dev "$dev" | grep -q 'inet '; then
            sudo dhclient "$dev" 2>/dev/null || sudo udhcpc -i "$dev" -n -q 2>/dev/null || sudo networkctl reconfigure "$dev" 2>/dev/null || true
        fi
    done
fi

# Eliminar acceso bootstrap de la VM (solo dejar las llaves generadas de la app)
sudo rm -f "/etc/sudoers.d/99-bootstrap" 2>/dev/null || true
BOOTSTRAP_HOME=$(eval echo "~$BOOTSTRAP_USER")
sudo rm -f "$BOOTSTRAP_HOME/.ssh/authorized_keys" 2>/dev/null || true
sudo passwd -l "$BOOTSTRAP_USER" 2>/dev/null || true

# Reiniciar servicio docker para aplicar grupos (si aplica)
sudo systemctl restart docker 2>/dev/null || true

# Limpiar contenedores residuales heredados de plantillas basadas en snapshots para que la VM inicie limpia
sudo docker rm -f $(sudo docker ps -aq) 2>/dev/null || true
`, targetUser, escapedPubKey, bootstrapUser, desiredHost)

	session, err := guestClient.NewSession()
	if err != nil {
		return fmt.Errorf("creando sesion SSH de provisioning: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(provisionScript); err != nil {
		return fmt.Errorf("ejecutando provisioning de usuario '%s': %w (stderr: %s)", targetUser, err, stderr.String())
	}

	log.Info("usuario unico Linux aprovisionado y acceso bootstrap removido exitosamente", "usuario", targetUser, "puerto", port)
	return nil
}

func sanitizeVMName(name string) string {
	r := regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)
	return strings.TrimSpace(r.ReplaceAllString(name, "-"))
}

func sanitizeLinuxHostname(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	r := regexp.MustCompile(`[^a-z0-9-]+`)
	clean = strings.Trim(r.ReplaceAllString(clean, "-"), "-")
	if len(clean) > 63 {
		clean = strings.TrimRight(clean[:63], "-")
	}
	return clean
}

func escapeForDoubleQuote(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func buildAuthMethodsFromKeyPath(keyPath string) ([]gossh.AuthMethod, error) {
	if keyPath == "" {
		return []gossh.AuthMethod{}, nil
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("leyendo llave privada '%s': %w", keyPath, err)
	}
	signer, err := gossh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parseando llave privada '%s': %w", keyPath, err)
	}
	return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
}
