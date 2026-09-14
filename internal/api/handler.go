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
	"github.com/feliren/ssh-web/internal/ssh"
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
	cfgMu   sync.RWMutex
	cfg     *config.Config
	log     *slog.Logger
	static  http.Handler
	vmStore *vmFileStore
	vmSvc   *vm.Service
}

// NewHandler inicializa el enrutador para archivos estaticos, configuracion y el endpoint WebSocket
func NewHandler(cfg *config.Config, log *slog.Logger, staticDir string) *Handler {
	vmStore := newVMFileStore(filepath.Join(".", "vms.json"))
	return &Handler{
		cfg:     cfg,
		log:     log,
		static:  http.FileServer(http.Dir(staticDir)),
		vmStore: vmStore,
		vmSvc:   vm.NewService(cfg, log),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/ws":
		h.handleWebSocket(w, r)
		return
	case r.URL.Path == "/api/config":
		h.handleConfig(w, r)
		return
	case r.URL.Path == "/api/vms":
		h.handleVirtualMachines(w, r)
		return
	case r.URL.Path == "/api/isos":
		h.handleISOs(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/vms/"):
		h.handleVirtualMachineByID(w, r)
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

		// Persistir cambios en config.properties
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

	isos := []string{}
	isoFile := filepath.Join(".", "isos.json")
	data, err := os.ReadFile(isoFile)
	if err == nil {
		var payload struct {
			ISOs []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"isos"`
		}
		if json.Unmarshal(data, &payload) == nil {
			for _, item := range payload.ISOs {
				if strings.TrimSpace(item.Name) != "" {
					isos = append(isos, item.Name)
				}
			}
		}
	}
	if len(isos) == 0 {
		isos = []string{
			"ubuntu-24.04.5-live-server-amd64.iso",
			"debian-13.5.0-amd64-netinst.iso",
		}
	}
	json.NewEncoder(w).Encode(map[string][]string{"isos": isos})
}

func (h *Handler) handleVirtualMachines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		vms, err := h.vmSvc.List()
		if err != nil {
			h.log.Warn("no se pudo listar maquinas virtuales por SSH", "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(vms)

	case http.MethodPost:
		var req ssh.VMCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"cuerpo de solicitud invalido"}`, http.StatusBadRequest)
			return
		}

		vm, err := h.vmSvc.Create(req)
		if err != nil {
			h.log.Warn("no se pudo crear maquina virtual por SSH", "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(vm)

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
	if decodedID == "" {
		http.Error(w, `{"error":"id requerido"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		vm, err := h.vmSvc.Get(decodedID)
		if err != nil {
			h.log.Warn("no se pudo consultar detalle de maquina virtual", "error", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(vm)

	case http.MethodDelete:
		if err := h.vmSvc.Delete(decodedID); err != nil {
			h.log.Warn("no se pudo eliminar maquina virtual por SSH", "error", err)
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

// handleWebSocket establece la sesion SSH y conecta los flujos bidireccionales con el navegador
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("fallo al actualizar la conexion a websocket", "error", err)
		return
	}

	h.log.Info("cliente websocket conectado", "remoto", r.RemoteAddr)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Obtener parametros de conexion desde query params con fallback a configuracion
	q := r.URL.Query()

	h.cfgMu.RLock()
	host := q.Get("host")
	if host == "" {
		host = h.cfg.SSHHost
	}

	port := h.cfg.SSHPort
	if pStr := q.Get("port"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			port = p
		}
	}

	user := q.Get("user")
	if user == "" {
		user = h.cfg.SSHUser
	}

	shell := q.Get("shell")
	if shell == "" {
		shell = h.cfg.SSHShell
	}

	startPath := q.Get("startPath")
	if startPath == "" {
		startPath = h.cfg.SSHStartPath
	}

	keyPath := q.Get("keyPath")
	if keyPath == "" {
		keyPath = h.cfg.SSHKeyPath
	}

	h.cfgMu.RUnlock()

	if host == "" || user == "" {
		h.log.Warn("intento de conexion con host o usuario vacio")
		conn.WriteJSON(wsMessage{
			Type: "error",
			Data: "Host o usuario no configurados. Abre el menú de configuración (⚙) en la esquina superior izquierda para ingresar los datos de conexión.",
		})
		conn.Close()
		return
	}

	session, err := ssh.NewSession(ctx, ssh.Config{
		Host:      host,
		Port:      port,
		User:      user,
		KeyPath:   keyPath,
		Shell:     shell,
		StartPath: startPath,
	}, h.log)
	if err != nil {
		h.log.Error("fallo en la conexion SSH", "error", err)
		conn.WriteJSON(wsMessage{Type: "error", Data: err.Error()})
		conn.Close()
		return
	}
	defer session.Close()

	var writeMu sync.Mutex

	// Reenvio inmediato de salida SSH hacia el canal WebSocket
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

	// Goroutine para envio periodico de paquetes Ping para mantener viva la conexion
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				h.log.Debug("goroutine de ping detenida")
				return
			case <-ticker.C:
				writeMu.Lock()
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				writeMu.Unlock()
				if err != nil {
					h.log.Debug("fallo en ping, cerrando conexion", "error", err)
					cancel()
					return
				}
			}
		}
	}()

	// Lectura de mensajes de entrada y eventos de redimensionamiento desde el cliente
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					h.log.Warn("error de lectura en websocket", "error", err)
				} else {
					h.log.Info("cliente websocket desconectado")
				}
				return
			}

			conn.SetReadDeadline(time.Now().Add(pongWait))

			var msg wsMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				if err := session.Write(message); err != nil {
					h.log.Error("error al escribir en stdin de SSH", "error", err)
					return
				}
				continue
			}

			switch msg.Type {
			case "input":
				if err := session.Write([]byte(msg.Data)); err != nil {
					h.log.Error("error al enviar entrada a SSH", "error", err)
					return
				}
			case "resize":
				if err := session.Resize(msg.Cols, msg.Rows); err != nil {
					h.log.Warn("error al redimensionar PTY", "error", err)
				}
			}
		}
	}()

	wg.Wait()

	writeMu.Lock()
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.Close()
	writeMu.Unlock()

	h.log.Info("sesion websocket finalizada y liberada", "remoto", r.RemoteAddr)
}
