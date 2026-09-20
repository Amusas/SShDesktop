package vm

import (
	"context"
	"fmt"
	"log/slog"
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
		}

		discoveredRec := VMRecord{
			ID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
			Name:      rvm.Name,
			OS:        osType,
			Cores:     cores,
			MemoryMB:  memoryMB,
			State:     status,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		activeRecords = append(activeRecords, discoveredRec)
	}

	_ = s.vmStore.Save(activeRecords)
	return activeRecords, nil
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

	return VMRecord{
		ID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
		Name:      rvm.Name,
		OS:        rvm.OS,
		Cores:     cores,
		MemoryMB:  memory,
		State:     rvm.Status,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// CreateFromTemplate ejecuta el workflow completo de creacion de VM a partir de plantilla con disco Multiattach
func (s *Service) CreateFromTemplate(ctx context.Context, req VMCreateTemplateRequest) (VMRecord, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return VMRecord{}, fmt.Errorf("el nombre de la máquina es obligatorio")
	}
	templateID := strings.TrimSpace(req.TemplateID)
	if templateID == "" {
		return VMRecord{}, fmt.Errorf("la plantilla es obligatoria")
	}

	mu := s.getVMLock(name)
	mu.Lock()
	defer mu.Unlock()

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

	// 5. Crear VM y conectar base.vdi en modo Multiattach
	vboxReq := ssh.VMTemplateCreateRequest{
		Name:         name,
		TemplateID:   tpl.ID,
		TemplateName: tpl.Name,
		OSType:       tpl.OSType,
		BaseDiskPath: tpl.DiskPath,
		VMRoot:       s.cfg.VMRoot,
		Cores:        req.Cores,
		MemoryMB:     req.MemoryMB,
		HostPort:     hostPort,
		GuestPort:    22,
	}

	if err := ssh.CreateVMFromTemplate(ctx, s.SSHConfig(), vboxReq, s.log); err != nil {
		if !strings.Contains(err.Error(), "ya existe en VirtualBox") {
			_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
		}
		return VMRecord{}, fmt.Errorf("creando VM en VirtualBox: %w", err)
	}

	// 5. Iniciar VM en modo headless para aprovisionar
	if err := ssh.StartVM(s.SSHConfig(), name, s.log); err != nil {
		_ = ssh.DeleteVM(s.SSHConfig(), name, s.log)
		_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
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
	if err := ssh.WaitForSSH(ctx, s.SSHConfig(), hostPort, 240*time.Second, s.log); err != nil {
		cancelBoot()
		s.log.Error("timeout esperando SSH en VM, ejecutando rollback", "error", err)
		_ = ssh.DeleteVM(s.SSHConfig(), name, s.log)
		_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
		return VMRecord{}, fmt.Errorf("tiempo de espera agotado esperando puerto SSH en VM: %w", err)
	}
	cancelBoot()

	// 7. Aprovisionar usuario unico con la clave bootstrap y eliminar bootstrap de la VM
	if err := ssh.ProvisionVMUser(ctx, s.SSHConfig(), hostPort, sshUser, s.cfg.BootstrapKeyPath, identity.Username, identity.PublicKey, s.log); err != nil {
		s.log.Error("error en provisioning de usuario, ejecutando rollback", "error", err)
		_ = ssh.DeleteVM(s.SSHConfig(), name, s.log)
		_ = keys.DeleteVMKeys(s.cfg.SSHRoot, name)
		return VMRecord{}, fmt.Errorf("fallo el aprovisionamiento de usuario unico en VM: %w", err)
	}

	// 8. Registrar la llave en el KeyStore para descarga desde la app
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

	// 10. Persistir registro en vms.json
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
		Network:        "NAT",
		FirstBoot:      true,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	savedRec, err := s.vmStore.Add(rec)
	if err != nil {
		s.log.Error("error al guardar en vms.json", "error", err)
		return rec, nil
	}

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

	if err := ssh.StartVM(s.SSHConfig(), name, s.log); err != nil {
		return err
	}
	_ = s.vmStore.UpdateState(name, "running", false)
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
