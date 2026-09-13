package vm

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/ssh"
)

type Service struct {
	cfg *config.Config
	log *slog.Logger
}

func NewService(cfg *config.Config, log *slog.Logger) *Service {
	return &Service{cfg: cfg, log: log}
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

func (s *Service) List() ([]ssh.VMInfo, error) {
	vms, err := ssh.ListVMs(s.SSHConfig(), s.log)
	if err != nil {
		return nil, fmt.Errorf("listando vm: %w", err)
	}
	return vms, nil
}

func (s *Service) Get(name string) (ssh.VMInfo, error) {
	vm, err := ssh.ShowVMInfo(s.SSHConfig(), name, s.log)
	if err != nil {
		return ssh.VMInfo{}, fmt.Errorf("consultando vm: %w", err)
	}
	return vm, nil
}

func (s *Service) Delete(name string) error {
	if err := ssh.DeleteVM(s.SSHConfig(), name, s.log); err != nil {
		return fmt.Errorf("eliminando vm: %w", err)
	}
	return nil
}

func (s *Service) Create(req ssh.VMCreateRequest) (ssh.VMInfo, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ssh.VMInfo{}, fmt.Errorf("nombre es obligatorio")
	}
	if strings.TrimSpace(req.ISO) == "" {
		return ssh.VMInfo{}, fmt.Errorf("la ISO es obligatoria")
	}

	if _, err := FindOSByISO(req.ISO); err != nil {
		return ssh.VMInfo{}, err
	}

	vm, err := ssh.CreateVM(s.SSHConfig(), req, s.log)
	if err != nil {
		return ssh.VMInfo{}, fmt.Errorf("creando vm: %w", err)
	}
	return vm, nil
}
