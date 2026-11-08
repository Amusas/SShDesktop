package docker

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// ClusterStore gestiona la persistencia en disco (clusters.json) de los clusters Docker Swarm creados en Vessel.
type ClusterStore struct {
	path string
	mu   sync.RWMutex
}

// NewClusterStore inicializa el almacén de clusters Docker Swarm.
func NewClusterStore(path string) *ClusterStore {
	if strings.TrimSpace(path) == "" {
		path = "clusters.json"
	}
	return &ClusterStore{path: path}
}

func (s *ClusterStore) Load() ([]SwarmCluster, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnlocked()
}

func (s *ClusterStore) loadUnlocked() ([]SwarmCluster, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SwarmCluster{}, nil
		}
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return []SwarmCluster{}, nil
	}

	var list []SwarmCluster
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	for i := range list {
		list[i].RecomputeCounters()
	}
	return list, nil
}

func (s *ClusterStore) Save(clusters []SwarmCluster) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(clusters)
}

func (s *ClusterStore) saveUnlocked(clusters []SwarmCluster) error {
	for i := range clusters {
		clusters[i].RecomputeCounters()
	}
	data, err := json.MarshalIndent(clusters, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *ClusterStore) Add(rec SwarmCluster) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.loadUnlocked()
	if err != nil {
		return err
	}

	rec.RecomputeCounters()
	filtered := make([]SwarmCluster, 0, len(list)+1)
	for _, item := range list {
		if !strings.EqualFold(item.Name, rec.Name) && item.ID != rec.ID {
			filtered = append(filtered, item)
		}
	}
	filtered = append(filtered, rec)
	return s.saveUnlocked(filtered)
}

func (s *ClusterStore) Update(rec SwarmCluster) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.loadUnlocked()
	if err != nil {
		return err
	}

	rec.RecomputeCounters()
	updated := false
	for i, item := range list {
		if item.ID == rec.ID || strings.EqualFold(item.Name, rec.Name) {
			list[i] = rec
			updated = true
			break
		}
	}
	if !updated {
		list = append(list, rec)
	}
	return s.saveUnlocked(list)
}

func (s *ClusterStore) Find(idOrName string) (SwarmCluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target := strings.TrimSpace(idOrName)
	if target == "" {
		return SwarmCluster{}, false
	}

	list, err := s.loadUnlocked()
	if err != nil {
		return SwarmCluster{}, false
	}

	for _, item := range list {
		if item.ID == target || strings.EqualFold(item.Name, target) {
			return item, true
		}
	}
	return SwarmCluster{}, false
}

func (s *ClusterStore) Delete(idOrName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := strings.TrimSpace(idOrName)
	list, err := s.loadUnlocked()
	if err != nil {
		return err
	}

	filtered := make([]SwarmCluster, 0, len(list))
	for _, item := range list {
		if item.ID != target && !strings.EqualFold(item.Name, target) {
			filtered = append(filtered, item)
		}
	}
	return s.saveUnlocked(filtered)
}

// FindClusterByVM busca si una máquina virtual (por ID o por nombre) ya pertenece a algún cluster registrado en Vessel.
func (s *ClusterStore) FindClusterByVM(vmID, vmName string) (SwarmCluster, SwarmNode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, err := s.loadUnlocked()
	if err != nil {
		return SwarmCluster{}, SwarmNode{}, false
	}

	cleanID := strings.TrimSpace(vmID)
	cleanName := strings.TrimSpace(vmName)

	for _, cluster := range list {
		if (cleanID != "" && cluster.ManagerVMID == cleanID) ||
			(cleanName != "" && strings.EqualFold(cluster.ManagerVMName, cleanName)) {
			for _, n := range cluster.Nodes {
				if (cleanID != "" && n.VMID == cleanID) || (cleanName != "" && strings.EqualFold(n.VMName, cleanName)) {
					return cluster, n, true
				}
			}
			return cluster, SwarmNode{
				VMID:   cluster.ManagerVMID,
				VMName: cluster.ManagerVMName,
				Role:   NodeRoleManager,
			}, true
		}

		for _, n := range cluster.Nodes {
			if (cleanID != "" && n.VMID == cleanID) ||
				(cleanName != "" && strings.EqualFold(n.VMName, cleanName)) {
				return cluster, n, true
			}
		}
	}

	return SwarmCluster{}, SwarmNode{}, false
}
