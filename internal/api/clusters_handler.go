package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/feliren/ssh-web/internal/docker"
	"github.com/feliren/ssh-web/internal/vm"
)

// vmSwarmAdapter adapta *vm.Service a la interfaz docker.SwarmVMProvider.
type vmSwarmAdapter struct {
	vmSvc *vm.Service
}

func toSwarmVMInfo(rec vm.VMRecord) docker.SwarmVMInfo {
	privIP := strings.TrimSpace(rec.PrivateIP)
	if privIP == "" {
		privIP = strings.TrimSpace(rec.IP)
	}
	return docker.SwarmVMInfo{
		ID:            rec.ID,
		Name:          rec.Name,
		TemplateID:    rec.TemplateID,
		TemplateName:  rec.TemplateName,
		OS:            rec.OS,
		Cores:         rec.Cores,
		MemoryMB:      rec.MemoryMB,
		State:         rec.State,
		IP:            rec.IP,
		PrivateIP:     privIP,
		NetworkName:   rec.NetworkName,
		NetworkCIDR:   rec.NetworkCIDR,
		MACAddress1:   rec.MACAddress1,
		MACAddress2:   rec.MACAddress2,
		Adapter1State: rec.Adapter1State,
		Adapter2State: rec.Adapter2State,
		Host:          rec.Host,
		HostPort:      rec.HostPort,
		GuestPort:     rec.GuestPort,
		SSHUser:       rec.SSHUser,
		CreatedAt:     rec.CreatedAt,
	}
}

func (a *vmSwarmAdapter) ListVMs() ([]docker.SwarmVMInfo, error) {
	list, err := a.vmSvc.List()
	if err != nil {
		return nil, err
	}
	out := make([]docker.SwarmVMInfo, 0, len(list))
	for _, item := range list {
		out = append(out, toSwarmVMInfo(item))
	}
	return out, nil
}

func (a *vmSwarmAdapter) GetVM(nameOrID string) (docker.SwarmVMInfo, error) {
	rec, err := a.vmSvc.GetCached(nameOrID)
	if err != nil {
		return docker.SwarmVMInfo{}, err
	}
	return toSwarmVMInfo(rec), nil
}

func (a *vmSwarmAdapter) EnsureVMNetwork(ctx context.Context, name string) (docker.SwarmVMInfo, error) {
	netInfo, err := a.vmSvc.GetNetworkInfo(ctx, name, true)
	if err != nil {
		rec, getErr := a.vmSvc.Get(name)
		if getErr != nil {
			return docker.SwarmVMInfo{}, err
		}
		return toSwarmVMInfo(rec), err
	}
	rec, getErr := a.vmSvc.Get(name)
	if getErr != nil {
		return docker.SwarmVMInfo{
			Name:        netInfo.VMName,
			State:       netInfo.State,
			IP:          netInfo.PrivateIP,
			PrivateIP:   netInfo.PrivateIP,
			NetworkName: netInfo.NetworkName,
			NetworkCIDR: netInfo.NetworkCIDR,
		}, nil
	}
	if netInfo.PrivateIP != "" {
		rec.PrivateIP = netInfo.PrivateIP
		rec.IP = netInfo.PrivateIP
	}
	if netInfo.NetworkName != "" {
		rec.NetworkName = netInfo.NetworkName
	}
	if netInfo.NetworkCIDR != "" {
		rec.NetworkCIDR = netInfo.NetworkCIDR
	}
	return toSwarmVMInfo(rec), nil
}

func (a *vmSwarmAdapter) CheckVMConnectivity(ctx context.Context, sourceVM, targetIP string, port int) (bool, string, error) {
	res, err := a.vmSvc.CheckConnectivity(ctx, sourceVM, targetIP, port)
	if err != nil {
		return false, "", err
	}
	detail := strings.TrimSpace(res.Output)
	if res.Error != "" {
		if detail != "" {
			detail += " (" + res.Error + ")"
		} else {
			detail = res.Error
		}
	}
	return res.Reachable, detail, nil
}

