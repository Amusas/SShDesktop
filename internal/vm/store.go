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
	ID        string `json:"id"`
	Name      string `json:"name"`
	OS        string `json:"os"`
	ISO       string `json:"iso"`
	Cores     int    `json:"cores"`
	MemoryMB  int    `json:"memoryMB"`
	DiskGB    int    `json:"diskGB"`
	State     string `json:"state"`
	KeyID     string `json:"keyId,omitempty"`
	KeyName   string `json:"keyName,omitempty"`
	DiskPath  string `json:"diskPath,omitempty"`
	Network   string `json:"network,omitempty"`
	CreatedAt string `json:"createdAt"`
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
