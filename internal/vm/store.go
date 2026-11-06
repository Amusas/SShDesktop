package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type VMRecord struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	TemplateID           string `json:"templateId,omitempty"`
	TemplateName         string `json:"templateName,omitempty"`
	OS                   string `json:"os"`
	ISO                  string `json:"iso,omitempty"`
	Cores                int    `json:"cores"`
	MemoryMB             int    `json:"memoryMB"`
	DiskGB               int    `json:"diskGB,omitempty"`
	State                string `json:"state"`
	SSHUser              string `json:"sshUser,omitempty"`        // Usuario unico Linux (ej: vm_otramv_a82f)
	Host                 string `json:"host,omitempty"`           // Host Windows (ej: 192.168.101.80)
	HostPort             int    `json:"hostPort,omitempty"`       // Puerto NAT asignado en NIC 2 (ej: 22001)
	GuestPort            int    `json:"guestPort,omitempty"`      // Puerto guest interno (22)
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"` // Ruta a id_ed25519 en el backend
	PublicKey            string `json:"publicKey,omitempty"`      // Clave publica Ed25519
	KeyID                string `json:"keyId,omitempty"`
	KeyName              string `json:"keyName,omitempty"`
	GuestUser            string `json:"guestUser,omitempty"`
	DiskPath             string `json:"diskPath,omitempty"` // Ruta al base.vdi
	IP                   string `json:"ip,omitempty"`       // Alias compatible con privateIp
	PrivateIP            string `json:"privateIp,omitempty"`
	Network              string `json:"network,omitempty"` // Alias compatible con networkName
	NetworkName          string `json:"networkName,omitempty"`
	NetworkCIDR          string `json:"networkCidr,omitempty"`
	MACAddress1          string `json:"macAddress1,omitempty"`
	MACAddress2          string `json:"macAddress2,omitempty"`
	Adapter1State        string `json:"adapter1State,omitempty"`
	Adapter2State        string `json:"adapter2State,omitempty"`
	LastNetworkDiscovery string `json:"lastNetworkDiscovery,omitempty"`
	FirstBoot            bool   `json:"firstBoot"`
	CreatedAt            string `json:"createdAt"`
}

// VMNetworkUpdate contiene los campos de red a actualizar en el almacenamiento persistente
type VMNetworkUpdate struct {
	PrivateIP            string
	NetworkName          string
	NetworkCIDR          string
	MACAddress1          string
	MACAddress2          string
	Adapter1State        string
	Adapter2State        string
	LastNetworkDiscovery string
}

type Store struct {
	mu   sync.RWMutex
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() ([]VMRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnlocked()
}

func (s *Store) Save(vms []VMRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(vms)
}

func (s *Store) Add(rec VMRecord) (VMRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vms, _ := s.loadUnlocked()
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("vm-%d", time.Now().UnixNano())
	}
	if rec.CreatedAt == "" {
		rec.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if rec.State == "" {
		rec.State = "Creada"
	}

	// Reemplazar si ya existe con el mismo nombre
	found := false
	for i, v := range vms {
		if strings.EqualFold(v.Name, rec.Name) {
			vms[i] = rec
			found = true
			break
		}
	}
	if !found {
		vms = append(vms, rec)
	}

	if err := s.saveUnlocked(vms); err != nil {
		return VMRecord{}, err
	}
	return rec, nil
}

func (s *Store) Find(nameOrID string) (VMRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vms, err := s.loadUnlocked()
	if err != nil {
		return VMRecord{}, false, err
	}
	for _, v := range vms {
		if strings.EqualFold(v.Name, nameOrID) || v.ID == nameOrID {
			return v, true, nil
		}
	}
	return VMRecord{}, false, nil
}

func (s *Store) Delete(nameOrID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vms, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	filtered := make([]VMRecord, 0, len(vms))
	for _, v := range vms {
		if !strings.EqualFold(v.Name, nameOrID) && v.ID != nameOrID {
			filtered = append(filtered, v)
		}
	}
	return s.saveUnlocked(filtered)
}

// UpdateState actualiza el estado y opcionalmente marca que ya arrancó por primera vez.
func (s *Store) UpdateState(nameOrID, state string, markFirstBoot bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vms, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	found := false
	for i := range vms {
		if strings.EqualFold(vms[i].Name, nameOrID) || vms[i].ID == nameOrID {
			vms[i].State = state
			if markFirstBoot {
				vms[i].FirstBoot = true
			}
			if strings.EqualFold(state, "poweroff") {
				vms[i].Adapter1State = "disconnected"
				vms[i].Adapter2State = "disconnected"
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("VM no encontrada para actualizar estado: %s", nameOrID)
	}
	return s.saveUnlocked(vms)
}

func (s *Store) UpdateConnection(nameOrID, ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vms, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	for i := range vms {
		if strings.EqualFold(vms[i].Name, nameOrID) || vms[i].ID == nameOrID {
			vms[i].IP = ip
			vms[i].PrivateIP = ip
			return s.saveUnlocked(vms)
		}
	}
	return fmt.Errorf("VM no encontrada para actualizar conexion: %s", nameOrID)
}

// UpdateNetworkInfo actualiza la informacion de red descubierta para una VM en vms.json
func (s *Store) UpdateNetworkInfo(nameOrID string, upd VMNetworkUpdate) (VMRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vms, err := s.loadUnlocked()
	if err != nil {
		return VMRecord{}, err
	}
	for i := range vms {
		if strings.EqualFold(vms[i].Name, nameOrID) || vms[i].ID == nameOrID {
			if upd.PrivateIP != "" {
				vms[i].PrivateIP = upd.PrivateIP
				vms[i].IP = upd.PrivateIP
			}
			if upd.NetworkName != "" {
				vms[i].NetworkName = upd.NetworkName
				vms[i].Network = upd.NetworkName
			}
			if upd.NetworkCIDR != "" {
				vms[i].NetworkCIDR = upd.NetworkCIDR
			}
			if upd.MACAddress1 != "" {
				vms[i].MACAddress1 = upd.MACAddress1
			}
			if upd.MACAddress2 != "" {
				vms[i].MACAddress2 = upd.MACAddress2
			}
			if upd.Adapter1State != "" {
				vms[i].Adapter1State = upd.Adapter1State
			}
			if upd.Adapter2State != "" {
				vms[i].Adapter2State = upd.Adapter2State
			}
			if upd.LastNetworkDiscovery != "" {
				vms[i].LastNetworkDiscovery = upd.LastNetworkDiscovery
			}
			if err := s.saveUnlocked(vms); err != nil {
				return VMRecord{}, err
			}
			return vms[i], nil
		}
	}
	return VMRecord{}, fmt.Errorf("VM no encontrada para actualizar informacion de red: %s", nameOrID)
}

func (s *Store) loadUnlocked() ([]VMRecord, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []VMRecord{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []VMRecord{}, nil
	}
	var vms []VMRecord
	if err := json.Unmarshal(data, &vms); err != nil {
		return nil, fmt.Errorf("parseando vms.json: %w", err)
	}
	return vms, nil
}

func (s *Store) saveUnlocked(vms []VMRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(vms, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
