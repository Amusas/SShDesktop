package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// RegistryStore gestiona la persistencia en disco (registry.json) del Registry local y del Pull-through Cache.
type RegistryStore struct {
	mu       sync.RWMutex
	filePath string
}

// NewRegistryStore crea una nueva instancia del almacén de configuración de Registry.
func NewRegistryStore(filePath string) *RegistryStore {
	if strings.TrimSpace(filePath) == "" {
		filePath = "registry.json"
	}
	return &RegistryStore{
		filePath: filePath,
	}
}

// Load carga el estado persistido de registry.json. Si no existe, devuelve una estructura vacía sin error.
func (s *RegistryStore) Load() (RegistryPersistedState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnlocked()
}

func (s *RegistryStore) loadUnlocked() (RegistryPersistedState, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return RegistryPersistedState{}, nil
		}
		return RegistryPersistedState{}, fmt.Errorf("leyendo %s: %w", s.filePath, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return RegistryPersistedState{}, nil
	}
	var state RegistryPersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return RegistryPersistedState{}, fmt.Errorf("parseando %s: %w", s.filePath, err)
	}
	return state, nil
}

// Save persiste el estado completo en registry.json.
func (s *RegistryStore) Save(state RegistryPersistedState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(state)
}

func (s *RegistryStore) saveUnlocked(state RegistryPersistedState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando estado de registry: %w", err)
	}
	if err := os.WriteFile(s.filePath, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("escribiendo %s: %w", s.filePath, err)
	}
	return nil
}

// SetInstance actualiza y persiste la configuración de la instancia "local" o "cache".
func (s *RegistryStore) SetInstance(cfg RegistryInstanceConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadUnlocked()
	if err != nil {
		state = RegistryPersistedState{}
	}
	copyCfg := cfg
	if strings.EqualFold(cfg.Target, RegistryTargetCache) {
		copyCfg.Target = RegistryTargetCache
		state.Cache = &copyCfg
	} else {
		copyCfg.Target = RegistryTargetLocal
		state.Local = &copyCfg
	}
	return s.saveUnlocked(state)
}

// ClearInstance elimina la configuración registrada de la instancia "local" o "cache".
func (s *RegistryStore) ClearInstance(target string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadUnlocked()
	if err != nil {
		return nil
	}
	if strings.EqualFold(target, RegistryTargetCache) {
		state.Cache = nil
	} else {
		state.Local = nil
	}
	return s.saveUnlocked(state)
}
