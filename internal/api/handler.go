package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/docker"
	"github.com/feliren/ssh-web/internal/ssh"
	"github.com/feliren/ssh-web/internal/template"
	"github.com/feliren/ssh-web/internal/vm"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler gestiona las peticiones HTTP, el endpoint de configuracion y la conexion WebSocket
type Handler struct {
	cfgMu     sync.RWMutex
	cfg       *config.Config
	log       *slog.Logger
	static    http.Handler
	vmSvc     *vm.Service
	tplSvc    *template.Service
	dockerSvc *docker.Service
	hostsPath string
	hosts     hostsFile
}

// NewHandler inicializa el enrutador para archivos estaticos, configuracion y el endpoint WebSocket
func NewHandler(cfg *config.Config, log *slog.Logger, staticDir string) *Handler {
	hostsPath := filepath.Join(".", "hosts.json")
	hosts, err := loadHosts(hostsPath, cfg)
	if err != nil {
		log.Warn("no se pudo cargar hosts.json, se usara el host del .env", "error", err)
		hosts = hostsFile{ActiveHostID: "env-default", Hosts: []hostProfile{newHostProfile(cfg)}}
	}
	active := findHost(hosts.Hosts, hosts.ActiveHostID)
	if active != nil {
		applyHostToConfig(*active, cfg)
	}

	handler := &Handler{
		cfg:       cfg,
		log:       log,
		static:    http.FileServer(http.Dir(staticDir)),
		vmSvc:     vm.NewService(cfg, log),
		tplSvc:    template.NewService(cfg, log),
		hostsPath: hostsPath,
		hosts:     hosts,
	}
	handler.dockerSvc = docker.NewService(handler.executeVMCommand, log)
	if _, err := os.Stat(hostsPath); os.IsNotExist(err) {
		if err := saveHosts(hostsPath, hosts); err != nil {
			log.Warn("no se pudo crear hosts.json", "error", err)
		}
	}
	return handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/ws":
		h.handleWebSocket(w, r)
		return
	case r.URL.Path == "/api/config":
		h.handleConfig(w, r)
		return
	case r.URL.Path == "/api/hosts":
		h.handleHosts(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/hosts/"):
		h.handleHostByID(w, r)
		return
	case r.URL.Path == "/api/templates":
		h.handleTemplates(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/templates/"):
		h.handleTemplateByID(w, r)
		return
	case r.URL.Path == "/api/vms":
		h.handleVirtualMachines(w, r)
		return
	case r.URL.Path == "/api/isos":
		h.handleISOs(w, r)
		return
	case strings.HasSuffix(r.URL.Path, "/start") && strings.HasPrefix(r.URL.Path, "/api/vms/"):
		h.handleVMAction(w, r, "start")
		return
	case strings.HasSuffix(r.URL.Path, "/stop") && strings.HasPrefix(r.URL.Path, "/api/vms/"):
		h.handleVMAction(w, r, "stop")
		return
	case strings.HasSuffix(r.URL.Path, "/poweroff") && strings.HasPrefix(r.URL.Path, "/api/vms/"):
		h.handleVMAction(w, r, "poweroff")
		return
	case r.URL.Path == "/api/docker/catalog":
		h.handleDockerCatalog(w, r)
		return
	case strings.HasSuffix(r.URL.Path, "/docker/status") && strings.HasPrefix(r.URL.Path, "/api/vms/"):
		h.handleVMStatusDocker(w, r)
		return
	case strings.HasSuffix(r.URL.Path, "/images") && strings.HasPrefix(r.URL.Path, "/api/vms/"):
		h.handleVMImages(w, r)
		return
	case r.URL.Path == "/api/containers":
		h.handleContainers(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/containers/"):
		h.handleContainerItem(w, r)
		return
	case r.URL.Path == "/api/images":
		h.handleImages(w, r)
		return
	case r.URL.Path == "/api/images/build":
		h.handleImageBuild(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/images/"):
		h.handleImageItem(w, r)
		return
	case r.URL.Path == "/api/volumes":
		h.handleVolumes(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/volumes/"):
		h.handleVolumeItem(w, r)
		return
	case r.URL.Path == "/api/networks":
		h.handleNetworks(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/networks/"):
		h.handleNetworkItem(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/vms/"):
		h.handleVirtualMachineByID(w, r)
		return
	case r.URL.Path == "/api/keys":
		h.handleKeys(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/keys/"):
		h.handleKeyByID(w, r)
		return
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.static.ServeHTTP(w, r)
}

type configResponse struct {
	SSHHost      string `json:"sshHost"`
	SSHPort      int    `json:"sshPort"`
	SSHUser      string `json:"sshUser"`
	SSHShell     string `json:"sshShell"`
	SSHStartPath string `json:"sshStartPath"`
	SSHKeyPath   string `json:"sshKeyPath"`
}

type configUpdateRequest struct {
	SSHHost      string `json:"sshHost"`
	SSHPort      int    `json:"sshPort"`
	SSHUser      string `json:"sshUser"`
	SSHShell     string `json:"sshShell"`
	SSHStartPath string `json:"sshStartPath"`
	SSHKeyPath   string `json:"sshKeyPath"`
}

type hostListResponse struct {
	ActiveHostID string        `json:"activeHostId"`
	Hosts        []hostProfile `json:"hosts"`
}

func (h *Handler) handleHosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		h.cfgMu.RLock()
		response := hostListResponse{ActiveHostID: h.hosts.ActiveHostID, Hosts: append([]hostProfile(nil), h.hosts.Hosts...)}
		h.cfgMu.RUnlock()
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	var profile hostProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, `{"error":"cuerpo de solicitud invalido"}`, http.StatusBadRequest)
		return
	}
	profile.ID = newHostID()
	profile, err := normalizeHostProfile(profile)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	h.cfgMu.Lock()
	h.hosts.Hosts = append(h.hosts.Hosts, profile)
	if h.hosts.ActiveHostID == "" {
		h.hosts.ActiveHostID = profile.ID
		applyHostToConfig(profile, h.cfg)
	}
	err = saveHosts(h.hostsPath, h.hosts)
	h.cfgMu.Unlock()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"no se pudo guardar el host: %s"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

func (h *Handler) handleHostByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rawID := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	isActivate := strings.HasSuffix(rawID, "/activate")
	if isActivate {
		rawID = strings.TrimSuffix(rawID, "/activate")
	}
	id, err := url.PathUnescape(rawID)
	if err != nil || strings.TrimSpace(id) == "" {
		http.Error(w, `{"error":"id de host requerido"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var profile hostProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, `{"error":"cuerpo de solicitud invalido"}`, http.StatusBadRequest)
			return
		}
		profile.ID = id
		profile, err = normalizeHostProfile(profile)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		h.cfgMu.Lock()
		stored := findHost(h.hosts.Hosts, id)
		if stored == nil {
			h.cfgMu.Unlock()
			http.Error(w, `{"error":"host no encontrado"}`, http.StatusNotFound)
			return
		}
		*stored = profile
		if h.hosts.ActiveHostID == id {
			applyHostToConfig(profile, h.cfg)
		}
		err = saveHosts(h.hostsPath, h.hosts)
		h.cfgMu.Unlock()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"no se pudo guardar el host: %s"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(profile)

	case http.MethodPost:
		if !isActivate {
			http.Error(w, `{"error":"accion no permitida"}`, http.StatusBadRequest)
			return
		}
		h.cfgMu.Lock()
		profile := findHost(h.hosts.Hosts, id)
		if profile == nil {
			h.cfgMu.Unlock()
			http.Error(w, `{"error":"host no encontrado"}`, http.StatusNotFound)
			return
		}
		h.hosts.ActiveHostID = id
		applyHostToConfig(*profile, h.cfg)
		err = saveHosts(h.hostsPath, h.hosts)
		h.cfgMu.Unlock()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"no se pudo activar el host: %s"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(profile)

	case http.MethodDelete:
		h.cfgMu.Lock()
		if len(h.hosts.Hosts) <= 1 {
			h.cfgMu.Unlock()
			http.Error(w, `{"error":"debe existir al menos un host"}`, http.StatusBadRequest)
			return
		}
		index := -1
		for i := range h.hosts.Hosts {
			if h.hosts.Hosts[i].ID == id {
				index = i
				break
			}
		}
		if index == -1 {
			h.cfgMu.Unlock()
			http.Error(w, `{"error":"host no encontrado"}`, http.StatusNotFound)
			return
		}
		h.hosts.Hosts = append(h.hosts.Hosts[:index], h.hosts.Hosts[index+1:]...)
		if h.hosts.ActiveHostID == id {
			h.hosts.ActiveHostID = h.hosts.Hosts[0].ID
			applyHostToConfig(h.hosts.Hosts[0], h.cfg)
		}
		err = saveHosts(h.hostsPath, h.hosts)
		h.cfgMu.Unlock()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"no se pudo guardar la lista: %s"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": id})

	default:
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.cfgMu.RLock()
		resp := configResponse{
			SSHHost:      h.cfg.SSHHost,
			SSHPort:      h.cfg.SSHPort,
			SSHUser:      h.cfg.SSHUser,
			SSHShell:     h.cfg.SSHShell,
			SSHStartPath: h.cfg.SSHStartPath,
			SSHKeyPath:   h.cfg.SSHKeyPath,
		}
		h.cfgMu.RUnlock()

		json.NewEncoder(w).Encode(resp)

	case http.MethodPost:
		var req configUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"cuerpo de solicitud invalido"}`, http.StatusBadRequest)
			return
		}

		h.cfgMu.Lock()
		if req.SSHHost != "" {
			h.cfg.SSHHost = req.SSHHost
		}
		if req.SSHPort > 0 {
			h.cfg.SSHPort = req.SSHPort
		}
		if req.SSHUser != "" {
			h.cfg.SSHUser = req.SSHUser
		}
		if req.SSHShell != "" {
			h.cfg.SSHShell = req.SSHShell
		}
		h.cfg.SSHStartPath = req.SSHStartPath
		if req.SSHKeyPath != "" {
			h.cfg.SSHKeyPath = req.SSHKeyPath
		}

		if err := config.SaveProperties("config.properties", h.cfg); err != nil {
			h.log.Warn("no se pudo guardar config.properties", "error", err)
		} else {
			h.log.Info("config.properties actualizado exitosamente")
		}

		resp := configResponse{
			SSHHost:      h.cfg.SSHHost,
			SSHPort:      h.cfg.SSHPort,
			SSHUser:      h.cfg.SSHUser,
			SSHShell:     h.cfg.SSHShell,
			SSHStartPath: h.cfg.SSHStartPath,
			SSHKeyPath:   h.cfg.SSHKeyPath,
		}
		h.cfgMu.Unlock()

		json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleISOs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	entries := vm.Catalog()
	names := []string{}
	for _, e := range entries {
		names = append(names, e.ISOFilename)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"isos":  names,
		"items": entries,
	})
}

// handleTemplates maneja GET /api/templates descubriendo plantillas en C:\VMManager\Templates
func (h *Handler) handleTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	templates, err := h.tplSvc.DiscoverTemplates(r.Context())
	if err != nil {
		h.log.Warn("error al descubrir plantillas", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(templates)
}

// handleTemplateByID maneja GET /api/templates/{id} para consultar una plantilla individual
func (h *Handler) handleTemplateByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	id = strings.TrimSpace(id)
	decodedID, err := url.PathUnescape(id)
	if err != nil {
		decodedID = id
	}

	tpl, err := h.tplSvc.GetTemplateByID(r.Context(), decodedID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(tpl)
}

func (h *Handler) handleVirtualMachines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		vms, err := h.vmSvc.List()
		if err != nil {
			h.log.Warn("no se pudo listar maquinas virtuales", "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(vms)

	case http.MethodPost:
		var req vm.VMCreateTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"cuerpo de solicitud invalido"}`, http.StatusBadRequest)
			return
		}

		createdVM, err := h.vmSvc.CreateFromTemplate(r.Context(), req)
		if err != nil {
			h.log.Warn("no se pudo crear maquina virtual", "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createdVM)

	default:
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleVirtualMachineByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimPrefix(r.URL.Path, "/api/vms/")
	id = strings.TrimSpace(id)
	if id == "" {
		http.Error(w, `{"error":"id requerido"}`, http.StatusBadRequest)
		return
	}

	decodedID, err := url.PathUnescape(id)
	if err != nil {
		decodedID = id
	}
	decodedID = strings.TrimSpace(decodedID)

	switch r.Method {
	case http.MethodGet:
		vmData, err := h.vmSvc.Get(decodedID)
		if err != nil {
			h.log.Warn("no se pudo consultar detalle de maquina virtual", "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(vmData)

	case http.MethodDelete:
		if err := h.vmSvc.Delete(decodedID); err != nil {
			h.log.Warn("no se pudo eliminar maquina virtual", "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": decodedID})

	default:
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
	}
}

// handleVMAction gestiona las acciones start, stop y poweroff sobre una máquina virtual.
func (h *Handler) handleVMAction(w http.ResponseWriter, r *http.Request, action string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"metodo no permitido, use POST"}`, http.StatusMethodNotAllowed)
		return
	}

	// Extraer el nombre de la VM del path: /api/vms/{name}/start, /api/vms/{name}/stop o /api/vms/{name}/poweroff
	rawPath := strings.TrimPrefix(r.URL.Path, "/api/vms/")
	rawPath = strings.TrimSuffix(rawPath, "/"+action)
	rawPath = strings.TrimSpace(rawPath)

	name, err := url.PathUnescape(rawPath)
	if err != nil {
		name = rawPath
	}
	name = strings.TrimSpace(name)

	if name == "" {
		http.Error(w, `{"error":"nombre de VM requerido"}`, http.StatusBadRequest)
		return
	}

	switch action {
	case "start":
		if err := h.vmSvc.Start(name); err != nil {
			h.log.Warn("no se pudo iniciar la maquina virtual", "name", name, "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "started", "name": name})

	case "stop":
		if err := h.vmSvc.Stop(name); err != nil {
			h.log.Warn("no se pudo detener la maquina virtual", "name", name, "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "stopped", "name": name})

	case "poweroff":
		if err := h.vmSvc.ForcePoweroff(name); err != nil {
			h.log.Warn("no se pudo forzar apagado de la maquina virtual", "name", name, "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "poweroff", "name": name})

	default:
		http.Error(w, `{"error":"accion no reconocida"}`, http.StatusBadRequest)
	}
}

func (h *Handler) handleKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	keyList, err := h.vmSvc.KeyStore().List()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(keyList)
}

func (h *Handler) handleKeyByID(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/api/keys/")
	relPath = strings.TrimSpace(relPath)

	if strings.HasSuffix(relPath, "/private") {
		id := strings.TrimSuffix(relPath, "/private")
		id = strings.TrimSpace(id)
		key, err := h.vmSvc.KeyStore().Get(id)
		if err != nil {
			http.Error(w, "Llave no encontrada", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.pem\"", key.Name))
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Write([]byte(key.PrivateKey))
		return
	}

	if strings.HasSuffix(relPath, "/public") {
		id := strings.TrimSuffix(relPath, "/public")
		id = strings.TrimSpace(id)
		key, err := h.vmSvc.KeyStore().Get(id)
		if err != nil {
			http.Error(w, "Llave no encontrada", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.pub\"", key.Name))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(key.PublicKey))
		return
	}

	decodedID, err := url.PathUnescape(relPath)
	if err != nil {
		decodedID = relPath
	}

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		key, err := h.vmSvc.KeyStore().Get(decodedID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(key)

	case http.MethodDelete:
		if err := h.vmSvc.KeyStore().Delete(decodedID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": decodedID})

	default:
		http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
	}
}

type wsMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("fallo al actualizar la conexion a websocket", "error", err)
		return
	}

	h.log.Info("cliente websocket conectado", "remoto", r.RemoteAddr)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	q := r.URL.Query()
	vmTarget := strings.TrimSpace(q.Get("vm"))
	containerTarget := strings.TrimSpace(q.Get("container"))

	var host, user, shell, startPath, keyPath string
	var port int

	if vmTarget != "" {
		// Conectar directamente a una VM especifica
		vmRec, err := h.vmSvc.Get(vmTarget)
		if err != nil {
			conn.WriteJSON(wsMessage{Type: "error", Data: fmt.Sprintf("Máquina virtual '%s' no encontrada: %s", vmTarget, err)})
			conn.Close()
			return
		}
		if vmRec.HostPort <= 0 {
			conn.WriteJSON(wsMessage{Type: "error", Data: fmt.Sprintf("La máquina virtual '%s' no tiene puerto NAT asignado", vmTarget)})
			conn.Close()
			return
		}
		host = vmRec.Host
		if host == "" {
			host = h.cfg.SSHHost
		}
		port = vmRec.HostPort
		user = vmRec.SSHUser
		if user == "" {
			user = "bootstrap"
		}
		shell = "bash"
		startPath = ""
		keyPath = vmRec.PrivateKeyPath
		if keyPath == "" {
			keyPath = filepath.Join(h.cfg.SSHRoot, "vms", vmRec.Name, "id_ed25519")
		}

		// Mecanismo de autorecuperación: si el archivo de llave no existe físicamente en disco,
		// lo restauramos automáticamente desde el KeyStore (donde está persistido en keys.json).
		if _, statErr := os.Stat(keyPath); os.IsNotExist(statErr) {
			if kRec, kErr := h.vmSvc.KeyStore().Get(vmRec.Name); kErr == nil && strings.TrimSpace(kRec.PrivateKey) != "" {
				_ = os.MkdirAll(filepath.Dir(keyPath), 0700)
				_ = os.WriteFile(keyPath, []byte(kRec.PrivateKey), 0600)
				if kRec.PublicKey != "" {
					_ = os.WriteFile(keyPath+".pub", []byte(kRec.PublicKey), 0644)
				}
				h.log.Info("archivo de llave privada restaurado desde KeyStore", "vm", vmRec.Name, "keyPath", keyPath)
			}
		}
	} else {
		// Conectar al host Windows predeterminado
		h.cfgMu.RLock()
		host = q.Get("host")
		if host == "" {
			host = h.cfg.SSHHost
		}

		port = h.cfg.SSHPort
		if pStr := q.Get("port"); pStr != "" {
			if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
				port = p
			}
		}

		user = q.Get("user")
		if user == "" {
			user = h.cfg.SSHUser
		}

		shell = q.Get("shell")
		if shell == "" {
			shell = h.cfg.SSHShell
		}

		startPath = q.Get("startPath")
		if startPath == "" {
			startPath = h.cfg.SSHStartPath
		}

		keyPath = q.Get("keyPath")
		if keyPath == "" {
			keyPath = h.cfg.SSHKeyPath
		}
		h.cfgMu.RUnlock()
	}

	if host == "" || user == "" {
		h.log.Warn("intento de conexion con host o usuario vacio")
		conn.WriteJSON(wsMessage{
			Type: "error",
			Data: "Host o usuario no configurados. Abre el menú de configuración en la esquina superior izquierda para ingresar los datos de conexión.",
		})
		conn.Close()
		return
	}

	sshTargetCfg := ssh.Config{
		Host:      host,
		Port:      port,
		User:      user,
		KeyPath:   keyPath,
		Shell:     shell,
		StartPath: startPath,
	}

	if vmTarget != "" {
		h.cfgMu.RLock()
		sshTargetCfg.ProxyHost = &ssh.Config{
			Host:    h.cfg.SSHHost,
			Port:    h.cfg.SSHPort,
			User:    h.cfg.SSHUser,
			KeyPath: h.cfg.SSHKeyPath,
		}
		h.cfgMu.RUnlock()

		if containerTarget != "" {
			if err := docker.ValidateContainerName(containerTarget); err == nil || len(containerTarget) >= 6 {
				sshTargetCfg.ExecCommand = fmt.Sprintf("sudo docker exec -it %s /bin/sh -c 'if [ -x /bin/bash ]; then exec /bin/bash; else exec /bin/sh; fi'", docker.EscapeShellArg(containerTarget))
			}
		}
	}

	session, err := ssh.NewSession(ctx, sshTargetCfg, h.log)
	if err != nil {
		h.log.Error("fallo en la conexion SSH", "error", err)
		conn.WriteJSON(wsMessage{Type: "error", Data: err.Error()})
		conn.Close()
		return
	}
	defer session.Close()

	var writeMu sync.Mutex

	session.SetOutputHandler(func(data []byte) {
		writeMu.Lock()
		defer writeMu.Unlock()

		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.log.Warn("error al escribir en websocket", "error", err)
			cancel()
		}
	})

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				writeMu.Lock()
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				writeMu.Unlock()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			conn.SetReadDeadline(time.Now().Add(pongWait))

			var msg wsMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				if err := session.Write(message); err != nil {
					return
				}
				continue
			}

			switch msg.Type {
			case "input":
				if err := session.Write([]byte(msg.Data)); err != nil {
					return
				}
			case "resize":
				_ = session.Resize(msg.Cols, msg.Rows)
			}
		}
	}()

	wg.Wait()

	writeMu.Lock()
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.Close()
	writeMu.Unlock()
}

func (h *Handler) executeVMCommand(ctx context.Context, vmName string, command string) (string, error) {
	vmRec, err := h.vmSvc.Get(vmName)
	if err != nil {
		return "", fmt.Errorf("máquina virtual '%s' no encontrada: %w", vmName, err)
	}
	if vmRec.HostPort <= 0 {
		return "", fmt.Errorf("la máquina virtual '%s' no tiene puerto NAT asignado", vmName)
	}

	h.cfgMu.RLock()
	hostCfg := ssh.Config{
		Host:    h.cfg.SSHHost,
		Port:    h.cfg.SSHPort,
		User:    h.cfg.SSHUser,
		KeyPath: h.cfg.SSHKeyPath,
	}
	sshRoot := h.cfg.SSHRoot
	h.cfgMu.RUnlock()

	keyPath := vmRec.PrivateKeyPath
	if keyPath == "" {
		keyPath = filepath.Join(sshRoot, "vms", vmRec.Name, "id_ed25519")
	}

	if _, statErr := os.Stat(keyPath); os.IsNotExist(statErr) {
		if kRec, kErr := h.vmSvc.KeyStore().Get(vmRec.Name); kErr == nil && strings.TrimSpace(kRec.PrivateKey) != "" {
			_ = os.MkdirAll(filepath.Dir(keyPath), 0700)
			_ = os.WriteFile(keyPath, []byte(kRec.PrivateKey), 0600)
			if kRec.PublicKey != "" {
				_ = os.WriteFile(keyPath+".pub", []byte(kRec.PublicKey), 0644)
			}
		}
	}

	user := vmRec.SSHUser
	if user == "" {
		user = "bootstrap"
	}

	vmTargetCfg := ssh.Config{
		Host:      "127.0.0.1",
		Port:      vmRec.HostPort,
		User:      user,
		KeyPath:   keyPath,
		ProxyHost: &hostCfg,
	}

	return ssh.ExecuteCommandContext(ctx, vmTargetCfg, command, h.log)
}

func (h *Handler) handleDockerCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(docker.GetDefaultCatalog())
}

