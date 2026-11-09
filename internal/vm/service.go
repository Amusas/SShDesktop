package vm

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/keys"
	"github.com/feliren/ssh-web/internal/ssh"
	"github.com/feliren/ssh-web/internal/template"
)

// VMCreateTemplateRequest estructura de solicitud para crear una VM basada en plantilla
type VMCreateTemplateRequest struct {
	Name       string `json:"name"`
	TemplateID string `json:"templateId"`
	Cores      int    `json:"cores"`
	MemoryMB   int    `json:"memoryMB"`
}

// VMNetworkInfoResponse describe el estado de red de una VM en Vessel
type VMNetworkInfoResponse struct {
	VMName               string `json:"vmName"`
	State                string `json:"state"`
	PrivateIP            string `json:"privateIp"`
	NetworkName          string `json:"networkName"`
	NetworkCIDR          string `json:"networkCidr"`
	Host                 string `json:"host"`
	HostPort             int    `json:"hostPort"`
	GuestPort            int    `json:"guestPort"`
	SSHUser              string `json:"sshUser"`
	MACAddress1          string `json:"macAddress1"`
	MACAddress2          string `json:"macAddress2"`
	Interface1           string `json:"interface1,omitempty"`
	Interface2           string `json:"interface2,omitempty"`
	Adapter1State        string `json:"adapter1State"`
	Adapter2State        string `json:"adapter2State"`
	LastNetworkDiscovery string `json:"lastNetworkDiscovery,omitempty"`
	DiscoveryError       string `json:"discoveryError,omitempty"`
}

type Service struct {
	cfg            *config.Config
	log            *slog.Logger
	vmStore        *Store
	keyStore       *keys.Store
	tplSvc         *template.Service
	portAlloc      *PortAllocator
	knownHostsMgr  *ssh.KnownHostsManager
	vmLocks        sync.Map // map[string]*sync.Mutex
}

func NewService(cfg *config.Config, log *slog.Logger) *Service {
	vmPath := filepath.Join(".", "vms.json")
	keyPath := filepath.Join(".", "keys.json")
	vmStore := NewStore(vmPath)
	knownHostsPath := filepath.Join(cfg.SSHRoot, "known_hosts")

	return &Service{
		cfg:           cfg,
		log:           log,
		vmStore:       vmStore,
		keyStore:      keys.NewStore(keyPath),
		tplSvc:        template.NewService(cfg, log),
		portAlloc:     NewPortAllocator(cfg.BasePort, vmStore),
		knownHostsMgr: ssh.NewKnownHostsManager(knownHostsPath),
	}
}

