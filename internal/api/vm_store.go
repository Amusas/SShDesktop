package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type VirtualMachine struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OS        string `json:"os"`
	ISO       string `json:"iso"`
	Cores     int    `json:"cores"`
	MemoryMB  int    `json:"memoryMB"`
	DiskGB    int    `json:"diskGB"`
	CreatedAt string `json:"createdAt"`
}

func (vm VirtualMachine) GetName() string { return vm.Name }
func (vm VirtualMachine) GetOS() string { return vm.OS }
func (vm VirtualMachine) GetISO() string { return vm.ISO }
func (vm VirtualMachine) GetCores() int { return vm.Cores }
func (vm VirtualMachine) GetMemoryMB() int { return vm.MemoryMB }
func (vm VirtualMachine) GetDiskGB() int { return vm.DiskGB }

type VirtualMachineCreateRequest struct {
	Name     string `json:"name"`
	OS       string `json:"os"`
	ISO      string `json:"iso"`
	Cores    int    `json:"cores"`
	MemoryMB int    `json:"memoryMB"`
	DiskGB   int    `json:"diskGB"`
}

type vmFileStore struct {
	path string
}

func newVMFileStore(path string) *vmFileStore {
	return &vmFileStore{path: path}
}

func (s *vmFileStore) Load() ([]VirtualMachine, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []VirtualMachine{}, nil
		}
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return []VirtualMachine{}, nil
	}

	var vms []VirtualMachine
	if err := json.Unmarshal(data, &vms); err != nil {
		return nil, fmt.Errorf("parseando vm store: %w", err)
	}
	return vms, nil
}

func (s *vmFileStore) Save(vms []VirtualMachine) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(vms, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *vmFileStore) Add(req VirtualMachineCreateRequest) (VirtualMachine, error) {
	vms, err := s.Load()
	if err != nil {
		return VirtualMachine{}, err
	}

	if req.Name == "" {
		return VirtualMachine{}, fmt.Errorf("nombre de la maquina virtual es obligatorio")
	}
	if req.OS == "" {
		return VirtualMachine{}, fmt.Errorf("so de la maquina virtual es obligatorio")
	}
	if req.ISO == "" {
		return VirtualMachine{}, fmt.Errorf("iso es obligatoria")
	}
	if req.Cores <= 0 {
		req.Cores = 1
	}
	if req.MemoryMB <= 0 {
		req.MemoryMB = 1024
	}
	if req.DiskGB <= 0 {
		req.DiskGB = 20
	}

	vm := VirtualMachine{
		ID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
		Name:      strings.TrimSpace(req.Name),
		OS:        strings.TrimSpace(req.OS),
		ISO:       strings.TrimSpace(req.ISO),
		Cores:     req.Cores,
		MemoryMB:  req.MemoryMB,
		DiskGB:    req.DiskGB,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	vms = append(vms, vm)
	if err := s.Save(vms); err != nil {
		return VirtualMachine{}, err
	}
	return vm, nil
}

func (s *vmFileStore) Delete(id string) error {
	vms, err := s.Load()
	if err != nil {
		return err
	}
	filtered := make([]VirtualMachine, 0, len(vms))
	for _, vm := range vms {
		if vm.ID != id {
			filtered = append(filtered, vm)
		}
	}
	return s.Save(filtered)
}

func (s *vmFileStore) Find(id string) (VirtualMachine, bool, error) {
	vms, err := s.Load()
	if err != nil {
		return VirtualMachine{}, false, err
	}
	for _, vm := range vms {
		if vm.ID == id {
			return vm, true, nil
		}
	}
	return VirtualMachine{}, false, nil
}

func (s *vmFileStore) ListSorted() ([]VirtualMachine, error) {
	vms, err := s.Load()
	if err != nil {
		return nil, err
	}
	sort.Slice(vms, func(i, j int) bool { return vms[i].Name < vms[j].Name })
	return vms, nil
}
