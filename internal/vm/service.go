package vm

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/keys"
	"github.com/feliren/ssh-web/internal/ssh"
)

type Service struct {
	cfg      *config.Config
	log      *slog.Logger
	vmStore  *Store
	keyStore *keys.Store
}

func NewService(cfg *config.Config, log *slog.Logger) *Service {
	vmPath := filepath.Join(".", "vms.json")
	keyPath := filepath.Join(".", "keys.json")
	return &Service{
		cfg:      cfg,
		log:      log,
		vmStore:  NewStore(vmPath),
		keyStore: keys.NewStore(keyPath),
	}
}

func (s *Service) KeyStore() *keys.Store {
	return s.keyStore
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
	// Cargar registros locales con características completas
	records, err := s.vmStore.Load()
	if err != nil {
		s.log.Warn("error leyendo vms.json", "error", err)
	}

	// Consultar estado remoto si hay conexión
	remoteVMs, rErr := ssh.ListVMs(s.SSHConfig(), s.log)
	if rErr == nil && len(remoteVMs) > 0 {
		remoteMap := make(map[string]ssh.VMInfo)
		for _, rvm := range remoteVMs {
			remoteMap[strings.ToLower(rvm.Name)] = rvm
		}

		// Actualizar estado de las VMs locales registradas
		for i := range records {
			if rvm, ok := remoteMap[strings.ToLower(records[i].Name)]; ok {
				if rvm.Status != "" {
					records[i].State = rvm.Status
				}
				delete(remoteMap, strings.ToLower(records[i].Name))
			}
		}

		// Agregar VMs remotas que no estuvieran en el archivo local
		for _, rvm := range remoteMap {
			records = append(records, VMRecord{
				ID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
				Name:      rvm.Name,
				OS:        rvm.OS,
				Cores:     rvm.CPU,
				MemoryMB:  rvm.MemoryMB,
				DiskGB:    rvm.DiskGB,
				State:     rvm.Status,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	return records, nil
}

func (s *Service) Get(name string) (VMRecord, error) {
	name = strings.TrimSpace(name)
	rec, found, err := s.vmStore.Find(name)
	if err != nil {
		s.log.Warn("error buscando en store local", "name", name, "error", err)
	}

	if found {
		// Consultar estado en vivo para actualizar si está disponible
		if rvm, err := ssh.ShowVMInfo(s.SSHConfig(), name, s.log); err == nil && rvm.Status != "" {
			rec.State = rvm.Status
			if rvm.DiskPath != "" {
				rec.DiskPath = rvm.DiskPath
			}
			if rvm.Network != "" {
				rec.Network = rvm.Network
			}
		}
		return rec, nil
	}

	// Si no está en store local, intentar consultar remotamente
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
	disk := rvm.DiskGB
	if disk <= 0 {
		disk = 40
	}

	return VMRecord{
		ID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
		Name:      rvm.Name,
		OS:        rvm.OS,
		Cores:     cores,
		MemoryMB:  memory,
		DiskGB:    disk,
		State:     rvm.Status,
		DiskPath:  rvm.DiskPath,
		Network:   rvm.Network,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) Delete(name string) error {
	name = strings.TrimSpace(name)
	_ = ssh.DeleteVM(s.SSHConfig(), name, s.log)
	_ = s.vmStore.Delete(name)
	_ = s.keyStore.Delete(name)
	return nil
}

func (s *Service) Create(req ssh.VMCreateRequest) (VMRecord, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return VMRecord{}, fmt.Errorf("el nombre de la máquina es obligatorio")
	}
	iso := strings.TrimSpace(req.ISO)
	if iso == "" {
		return VMRecord{}, fmt.Errorf("la imagen ISO es obligatoria")
	}

	if req.Cores <= 0 {
		req.Cores = 2
	}
	if req.MemoryMB <= 0 {
		req.MemoryMB = 2048
	}
	if req.DiskGB <= 0 {
		req.DiskGB = 40
	}
	if req.OS == "" {
		req.OS = "Linux"
	}

	// 1. Generar automáticamente un par de llaves SSH (pública y privada) para esta MV
	key, err := s.keyStore.GenerateKeyPairForVM(name)
	if err != nil {
		s.log.Warn("no se pudo generar llave SSH para la VM", "name", name, "error", err)
	}

	// 2. Ejecutar comando de creación remota vía SSH
	createdVM, sshErr := ssh.CreateVM(s.SSHConfig(), req, s.log)
	if sshErr != nil {
		s.log.Warn("fallo comando remoto SSH de creacion de VM (guardando registro local)", "error", sshErr)
	}

	diskPath := createdVM.DiskPath
	if diskPath == "" {
		diskPath = fmt.Sprintf("C:/VMs/%s/%s.vdi", name, name)
	}

	// 3. Persistir la máquina virtual en vms.json con todas sus características
	rec := VMRecord{
		ID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
		Name:      name,
		OS:        req.OS,
		ISO:       iso,
		Cores:     req.Cores,
		MemoryMB:  req.MemoryMB,
		DiskGB:    req.DiskGB,
		State:     "Creada",
		KeyID:     key.ID,
		KeyName:   key.Name,
		DiskPath:  diskPath,
		Network:   "NAT",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	savedRec, err := s.vmStore.Add(rec)
	if err != nil {
		s.log.Error("error al guardar en vms.json", "error", err)
		return rec, nil
	}

	return savedRec, nil
}