func (s *Service) getVMLock(name string) *sync.Mutex {
	lock, _ := s.vmLocks.LoadOrStore(strings.ToLower(name), &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (s *Service) KeyStore() *keys.Store {
	return s.keyStore
}

func (s *Service) KnownHostsManager() *ssh.KnownHostsManager {
	return s.knownHostsMgr
}

func (s *Service) SSHConfig() ssh.Config {
	return ssh.Config{
		Host:      s.cfg.SSHHost,
		Port:      s.cfg.SSHPort,
		User:      s.cfg.SSHUser,
		KeyPath:   s.cfg.SSHKeyPath,
		Shell:     s.cfg.SSHShell,
		StartPath: s.cfg.SSHStartPath,
	}
}

func (s *Service) NATNetworkName() string {
	if s.cfg != nil && strings.TrimSpace(s.cfg.NATNetworkName) != "" {
		return strings.TrimSpace(s.cfg.NATNetworkName)
	}
	return ssh.DefaultNATNetworkName
}

func (s *Service) NATNetworkCIDR() string {
	if s.cfg != nil && strings.TrimSpace(s.cfg.NATNetworkCIDR) != "" {
		return strings.TrimSpace(s.cfg.NATNetworkCIDR)
	}
	return ssh.DefaultNATNetworkCIDR
}

// VMSSHConfig construye la configuracion SSH administrativa para conectar al guest Linux
// a traves del segundo adaptador NAT (ProxyHost -> 127.0.0.1:HostPort).
func (s *Service) VMSSHConfig(rec VMRecord) ssh.Config {
	hostCfg := s.SSHConfig()
	keyPath := rec.PrivateKeyPath
	if keyPath == "" && s.cfg != nil && s.cfg.SSHRoot != "" {
		keyPath = filepath.Join(s.cfg.SSHRoot, "vms", rec.Name, "id_ed25519")
	}
	if keyPath != "" {
		if _, statErr := os.Stat(keyPath); os.IsNotExist(statErr) && s.keyStore != nil {
			if kRec, kErr := s.keyStore.Get(rec.Name); kErr == nil && strings.TrimSpace(kRec.PrivateKey) != "" {
				_ = os.MkdirAll(filepath.Dir(keyPath), 0700)
				_ = os.WriteFile(keyPath, []byte(kRec.PrivateKey), 0600)
				if kRec.PublicKey != "" {
					_ = os.WriteFile(keyPath+".pub", []byte(kRec.PublicKey), 0644)
				}
			}
		}
	}
	user := rec.SSHUser
	if user == "" {
		user = "bootstrap"
	}
	return ssh.Config{
		Host:      "127.0.0.1",
		Port:      rec.HostPort,
		User:      user,
		KeyPath:   keyPath,
		ProxyHost: &hostCfg,
	}
}

func (s *Service) normalizeRecordNetworkDefaults(rec *VMRecord) {
	if rec.PrivateIP == "" && rec.IP != "" {
		rec.PrivateIP = rec.IP
	}
	if rec.IP == "" && rec.PrivateIP != "" {
		rec.IP = rec.PrivateIP
	}
	if rec.NetworkName == "" {
		if rec.Network != "" && !strings.EqualFold(rec.Network, "NAT") && !strings.EqualFold(rec.Network, "natnetwork") {
			rec.NetworkName = rec.Network
		} else {
			rec.NetworkName = s.NATNetworkName()
		}
	}
	if rec.Network == "" || strings.EqualFold(rec.Network, "NAT") || strings.EqualFold(rec.Network, "natnetwork") {
		rec.Network = rec.NetworkName
	}
	if rec.NetworkCIDR == "" {
		rec.NetworkCIDR = s.NATNetworkCIDR()
	}
	if strings.EqualFold(rec.State, "poweroff") {
		rec.Adapter1State = "disconnected"
		rec.Adapter2State = "disconnected"
	}
}

func (s *Service) List() ([]VMRecord, error) {
	records, err := s.vmStore.Load()
	if err != nil {
		s.log.Warn("error leyendo vms.json", "error", err)
	}

	remoteVMs, rErr := ssh.ListVMs(s.SSHConfig(), s.log)
	if rErr != nil {
		// Si no hay comunicacion remota, retornar registros locales
		return records, nil
	}

	// Mapa de VMs remotas existentes en VirtualBox: nombre en minusculas -> VMInfo
	remoteMap := make(map[string]ssh.VMInfo)
	for _, rvm := range remoteVMs {
		remoteMap[strings.ToLower(rvm.Name)] = rvm
	}

	// 1. Filtrar registros locales: solo conservar las que REALMENTE existen en VirtualBox
	activeRecords := make([]VMRecord, 0, len(records))
	for _, rec := range records {
		if rvm, exists := remoteMap[strings.ToLower(rec.Name)]; exists {
			if rvm.Status != "" {
				rec.State = rvm.Status
			}
			s.normalizeRecordNetworkDefaults(&rec)
			activeRecords = append(activeRecords, rec)
			delete(remoteMap, strings.ToLower(rec.Name))
		} else {
			s.log.Info("VM en cache local ya no existe en VirtualBox, limpiando registro y llaves asociadas", "name", rec.Name)
			_ = keys.DeleteVMKeys(s.cfg.SSHRoot, rec.Name)
			if rec.KeyID != "" {
				_ = s.keyStore.Delete(rec.KeyID)
			}
			_ = s.keyStore.Delete(rec.Name)
		}
	}

	// 2. Agregar VMs que existen en VirtualBox pero que NO estaban en vms.json
	for _, rvm := range remoteMap {
		status := rvm.Status
		if status == "" {
			status = "poweroff"
		}
		info, sErr := ssh.ShowVMInfo(s.SSHConfig(), rvm.Name, s.log)
		cores := 2
		memoryMB := 2048
		osType := "Linux"
		mac1 := ""
		mac2 := ""
		if sErr == nil {
			if info.Status != "" {
				status = info.Status
			}
			if info.CPU > 0 {
				cores = info.CPU
			}
			if info.MemoryMB > 0 {
				memoryMB = info.MemoryMB
			}
			if info.OS != "" {
				osType = info.OS
			}
			mac1 = info.MACAddress1
			mac2 = info.MACAddress2
		}

		discoveredRec := VMRecord{
			ID:          fmt.Sprintf("vm-%d", time.Now().UnixNano()),
			Name:        rvm.Name,
			OS:          osType,
			Cores:       cores,
			MemoryMB:    memoryMB,
			State:       status,
			Network:     s.NATNetworkName(),
			NetworkName: s.NATNetworkName(),
			NetworkCIDR: s.NATNetworkCIDR(),
			MACAddress1: mac1,
			MACAddress2: mac2,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		}
		s.normalizeRecordNetworkDefaults(&discoveredRec)
		activeRecords = append(activeRecords, discoveredRec)
	}

	_ = s.vmStore.Save(activeRecords)
	return activeRecords, nil
}

// GetCached devuelve la VM desde el almacén local vms.json de forma inmediata sin ejecutar llamadas VBoxManage por SSH.
// Si no se encuentra en el almacén local (por nombre o ID), delega a Get().
func (s *Service) GetCached(nameOrID string) (VMRecord, error) {
	nameOrID = strings.TrimSpace(nameOrID)
	if rec, found, err := s.vmStore.Find(nameOrID); err == nil && found {
		s.normalizeRecordNetworkDefaults(&rec)
		return rec, nil
	}
	if list, err := s.vmStore.Load(); err == nil {
		for _, rec := range list {
			if rec.ID == nameOrID || strings.EqualFold(rec.Name, nameOrID) {
				s.normalizeRecordNetworkDefaults(&rec)
				return rec, nil
			}
		}
	}
	return s.Get(nameOrID)
}

func (s *Service) Get(name string) (VMRecord, error) {
	name = strings.TrimSpace(name)
	rec, found, err := s.vmStore.Find(name)
	if err != nil {
		s.log.Warn("error buscando en store local", "name", name, "error", err)
	}

	if found {
		if rvm, err := ssh.ShowVMInfo(s.SSHConfig(), name, s.log); err == nil && rvm.Status != "" {
			rec.State = rvm.Status
			if rvm.MACAddress1 != "" {
				rec.MACAddress1 = rvm.MACAddress1
			}
			if rvm.MACAddress2 != "" {
				rec.MACAddress2 = rvm.MACAddress2
			}
			s.normalizeRecordNetworkDefaults(&rec)
			return rec, nil
		}
		if vms, vErr := ssh.ListVMs(s.SSHConfig(), s.log); vErr == nil {
			exists := false
			for _, v := range vms {
				if strings.EqualFold(v.Name, name) {
					exists = true
					break
				}
			}
			if !exists {
				s.log.Info("VM no encontrada en VirtualBox al consultar Get, limpiando registro local", "name", name)
				_ = s.vmStore.Delete(name)
				return VMRecord{}, fmt.Errorf("maquina virtual no encontrada: %s", name)
			}
		}
		s.normalizeRecordNetworkDefaults(&rec)
		return rec, nil
	}

	// Si no está en store local, consultar directamente en VirtualBox
	rvm, err := ssh.ShowVMInfo(s.SSHConfig(), name, s.log)
	if err != nil {
		return VMRecord{}, fmt.Errorf("maquina virtual no encontrada: %s", name)
	}

	cores := rvm.CPU
	if cores <= 0 {
		cores = 2
	}
	memory := rvm.MemoryMB
	if memory <= 0 {
		memory = 2048
	}

	newRec := VMRecord{
		ID:          fmt.Sprintf("vm-%d", time.Now().UnixNano()),
		Name:        rvm.Name,
		OS:          rvm.OS,
		Cores:       cores,
		MemoryMB:    memory,
		State:       rvm.Status,
		Network:     s.NATNetworkName(),
		NetworkName: s.NATNetworkName(),
		NetworkCIDR: s.NATNetworkCIDR(),
		MACAddress1: rvm.MACAddress1,
		MACAddress2: rvm.MACAddress2,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	s.normalizeRecordNetworkDefaults(&newRec)
	return newRec, nil
}

// VMProgressCallback define la función que recibe notificaciones de avance en la creación de una VM.
type VMProgressCallback func(stage string, step int, total int, message string)

// CreateFromTemplate ejecuta el workflow completo de creacion de VM a partir de plantilla con disco Multiattach
func (s *Service) CreateFromTemplate(ctx context.Context, req VMCreateTemplateRequest) (VMRecord, error) {
	return s.CreateFromTemplateWithProgress(ctx, req, nil)
}

// CreateFromTemplateWithProgress ejecuta el workflow completo reportando el progreso de cada etapa.
func (s *Service) CreateFromTemplateWithProgress(ctx context.Context, req VMCreateTemplateRequest, onProgress VMProgressCallback) (VMRecord, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return VMRecord{}, fmt.Errorf("el nombre de la máquina es obligatorio")
	}
	templateID := strings.TrimSpace(req.TemplateID)
	if templateID == "" {
		return VMRecord{}, fmt.Errorf("la plantilla es obligatoria")
	}

	notify := func(stage string, step int, total int, message string) {
		if onProgress != nil {
			onProgress(stage, step, total, message)
		}
	}

	mu := s.getVMLock(name)
	mu.Lock()
	defer mu.Unlock()

	notify("validating", 1, 6, "Validando plantilla base y generando claves Ed25519...")

	// 1. Validar si la máquina ya existe en VirtualBox para evitar colisiones
	if vms, err := ssh.ListVMs(s.SSHConfig(), s.log); err == nil {
		for _, vm := range vms {
			if strings.EqualFold(vm.Name, name) {
				return VMRecord{}, fmt.Errorf("la máquina virtual '%s' ya existe en VirtualBox", name)
			}
		}
	}
	// Limpiar cualquier registro huérfano local de una VM eliminada externamente
	_ = s.vmStore.Delete(name)

	// 2. Validar existencia y metadatos de la plantilla
	tpl, err := s.tplSvc.GetTemplateByID(ctx, templateID)
	if err != nil {
		return VMRecord{}, fmt.Errorf("validando plantilla: %w", err)
	}

	if req.Cores <= 0 {
		req.Cores = 2
	}
	if req.MemoryMB <= 0 {
		req.MemoryMB = 4096
	}

	// 3. Asignar puerto libre para NAT Port Forwarding
	hostPort, err := s.portAlloc.AllocatePort(ctx)
	if err != nil {
		return VMRecord{}, fmt.Errorf("asignando puerto NAT: %w", err)
	}

	// 4. Generar par de llaves Ed25519 nativo y usuario unico Linux
	identity, err := keys.GenerateEd25519KeyForVM(s.cfg.SSHRoot, name)
	if err != nil {
		return VMRecord{}, fmt.Errorf("generando claves Ed25519: %w", err)
	}

	s.log.Info("aprovisionando VM a partir de plantilla",
		"vm", name,
		"template", tpl.Name,
		"hostPort", hostPort,
		"user", identity.Username,
	)

	// 5. Crear VM y conectar base.vdi en modo Multiattach con doble adaptador de red
	vboxReq := ssh.VMTemplateCreateRequest{
		Name:           name,
		TemplateID:     tpl.ID,
		TemplateName:   tpl.Name,
		OSType:         tpl.OSType,
		BaseDiskPath:   tpl.DiskPath,
		VMRoot:         s.cfg.VMRoot,
		Cores:          req.Cores,
		MemoryMB:       req.MemoryMB,
		HostPort:       hostPort,
		GuestPort:      22,
		NATNetworkName: s.NATNetworkName(),
		NATNetworkCIDR: s.NATNetworkCIDR(),
	}

	notify("creating_vm", 2, 6, fmt.Sprintf("Creando máquina virtual '%s' en VirtualBox...", name))
	if err := ssh.CreateVMFromTemplate(ctx, s.SSHConfig(), vboxReq, s.log); err != nil {
		if !strings.Contains(err.Error(), "ya existe en VirtualBox") {
			_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
		}
		return VMRecord{}, fmt.Errorf("creando VM en VirtualBox: %w", err)
	}

	// 5. Iniciar VM en modo headless para aprovisionar
	notify("starting_vm", 3, 6, "Iniciando máquina virtual en modo headless...")
	if err := ssh.StartVM(s.SSHConfig(), name, s.log); err != nil {
		_ = ssh.DeleteVM(s.SSHConfig(), name, s.log)
		_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
		if strings.Contains(err.Error(), "Locking of attached media failed") {
			return VMRecord{}, fmt.Errorf("el disco base de la plantilla está bloqueado por otra VM en ejecución en VirtualBox (ej. una VM previa encendida). Apaga las otras máquinas virtuales que usen esta plantilla o configura el disco base en modo 'multiattach'")
		}
		return VMRecord{}, fmt.Errorf("iniciando VM para provisioning: %w", err)
	}

	sshUser := tpl.SSHUser
	if sshUser == "" {
		sshUser = "bootstrap"
	}

	// 5.1 Enviar por consola virtual / guestcontrol la generacion de host keys y reinicio de sshd en background con reintentos
	// (requerido cuando la plantilla no tiene host keys preinstaladas para que sean unicas)
	bootCtx, cancelBoot := context.WithCancel(ctx)
	defer cancelBoot()
	go func() {
		// La VM tarda en arrancar la primera vez (interfaz grafica/servicios del sistema).
		// Esperamos 15s antes del primer intento y luego reintentamos cada 15s hasta que SSH responda.
		select {
		case <-bootCtx.Done():
			return
		case <-time.After(15 * time.Second):
		}

		for attempt := 1; attempt <= 14; attempt++ {
			select {
			case <-bootCtx.Done():
				return
			default:
			}
			s.log.Info("ejecutando inicializacion de host keys SSH en la VM...", "vm", name, "intento", attempt)
			_ = ssh.BootstrapVMHostKeys(bootCtx, s.SSHConfig(), name, sshUser, "admin", s.log)
			select {
			case <-bootCtx.Done():
				return
			case <-time.After(15 * time.Second):
			}
		}
	}()

	// 6. Esperar a que SSH este disponible en el puerto NAT forwardeado (via tunel)
	// Timeout extendido a 240s (4 minutos) para dar margen holgado al primer arranque con GUI
	notify("waiting_ssh", 4, 6, "Esperando inicio del sistema operativo y servicio SSH (~30-60s)...")
	if err := ssh.WaitForSSH(ctx, s.SSHConfig(), hostPort, 240*time.Second, s.log); err != nil {
		cancelBoot()
		s.log.Error("timeout esperando SSH en VM, ejecutando rollback", "error", err)
		_ = ssh.DeleteVM(s.SSHConfig(), name, s.log)
		_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
		return VMRecord{}, fmt.Errorf("tiempo de espera agotado esperando puerto SSH en VM: %w", err)
	}
	cancelBoot()

	// 7. Aprovisionar usuario unico con la clave bootstrap y eliminar bootstrap de la VM
	notify("configuring_user", 5, 6, fmt.Sprintf("Configurando usuario único '%s' y accesos SSH...", identity.Username))
	if err := ssh.ProvisionVMUser(ctx, s.SSHConfig(), hostPort, sshUser, s.cfg.BootstrapKeyPath, identity.Username, identity.PublicKey, s.log, name); err != nil {
		s.log.Error("error en provisioning de usuario, ejecutando rollback", "error", err)
		_ = ssh.DeleteVM(s.SSHConfig(), name, s.log)
		_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
		return VMRecord{}, fmt.Errorf("fallo el aprovisionamiento de usuario unico en VM: %w", err)
	}

	// 8. Registrar la llave en el KeyStore para descarga desde la app
	notify("configuring_docker", 6, 6, "Configurando permisos de Docker y registrando máquina virtual...")
	keyID := fmt.Sprintf("key-%s-%d", keys.SanitizeName(name), time.Now().Unix())
	keyName := fmt.Sprintf("id_ed25519_%s", keys.SanitizeName(name))
	keyRec := keys.SSHKey{
		ID:          keyID,
		VMName:      name,
		Name:        keyName,
		Algorithm:   "Ed25519",
		PublicKey:   identity.PublicKey,
		PrivateKey:  identity.PrivateKeyPEM,
		Fingerprint: identity.Fingerprint,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.keyStore.Add(keyRec); err != nil {
		s.log.Warn("no se pudo guardar la llave en keys.json", "error", err)
	}

	// 9. Intentar registrar la Host Key en known_hosts (opcional)
	_, _ = s.knownHostsMgr.FetchAndStoreHostKey(s.cfg.SSHHost, hostPort)

	// 10. Consultar MAC addresses configuradas en VirtualBox para los dos adaptadores
	var mac1, mac2 string
	if vInfo, iErr := ssh.ShowVMInfo(s.SSHConfig(), name, s.log); iErr == nil {
		mac1 = vInfo.MACAddress1
		mac2 = vInfo.MACAddress2
	}

	// 11. Persistir registro inicial en vms.json
	rec := VMRecord{
		ID:             fmt.Sprintf("vm-%d", time.Now().UnixNano()),
		Name:           name,
		TemplateID:     tpl.ID,
		TemplateName:   tpl.Name,
		OS:             tpl.Name,
		Cores:          req.Cores,
		MemoryMB:       req.MemoryMB,
		State:          "running",
		KeyID:          keyID,
		KeyName:        keyName,
		SSHUser:        identity.Username,
		Host:           s.cfg.SSHHost,
		HostPort:       hostPort,
		GuestPort:      22,
		PrivateKeyPath: identity.PrivateKeyPath,
		PublicKey:      identity.PublicKey,
		DiskPath:       tpl.DiskPath,
		Network:        s.NATNetworkName(),
		NetworkName:    s.NATNetworkName(),
		NetworkCIDR:    s.NATNetworkCIDR(),
		MACAddress1:    mac1,
		MACAddress2:    mac2,
		Adapter1State:  "connected",
		Adapter2State:  "connected",
		FirstBoot:      true,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	savedRec, err := s.vmStore.Add(rec)
	if err != nil {
		s.log.Error("error al guardar en vms.json", "error", err)
		savedRec = rec
	}

	// 12. Descubrir la IP privada de la VM en la red compartida (vessel-net) relacionando las MACs
	notify("configuring_docker", 6, 6, fmt.Sprintf("Detectando IP privada en red compartida '%s'...", s.NATNetworkName()))
	if updatedRec, _, dErr := s.discoverNetworkUnlocked(ctx, savedRec); dErr == nil {
		savedRec = updatedRec
	} else {
		s.log.Warn("no se pudo completar el descubrimiento inicial de IP privada tras crear la VM", "vm", name, "error", dErr)
	}

	notify("done", 6, 6, fmt.Sprintf("¡Máquina virtual '%s' creada y aprovisionada con éxito!", name))
	return savedRec, nil
}

func (s *Service) Delete(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("nombre de VM obligatorio")
	}

	mu := s.getVMLock(name)
	mu.Lock()
	defer mu.Unlock()

	rec, found, _ := s.vmStore.Find(name)

	// 1. Eliminar de VirtualBox (sin tocar el disco base de la plantilla)
	if err := ssh.DeleteVM(s.SSHConfig(), name, s.log); err != nil {
		return fmt.Errorf("eliminando VM remota %q: %w", name, err)
	}

	// 2. Limpiar llaves locales Ed25519 en disco
	_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)

	// 3. Limpiar llave del almacén keys.json (por KeyID y por nombre de VM)
	if found && rec.KeyID != "" {
		_ = s.keyStore.Delete(rec.KeyID)
	}
	_ = s.keyStore.Delete(name)

	// 4. Limpiar entrada de known_hosts si tenia hostPort
	if found && rec.HostPort > 0 {
		_ = s.knownHostsMgr.RemoveHostKey(s.cfg.SSHHost, rec.HostPort)
	}

	// 5. Eliminar registro del store vms.json
	if err := s.vmStore.Delete(name); err != nil {
		return fmt.Errorf("eliminando registro local de VM %q: %w", name, err)
	}

	return nil
}

func (s *Service) Start(name string) error {
	name = strings.TrimSpace(name)
	mu := s.getVMLock(name)
	mu.Lock()
	defer mu.Unlock()

	rec, found, _ := s.vmStore.Find(name)

	// Antes de iniciar, garantizar que la red compartida NAT Network este activa
	// y si la VM fue creada con la topologia antigua (un solo NIC NAT), migrarla de forma transparente.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if found && rec.HostPort > 0 {
		guestPort := rec.GuestPort
		if guestPort <= 0 {
			guestPort = 22
		}
		if migrated, mErr := ssh.MigrateVMToDualNIC(ctx, s.SSHConfig(), name, s.NATNetworkName(), s.NATNetworkCIDR(), rec.HostPort, guestPort, s.log); mErr != nil {
			s.log.Warn("aviso verificando/migrando adaptadores de red antes de iniciar VM", "vm", name, "error", mErr)
		} else if migrated {
			s.log.Info("VM existente migrada exitosamente a doble adaptador antes de encender", "vm", name, "network", s.NATNetworkName())
		}
	} else {
		_, _ = ssh.EnsureNATNetwork(ctx, s.SSHConfig(), s.NATNetworkName(), s.NATNetworkCIDR(), s.log)
	}

	if err := ssh.StartVM(s.SSHConfig(), name, s.log); err != nil {
		return err
	}
	_ = s.vmStore.UpdateState(name, "running", false)

	// Si conocemos los datos de la VM, actualizar MACs y lanzar descubrimiento de IP en segundo plano
	if updatedRec, ok, _ := s.vmStore.Find(name); ok {
		go func(r VMRecord) {
			bgCtx, bgCancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer bgCancel()
			if r.HostPort > 0 {
				if err := ssh.WaitForSSH(bgCtx, s.SSHConfig(), r.HostPort, 60*time.Second, s.log); err == nil {
					_, _, _ = s.DiscoverAndPersistVMNetwork(bgCtx, r.Name)
				}
			}
		}(updatedRec)
	}

	return nil
}

func (s *Service) Stop(name string) error {
	name = strings.TrimSpace(name)
	mu := s.getVMLock(name)
	mu.Lock()
	defer mu.Unlock()

	if err := ssh.StopVM(s.SSHConfig(), name, s.log); err != nil {
		return err
	}
	_ = s.vmStore.UpdateState(name, "poweroff", false)
	return nil
}

func (s *Service) ForcePoweroff(name string) error {
	name = strings.TrimSpace(name)
	mu := s.getVMLock(name)
	mu.Lock()
	defer mu.Unlock()

	if err := ssh.ForcePoweroffVM(s.SSHConfig(), name, s.log); err != nil {
		return err
	}
	_ = s.vmStore.UpdateState(name, "poweroff", false)
	return nil
}

// DiscoverAndPersistVMNetwork descubre la IP privada de la VM relacionando las direcciones MAC
// de VirtualBox con las interfaces en Linux y persiste el resultado en vms.json.
func (s *Service) DiscoverAndPersistVMNetwork(ctx context.Context, name string) (VMRecord, *ssh.DiscoveredVMNetwork, error) {
	name = strings.TrimSpace(name)
	mu := s.getVMLock(name)
	mu.Lock()
	defer mu.Unlock()

	rec, err := s.Get(name)
	if err != nil {
		return VMRecord{}, nil, err
	}
	return s.discoverNetworkUnlocked(ctx, rec)
}

func (s *Service) discoverNetworkUnlocked(ctx context.Context, rec VMRecord) (VMRecord, *ssh.DiscoveredVMNetwork, error) {
	vInfo, err := ssh.ShowVMInfo(s.SSHConfig(), rec.Name, s.log)
	if err == nil {
		if vInfo.Status != "" {
			rec.State = vInfo.Status
		}
		if vInfo.MACAddress1 != "" {
			rec.MACAddress1 = vInfo.MACAddress1
		}
		if vInfo.MACAddress2 != "" {
			rec.MACAddress2 = vInfo.MACAddress2
		}
	}

	netName := s.NATNetworkName()
	netCIDR := s.NATNetworkCIDR()
	if err == nil && strings.EqualFold(vInfo.NIC1, "natnetwork") && vInfo.NATNet1 != "" {
		netName = vInfo.NATNet1
	}

	if !strings.EqualFold(rec.State, "running") {
		upd := VMNetworkUpdate{
			NetworkName:   netName,
			NetworkCIDR:   netCIDR,
			MACAddress1:   rec.MACAddress1,
			MACAddress2:   rec.MACAddress2,
			Adapter1State: "disconnected",
			Adapter2State: "disconnected",
		}
		if saved, uErr := s.vmStore.UpdateNetworkInfo(rec.Name, upd); uErr == nil {
			rec = saved
		}
		return rec, nil, fmt.Errorf("la máquina virtual '%s' está apagada (%s); enciéndela para descubrir su IP privada en '%s'", rec.Name, rec.State, netName)
	}

	if rec.HostPort <= 0 {
		return rec, nil, fmt.Errorf("la máquina virtual '%s' no tiene un puerto SSH administrativo asignado", rec.Name)
	}

	vmCfg := s.VMSSHConfig(rec)
	discovered, dErr := ssh.DiscoverVMNetworkByMAC(ctx, vmCfg, rec.MACAddress1, rec.MACAddress2, netName, netCIDR, s.log)
	if discovered != nil {
		upd := VMNetworkUpdate{
			PrivateIP:            discovered.PrivateIP,
			NetworkName:          discovered.NetworkName,
			NetworkCIDR:          discovered.NetworkCIDR,
			MACAddress1:          discovered.MACAddress1,
			MACAddress2:          discovered.MACAddress2,
			Adapter1State:        discovered.Adapter1State,
			Adapter2State:        discovered.Adapter2State,
			LastNetworkDiscovery: discovered.DiscoveredAt,
		}
		if saved, uErr := s.vmStore.UpdateNetworkInfo(rec.Name, upd); uErr == nil {
			rec = saved
		} else {
			rec.PrivateIP = discovered.PrivateIP
			rec.IP = discovered.PrivateIP
			rec.NetworkName = discovered.NetworkName
			rec.Network = discovered.NetworkName
			rec.NetworkCIDR = discovered.NetworkCIDR
			rec.MACAddress1 = discovered.MACAddress1
			rec.MACAddress2 = discovered.MACAddress2
			rec.Adapter1State = discovered.Adapter1State
			rec.Adapter2State = discovered.Adapter2State
			rec.LastNetworkDiscovery = discovered.DiscoveredAt
		}
	}
	if dErr != nil {
		return rec, discovered, dErr
	}
	return rec, discovered, nil
}

// GetNetworkInfo devuelve el estado detallado de red de una VM. Si esta encendida y no tiene IP
// (o se solicita forceRefresh), ejecuta el descubrimiento por MAC a traves de SSH.
func (s *Service) GetNetworkInfo(ctx context.Context, name string, forceRefresh bool) (VMNetworkInfoResponse, error) {
	rec, err := s.Get(name)
	if err != nil {
		return VMNetworkInfoResponse{}, err
	}

	var disc *ssh.DiscoveredVMNetwork
	var discErr error

	isRunning := strings.EqualFold(rec.State, "running")
	if isRunning && (forceRefresh || strings.TrimSpace(rec.PrivateIP) == "" || strings.TrimSpace(rec.MACAddress1) == "") {
		var updated VMRecord
		updated, disc, discErr = s.DiscoverAndPersistVMNetwork(ctx, name)
		if updated.Name != "" {
			rec = updated
		}
	}

	guestPort := rec.GuestPort
	if guestPort <= 0 {
		guestPort = 22
	}
	host := rec.Host
	if host == "" && s.cfg != nil {
		host = s.cfg.SSHHost
	}

	resp := VMNetworkInfoResponse{
		VMName:               rec.Name,
		State:                rec.State,
		PrivateIP:            rec.PrivateIP,
		NetworkName:          rec.NetworkName,
		NetworkCIDR:          rec.NetworkCIDR,
		Host:                 host,
		HostPort:             rec.HostPort,
		GuestPort:            guestPort,
		SSHUser:              rec.SSHUser,
		MACAddress1:          rec.MACAddress1,
		MACAddress2:          rec.MACAddress2,
		Adapter1State:        rec.Adapter1State,
		Adapter2State:        rec.Adapter2State,
		LastNetworkDiscovery: rec.LastNetworkDiscovery,
	}

	if disc != nil {
		resp.Interface1 = disc.Interface1
		resp.Interface2 = disc.Interface2
	}
	if !isRunning {
		resp.Adapter1State = "disconnected"
		resp.Adapter2State = "disconnected"
	} else if resp.Adapter1State == "" {
		if resp.PrivateIP != "" {
			resp.Adapter1State = "connected"
		} else {
			resp.Adapter1State = "pending_dhcp"
		}
	}
	if isRunning && resp.Adapter2State == "" && resp.HostPort > 0 {
		resp.Adapter2State = "connected"
	}
	if discErr != nil {
		resp.DiscoveryError = discErr.Error()
	}

	return resp, nil
}

// CheckConnectivity verifica la comunicacion directa por la red compartida (vessel-net)
// desde una VM origen hacia otra VM destino (por nombre de VM o direccion IP privada).
func (s *Service) CheckConnectivity(ctx context.Context, sourceVMName, targetVMOrIP string, targetPort int) (*ssh.VMConnectivityResult, error) {
	sourceVMName = strings.TrimSpace(sourceVMName)
	targetVMOrIP = strings.TrimSpace(targetVMOrIP)
	if sourceVMName == "" || targetVMOrIP == "" {
		return nil, fmt.Errorf("debe especificar la VM origen y la VM o IP destino")
	}

	sourceRec, err := s.Get(sourceVMName)
	if err != nil {
		return nil, fmt.Errorf("VM origen no encontrada: %w", err)
	}
	if !strings.EqualFold(sourceRec.State, "running") {
		return nil, fmt.Errorf("la VM origen '%s' debe estar encendida para verificar conectividad (estado actual: %s)", sourceRec.Name, sourceRec.State)
	}
	if sourceRec.HostPort <= 0 {
		return nil, fmt.Errorf("la VM origen '%s' no tiene puerto SSH configurado", sourceRec.Name)
	}

	// Asegurar que la VM origen tenga su red descubierta y configurada
	if strings.TrimSpace(sourceRec.PrivateIP) == "" {
		if updated, _, dErr := s.DiscoverAndPersistVMNetwork(ctx, sourceRec.Name); dErr == nil {
			sourceRec = updated
		}
	}

	targetName := ""
	targetIP := targetVMOrIP

	if parsedIP := net.ParseIP(targetVMOrIP); parsedIP == nil {
		// Es el nombre o ID de otra VM
		targetRec, tErr := s.Get(targetVMOrIP)
		if tErr != nil {
			return nil, fmt.Errorf("VM destino '%s' no encontrada: %w", targetVMOrIP, tErr)
		}
		targetName = targetRec.Name
		if !strings.EqualFold(targetRec.State, "running") {
			return nil, fmt.Errorf("la VM destino '%s' está apagada (%s)", targetRec.Name, targetRec.State)
		}
		if strings.TrimSpace(targetRec.PrivateIP) == "" {
			updatedTarget, _, dErr := s.DiscoverAndPersistVMNetwork(ctx, targetRec.Name)
			if dErr != nil || strings.TrimSpace(updatedTarget.PrivateIP) == "" {
				return nil, fmt.Errorf("la VM destino '%s' aún no tiene una IP privada descubierta en '%s': %v", targetRec.Name, s.NATNetworkName(), dErr)
			}
			targetRec = updatedTarget
		}
		targetIP = targetRec.PrivateIP
	}

	return ssh.VerifyInterVMConnectivity(ctx, s.VMSSHConfig(sourceRec), sourceRec.Name, targetName, targetIP, targetPort, s.log)
}