func (h *Handler) handleVMStatusDocker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "ruta no válida", http.StatusBadRequest)
		return
	}
	vmName := parts[2]
	status, err := h.dockerSvc.CheckDockerStatus(r.Context(), vmName)
	if err != nil {
		json.NewEncoder(w).Encode(docker.DockerStatus{
			Available: false,
			Error:     err.Error(),
		})
		return
	}
	json.NewEncoder(w).Encode(status)
}

func (h *Handler) handleContainers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		vmTarget := strings.TrimSpace(r.URL.Query().Get("vm"))
		if vmTarget != "" {
			containers, err := h.dockerSvc.ListContainers(r.Context(), vmTarget)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error listando contenedores: %s", err), http.StatusInternalServerError)
				return
			}
			if containers == nil {
				containers = []docker.Container{}
			}
			json.NewEncoder(w).Encode(containers)
			return
		}

		vms, err := h.vmSvc.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error listando VMs: %s", err), http.StatusInternalServerError)
			return
		}

		var allContainers []docker.Container
		for _, v := range vms {
			conts, err := h.dockerSvc.ListContainers(r.Context(), v.Name)
			if err == nil && len(conts) > 0 {
				allContainers = append(allContainers, conts...)
			}
		}
		if allContainers == nil {
			allContainers = []docker.Container{}
		}
		json.NewEncoder(w).Encode(allContainers)

	case http.MethodPost:
		var req docker.CreateContainerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("JSON inválido: %s", err), http.StatusBadRequest)
			return
		}

		created, err := h.dockerSvc.CreateContainer(r.Context(), req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creando contenedor: %s", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)

	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleContainerItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "ruta de contenedor inválida", http.StatusBadRequest)
		return
	}
	vmName := parts[2]
	containerID := parts[3]

	if len(parts) == 4 {
		switch r.Method {
		case http.MethodGet:
			inspect, err := h.dockerSvc.InspectContainer(r.Context(), vmName, containerID)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error inspeccionando contenedor: %s", err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(inspect)

		case http.MethodDelete:
			force := r.URL.Query().Get("force") == "true"
			if err := h.dockerSvc.DeleteContainer(r.Context(), vmName, containerID, force); err != nil {
				http.Error(w, fmt.Sprintf("Error eliminando contenedor: %s", err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Contenedor eliminado"})

		default:
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		}
		return
	}

	action := parts[4]
	if r.Method == http.MethodPost {
		switch action {
		case "start":
			if err := h.dockerSvc.StartContainer(r.Context(), vmName, containerID); err != nil {
				http.Error(w, fmt.Sprintf("Error iniciando contenedor: %s", err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Contenedor iniciado"})
		case "stop":
			if err := h.dockerSvc.StopContainer(r.Context(), vmName, containerID); err != nil {
				http.Error(w, fmt.Sprintf("Error deteniendo contenedor: %s", err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Contenedor detenido"})
		case "restart":
			if err := h.dockerSvc.RestartContainer(r.Context(), vmName, containerID); err != nil {
				http.Error(w, fmt.Sprintf("Error reiniciando contenedor: %s", err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Contenedor reiniciado"})
		case "commit":
			var req docker.CommitContainerRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, fmt.Sprintf("Error decodificando solicitud: %s", err), http.StatusBadRequest)
				return
			}
			req.VMName = vmName
			req.ContainerID = containerID
			resp, err := h.dockerSvc.CommitContainer(r.Context(), req)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error creando imagen desde contenedor: %s", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)
		default:
			http.Error(w, "acción no soportada", http.StatusBadRequest)
		}
		return
	}

	if r.Method == http.MethodGet && action == "logs" {
		tail := 200
		if tStr := r.URL.Query().Get("tail"); tStr != "" {
			if t, err := strconv.Atoi(tStr); err == nil && t > 0 {
				tail = t
			}
		}
		logs, err := h.dockerSvc.GetLogs(r.Context(), vmName, containerID, tail)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error obteniendo logs: %s", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"logs": logs})
		return
	}

	http.Error(w, "método o acción no permitida", http.StatusMethodNotAllowed)
}

func (h *Handler) handleVMImages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "ruta de VM inválida", http.StatusBadRequest)
		return
	}
	vmName := parts[2]
	images, err := h.dockerSvc.ListImages(r.Context(), vmName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listando imágenes en VM '%s': %s", vmName, err), http.StatusInternalServerError)
		return
	}
	if images == nil {
		images = []docker.Image{}
	}
	json.NewEncoder(w).Encode(images)
}

func (h *Handler) handleImages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		vmTarget := strings.TrimSpace(r.URL.Query().Get("vm"))
		if vmTarget != "" {
			images, err := h.dockerSvc.ListImages(r.Context(), vmTarget)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error listando imágenes: %s", err), http.StatusInternalServerError)
				return
			}
			if images == nil {
				images = []docker.Image{}
			}
			json.NewEncoder(w).Encode(images)
			return
		}

		vms, err := h.vmSvc.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error listando VMs: %s", err), http.StatusInternalServerError)
			return
		}

		var allImages []docker.Image
		for _, v := range vms {
			imgs, err := h.dockerSvc.ListImages(r.Context(), v.Name)
			if err == nil && len(imgs) > 0 {
				allImages = append(allImages, imgs...)
			}
		}
		if allImages == nil {
			allImages = []docker.Image{}
		}
		json.NewEncoder(w).Encode(allImages)

	case http.MethodPost:
		h.handleImageBuild(w, r)

	case http.MethodDelete:
		vmTarget := strings.TrimSpace(r.URL.Query().Get("vm"))
		imageRef := strings.TrimSpace(r.URL.Query().Get("image"))
		force := r.URL.Query().Get("force") == "true"
		if vmTarget == "" || imageRef == "" {
			http.Error(w, "Parámetros 'vm' e 'image' son obligatorios para eliminar", http.StatusBadRequest)
			return
		}
		if err := h.dockerSvc.DeleteImage(r.Context(), vmTarget, imageRef, force); err != nil {
			http.Error(w, fmt.Sprintf("Error eliminando imagen: %s", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imagen eliminada"})

	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleImageBuild(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req docker.BuildImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("JSON inválido: %s", err), http.StatusBadRequest)
		return
	}

	res, err := h.dockerSvc.BuildImage(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		outStr := ""
		if res != nil {
			outStr = res.Output
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"output": outStr,
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) handleImageItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	prefix := "/api/images/"
	sub := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(sub, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "ruta de imagen inválida (se espera /api/images/{vm}/{imageRef})", http.StatusBadRequest)
		return
	}
	vmName := parts[0]
	imageRef := parts[1]

	switch r.Method {
	case http.MethodDelete:
		force := r.URL.Query().Get("force") == "true"
		if err := h.dockerSvc.DeleteImage(r.Context(), vmName, imageRef, force); err != nil {
			http.Error(w, fmt.Sprintf("Error eliminando imagen: %s", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imagen eliminada"})
	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		vmTarget := strings.TrimSpace(r.URL.Query().Get("vm"))
		if vmTarget != "" {
			vols, err := h.dockerSvc.ListVolumes(r.Context(), vmTarget)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error listando volúmenes: %s", err), http.StatusInternalServerError)
				return
			}
			if vols == nil {
				vols = []docker.Volume{}
			}
			json.NewEncoder(w).Encode(vols)
			return
		}

		vms, err := h.vmSvc.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error listando VMs: %s", err), http.StatusInternalServerError)
			return
		}

		var allVols []docker.Volume
		for _, v := range vms {
			vols, err := h.dockerSvc.ListVolumes(r.Context(), v.Name)
			if err == nil && len(vols) > 0 {
				allVols = append(allVols, vols...)
			}
		}
		if allVols == nil {
			allVols = []docker.Volume{}
		}
		json.NewEncoder(w).Encode(allVols)

	case http.MethodPost:
		var req docker.CreateVolumeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("JSON inválido: %s", err), http.StatusBadRequest)
			return
		}

		res, err := h.dockerSvc.CreateVolume(r.Context(), req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creando volumen: %s", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(res)

	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleVolumeItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	prefix := "/api/volumes/"
	sub := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(sub, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "ruta de volumen inválida (se espera /api/volumes/{vm}/{volumeName})", http.StatusBadRequest)
		return
	}
	vmName := parts[0]
	volumeName := parts[1]

	switch r.Method {
	case http.MethodDelete:
		if err := h.dockerSvc.DeleteVolume(r.Context(), vmName, volumeName); err != nil {
			http.Error(w, fmt.Sprintf("Error eliminando volumen: %s", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Volumen eliminado"})
	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleNetworks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		vmName := r.URL.Query().Get("vm")
		if vmName == "" {
			var allNetworks []docker.Network
			vms, err := h.vmSvc.List()
			if err != nil {
				http.Error(w, fmt.Sprintf("Error listando VMs: %s", err), http.StatusInternalServerError)
				return
			}
			for _, v := range vms {
				nets, err := h.dockerSvc.ListNetworks(r.Context(), v.Name)
				if err == nil {
					allNetworks = append(allNetworks, nets...)
				}
			}
			if allNetworks == nil {
				allNetworks = []docker.Network{}
			}
			json.NewEncoder(w).Encode(allNetworks)
			return
		}

		nets, err := h.dockerSvc.ListNetworks(r.Context(), vmName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error listando redes: %s", err), http.StatusInternalServerError)
			return
		}
		if nets == nil {
			nets = []docker.Network{}
		}
		json.NewEncoder(w).Encode(nets)

	case http.MethodPost:
		var req docker.CreateNetworkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "cuerpo de solicitud inválido", http.StatusBadRequest)
			return
		}

		res, err := h.dockerSvc.CreateNetwork(r.Context(), req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creando red: %s", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(res)

	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleNetworkItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	prefix := "/api/networks/"
	sub := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(sub, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "ruta de red inválida (se espera /api/networks/{vm}/{networkName})", http.StatusBadRequest)
		return
	}
	vmName := parts[0]
	networkName := parts[1]

	switch r.Method {
	case http.MethodGet:
		netDetail, err := h.dockerSvc.InspectNetwork(r.Context(), vmName, networkName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error inspeccionando red: %s", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(netDetail)

	case http.MethodDelete:
		if err := h.dockerSvc.DeleteNetwork(r.Context(), vmName, networkName); err != nil {
			http.Error(w, fmt.Sprintf("Error eliminando red: %s", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Red eliminada"})

	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}


