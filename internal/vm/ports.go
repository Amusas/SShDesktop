package vm

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// PortAllocator gestiona la asignacion concurrente y libre de colisiones de puertos para NAT Port Forwarding
type PortAllocator struct {
	mu       sync.Mutex
	basePort int
	store    *Store
}

// NewPortAllocator crea un nuevo asignador de puertos a partir del puerto base y el store de VMs
func NewPortAllocator(basePort int, store *Store) *PortAllocator {
	if basePort <= 0 {
		basePort = 22000
	}
	return &PortAllocator{
		basePort: basePort,
		store:    store,
	}
}

// AllocatePort busca el siguiente puerto libre disponible a partir de basePort,
// asegurando que no este en uso por ninguna VM registrada ni escuchando localmente en el host.
func (pa *PortAllocator) AllocatePort(ctx context.Context) (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	usedPorts := make(map[int]bool)
	if pa.store != nil {
		vms, err := pa.store.Load()
		if err == nil {
			for _, v := range vms {
				if v.HostPort > 0 {
					usedPorts[v.HostPort] = true
				}
			}
		}
	}

	maxPort := pa.basePort + 1000
	for port := pa.basePort; port < maxPort; port++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if usedPorts[port] {
			continue
		}

		// Validar si el puerto esta disponible localmente
		if isPortAvailableLocal(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no se encontraron puertos libres en el rango %d-%d", pa.basePort, maxPort)
}

func isPortAvailableLocal(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	_ = listener.Close()
	return true
}
