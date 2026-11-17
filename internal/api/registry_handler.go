package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/feliren/ssh-web/internal/docker"
)

// handleRegistry enruta todas las peticiones bajo /api/registry/...
func (h *Handler) handleRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sub := strings.TrimPrefix(r.URL.Path, "/api/registry")
	sub = strings.Trim(sub, "/")

	switch sub {
	case "", "status":
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		overview, err := h.registrySvc.GetOverview(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(overview)

	case "deploy":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		var req docker.RegistryDeployRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON de solicitud inválido"})
			return
		}
		st, err := h.registrySvc.DeployInstance(r.Context(), req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(st)

	case "action":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		var req docker.RegistryControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON de solicitud inválido"})
			return
		}
		st, err := h.registrySvc.ControlInstance(r.Context(), req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(st)

	case "logs":
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		target := r.URL.Query().Get("target")
		tail := 200
		if rawTail := r.URL.Query().Get("tail"); rawTail != "" {
			if n, err := strconv.Atoi(rawTail); err == nil && n > 0 {
				tail = n
			}
		}
		logs, err := h.registrySvc.GetInstanceLogs(r.Context(), target, tail)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error(), "logs": logs})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"logs": logs})

	case "images":
		switch r.Method {
		case http.MethodGet:
			target := r.URL.Query().Get("target")
			res, err := h.registrySvc.ListRegistryImages(r.Context(), target)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(res)

		case http.MethodDelete:
			repo := r.URL.Query().Get("repository")
			tag := r.URL.Query().Get("tag")
			force := r.URL.Query().Get("force") == "true" || r.URL.Query().Get("force") == "1"
			if repo == "" || tag == "" {
				var body struct {
					Repository string `json:"repository"`
					Tag        string `json:"tag"`
					Force      bool   `json:"force"`
				}
				if json.NewDecoder(r.Body).Decode(&body) == nil {
					if repo == "" {
						repo = body.Repository
					}
					if tag == "" {
						tag = body.Tag
					}
					if body.Force {
						force = true
					}
				}
			}
			if err := h.registrySvc.DeleteRegistryImage(r.Context(), repo, tag, force); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Imagen eliminada del Registry local",
			})

		default:
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
		}

	case "push":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		var req docker.RegistryPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON de solicitud inválido"})
			return
		}
		res, err := h.registrySvc.PushToLocalRegistry(r.Context(), req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			out := ""
			if res != nil {
				out = res.Output
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  err.Error(),
				"output": out,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(res)

	case "pull":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		var req docker.RegistryPullRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON de solicitud inválido"})
			return
		}
		res, err := h.registrySvc.PullFromLocalRegistry(r.Context(), req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			out := ""
			if res != nil {
				out = res.Output
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  err.Error(),
				"output": out,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(res)

	case "smart-pull":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		var req docker.SmartPullRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON de solicitud inválido"})
			return
		}
		res, err := h.registrySvc.SmartPullImage(r.Context(), req)
		if err != nil {
			if res != nil && res.LocalRegistryUnreachable {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(res)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			out := ""
			if res != nil {
				out = res.Output
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  err.Error(),
				"output": out,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(res)

	case "nodes/configure":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"metodo no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		var req docker.ConfigureNodesRegistryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON de solicitud inválido"})
			return
		}
		res, err := h.registrySvc.ConfigureNodesMirror(r.Context(), req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(res)

	default:
		http.Error(w, `{"error":"ruta de registry no encontrada"}`, http.StatusNotFound)
	}
}