// handleClusterEligibleVMs devuelve todas las VMs existentes y su estado de elegibilidad para Docker Swarm.
func (h *Handler) handleClusterEligibleVMs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	items, err := h.swarmSvc.GetVMEligibility(r.Context())
	if err != nil {
		h.log.Warn("no se pudo evaluar elegibilidad de VMs para Swarm", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// handleClusters gestiona GET /api/clusters y POST /api/clusters (con soporte SSE paso a paso).
func (h *Handler) handleClusters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		clusters, err := h.swarmSvc.ListClusters(r.Context())
		if err != nil {
			h.log.Warn("error listando clusters Docker Swarm", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(clusters)

	case http.MethodPost:
		var req docker.CreateClusterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON de solicitud inválido"})
			return
		}

		// Si el cliente solicita progreso en tiempo real mediante Server-Sent Events (SSE)
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			flusher, ok := w.(http.Flusher)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "streaming SSE no soportado"})
				return
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)
			flusher.Flush()

			sendEvent := func(ev docker.ClusterProgressEvent) {
				b, _ := json.Marshal(ev)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
			}

			created, err := h.swarmSvc.CreateClusterWithProgress(r.Context(), req, sendEvent)
			if err != nil {
				h.log.Warn("no se pudo crear cluster Docker Swarm", "name", req.Name, "error", err)
				sendEvent(docker.ClusterProgressEvent{
					Stage:   "error",
					Total:   11,
					Status:  docker.StepStatusFailed,
					Message: err.Error(),
					Error:   err.Error(),
				})
				return
			}

			sendEvent(docker.ClusterProgressEvent{
				Stage:   "done",
				Step:    11,
				Total:   11,
				Status:  docker.StepStatusCompleted,
				Message: fmt.Sprintf("¡Cluster '%s' creado correctamente!", created.Name),
				Cluster: created,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		created, err := h.swarmSvc.CreateClusterWithProgress(r.Context(), req, nil)
		if err != nil {
			h.log.Warn("no se pudo crear cluster Docker Swarm", "name", req.Name, "error", err)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)

	default:
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
	}
}

// handleClusterItem gestiona las rutas bajo /api/clusters/{id}...
// - GET /api/clusters/{id}
// - POST /api/clusters/{id}/workers
// - DELETE /api/clusters/{id}/nodes/{vm} (o POST /api/clusters/{id}/nodes/{vm}/remove)
func (h *Handler) handleClusterItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/clusters/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		http.Error(w, `{"error":"identificador de cluster requerido"}`, http.StatusBadRequest)
		return
	}

	parts := strings.Split(trimmed, "/")
	clusterID, err := url.PathUnescape(parts[0])
	if err != nil {
		clusterID = parts[0]
	}
	clusterID = strings.TrimSpace(clusterID)

	// Caso 1: GET /api/clusters/{id}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		cluster, getErr := h.swarmSvc.GetCluster(r.Context(), clusterID)
		if getErr != nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": getErr.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(cluster)
		return
	}

	// Caso 2: POST /api/clusters/{id}/workers (reintentar o incorporar un Worker)
	if len(parts) == 2 && parts[1] == "workers" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			VM string `json:"vm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.VM) == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "el campo 'vm' es obligatorio"})
			return
		}
		updated, retryErr := h.swarmSvc.RetryOrAddWorker(r.Context(), clusterID, strings.TrimSpace(body.VM))
		if retryErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": retryErr.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(updated)
		return
	}

	// Caso 3: DELETE /api/clusters/{id}/nodes/{vm} o POST /api/clusters/{id}/nodes/{vm}/remove
	if len(parts) >= 3 && parts[1] == "nodes" {
		if r.Method != http.MethodDelete && r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		vmRef, unescapeErr := url.PathUnescape(parts[2])
		if unescapeErr != nil {
			vmRef = parts[2]
		}
		vmRef = strings.TrimSpace(vmRef)
		if vmRef == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "nombre o ID de VM requerido"})
			return
		}

		force := r.URL.Query().Get("force") == "true" || r.URL.Query().Get("force") == "1"
		if !force && r.Body != nil {
			var body struct {
				Force bool `json:"force"`
			}
			if json.NewDecoder(r.Body).Decode(&body) == nil && body.Force {
				force = true
			}
		}

		result, rmErr := h.swarmSvc.RemoveNodeFromCluster(r.Context(), clusterID, vmRef, force)
		if rmErr != nil {
			if errors.Is(rmErr, docker.ErrWorkerUnreachable) && result != nil {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":             result.Message,
					"workerUnreachable": true,
					"removedVm":         result.RemovedVM,
					"role":              result.Role,
				})
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": rmErr.Error()})
			return
		}

		_ = json.NewEncoder(w).Encode(result)
		return
	}

	http.Error(w, `{"error":"ruta de cluster no encontrada"}`, http.StatusNotFound)
}
