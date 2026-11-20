package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strings"
	"time"
)

const registryAcceptHeaders = "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json"

// RegistryService administra el Registry local escribible, el Pull-through Cache de Docker Hub y la configuración de mirrors en los nodos.
type RegistryService struct {
	runner     CommandRunner
	vmProvider SwarmVMProvider
	store      *RegistryStore
	log        *slog.Logger
}

// NewRegistryService crea una nueva instancia de RegistryService.
func NewRegistryService(runner CommandRunner, vmProvider SwarmVMProvider, store *RegistryStore, logger *slog.Logger) *RegistryService {
	if logger == nil {
		logger = slog.Default()
	}
	if store == nil {
		store = NewRegistryStore("registry.json")
	}
	return &RegistryService{
		runner:     runner,
		vmProvider: vmProvider,
		store:      store,
		log:        logger,
	}
}

// Store devuelve el almacén de persistencia de Registry.
func (s *RegistryService) Store() *RegistryStore {
	return s.store
}

// MergeDaemonJSON combina de forma segura `registry-mirrors` e `insecure-registries` dentro del JSON existente de `/etc/docker/daemon.json`,
// preservando todas las demás propiedades previamente configuradas en el nodo.
func MergeDaemonJSON(existingRaw string, mirrorsToAdd []string, insecureToAdd []string) (string, bool, error) {
	trimmed := strings.TrimSpace(existingRaw)
	doc := make(map[string]interface{})

	if trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
			return "", false, fmt.Errorf("el archivo /etc/docker/daemon.json existente contiene JSON inválido: %w", err)
		}
	}

	changed := false

	mergeStringSliceField := func(field string, toAdd []string) {
		var current []string
		if rawVal, exists := doc[field]; exists && rawVal != nil {
			switch v := rawVal.(type) {
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
						current = append(current, strings.TrimSpace(str))
					}
				}
			case []string:
				for _, str := range v {
					if strings.TrimSpace(str) != "" {
						current = append(current, strings.TrimSpace(str))
					}
				}
			}
		}

		seen := make(map[string]bool, len(current))
		for _, item := range current {
			seen[strings.ToLower(item)] = true
		}

		for _, item := range toAdd {
			clean := strings.TrimSpace(item)
			if clean == "" {
				continue
			}
			key := strings.ToLower(clean)
			if !seen[key] {
				current = append(current, clean)
				seen[key] = true
				changed = true
			}
		}

		if len(current) > 0 {
			doc[field] = current
		}
	}

	if len(mirrorsToAdd) > 0 {
		mergeStringSliceField("registry-mirrors", mirrorsToAdd)
	}
	if len(insecureToAdd) > 0 {
		mergeStringSliceField("insecure-registries", insecureToAdd)
	}

	outBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", false, fmt.Errorf("serializando daemon.json: %w", err)
	}
	return string(outBytes) + "\n", changed, nil
}

// ParseImageReference separa una referencia como "mysql:8.4", "mi-org/api:v1" o "nginx" en (repositorio, tag).
func ParseImageReference(ref string) (string, string, error) {
	clean := strings.TrimSpace(ref)
	if clean == "" {
		return "", "", fmt.Errorf("la referencia de la imagen no puede estar vacía")
	}
	if dangerousCharsRegex.MatchString(clean) {
		return "", "", fmt.Errorf("la referencia de la imagen contiene caracteres no permitidos: %s", clean)
	}

	// Si incluye digest @sha256:..., separar antes
	if atIdx := strings.Index(clean, "@"); atIdx != -1 {
		clean = clean[:atIdx]
	}

	lastSlash := strings.LastIndex(clean, "/")
	lastColon := strings.LastIndex(clean, ":")

	var repo, tag string
	if lastColon > lastSlash {
		repo = strings.TrimSpace(clean[:lastColon])
		tag = strings.TrimSpace(clean[lastColon+1:])
	} else {
		repo = clean
		tag = "latest"
	}

	if tag == "" {
		tag = "latest"
	}
	if err := ValidateImageRepository(repo); err != nil {
		return "", "", err
	}
	if err := ValidateImageTag(tag); err != nil {
		return "", "", err
	}
	return repo, tag, nil
}

func defaultInstanceConfig(target string) RegistryInstanceConfig {
	if strings.EqualFold(target, RegistryTargetCache) {
		return RegistryInstanceConfig{
			Target:        RegistryTargetCache,
			Port:          DefaultCacheRegistryPort,
			ContainerPort: 5000,
			ContainerName: DefaultCacheRegistryContainer,
			VolumeName:    DefaultCacheRegistryVolume,
			Image:         DefaultRegistryImage,
			RemoteURL:     DefaultCacheRemoteURL,
		}
	}
	return RegistryInstanceConfig{
		Target:        RegistryTargetLocal,
		Port:          DefaultLocalRegistryPort,
		ContainerPort: 5000,
		ContainerName: DefaultLocalRegistryContainer,
		VolumeName:    DefaultLocalRegistryVolume,
		Image:         DefaultRegistryImage,
	}
}

func (s *RegistryService) resolveVMAccessibleIP(ctx context.Context, vmName string, preferredIP string) (SwarmVMInfo, string, error) {
	vmInfo, err := s.vmProvider.GetVM(vmName)
	if err != nil {
		return SwarmVMInfo{}, "", fmt.Errorf("no se encontró la máquina virtual %q: %w", vmName, err)
	}
	if !isVMRunningState(vmInfo.State) {
		return vmInfo, "", fmt.Errorf("la máquina virtual %q está apagada (estado: %s); iníciala antes de desplegar el Registry", vmInfo.Name, vmInfo.State)
	}

	ip := strings.TrimSpace(preferredIP)
	if ip == "" {
		ip = strings.TrimSpace(vmInfo.PrivateIP)
	}
	if ip == "" {
		ip = strings.TrimSpace(vmInfo.IP)
	}
	if ip == "" || strings.HasPrefix(ip, "10.0.2.") || strings.HasPrefix(ip, "10.0.3.") {
		if refreshed, netErr := s.vmProvider.EnsureVMNetwork(ctx, vmInfo.Name); netErr == nil {
			vmInfo = refreshed
			if strings.TrimSpace(preferredIP) == "" {
				ip = strings.TrimSpace(refreshed.PrivateIP)
				if ip == "" {
					ip = strings.TrimSpace(refreshed.IP)
				}
			}
		}
	}

	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil || parsed.IsLoopback() || parsed.IsUnspecified() {
		return vmInfo, "", fmt.Errorf("la VM %q no tiene una dirección IPv4 válida accesible desde los demás nodos (%q)", vmInfo.Name, ip)
	}
	return vmInfo, ip, nil
}

// GetOverview devuelve el estado actual del Registry local, del Pull-through Cache y de todos los nodos Docker.
func (s *RegistryService) GetOverview(ctx context.Context) (*RegistryOverviewResponse, error) {
	persisted, _ := s.store.Load()

	vms, err := s.vmProvider.ListVMs()
	if err != nil {
		vms = nil
	}
	vmByName := make(map[string]SwarmVMInfo, len(vms))
	for _, v := range vms {
		vmByName[strings.ToLower(v.Name)] = v
	}

	localStatus := s.inspectInstanceStatus(ctx, RegistryTargetLocal, persisted.Local, vmByName)
	cacheStatus := s.inspectInstanceStatus(ctx, RegistryTargetCache, persisted.Cache, vmByName)
	nodes := s.inspectNodesRegistryStatus(ctx, vms, localStatus, cacheStatus)

	return &RegistryOverviewResponse{
		Local: localStatus,
		Cache: cacheStatus,
		Nodes: nodes,
	}, nil
}

func (s *RegistryService) inspectInstanceStatus(ctx context.Context, target string, cfg *RegistryInstanceConfig, vmByName map[string]SwarmVMInfo) RegistryInstanceStatus {
	def := defaultInstanceConfig(target)
	if cfg == nil || strings.TrimSpace(cfg.VMName) == "" {
		return RegistryInstanceStatus{
			Target:         def.Target,
			Configured:     false,
			Port:           def.Port,
			ContainerPort:  def.ContainerPort,
			ContainerName:  def.ContainerName,
			ContainerState: "not-deployed",
			VolumeName:     def.VolumeName,
			Image:          def.Image,
			RemoteURL:      def.RemoteURL,
			StatusText:     "No desplegado",
		}
	}

	port := cfg.Port
	if port <= 0 {
		port = def.Port
	}
	cPort := cfg.ContainerPort
	if cPort <= 0 {
		cPort = 5000
	}
	cName := strings.TrimSpace(cfg.ContainerName)
	if cName == "" {
		cName = def.ContainerName
	}
	volName := strings.TrimSpace(cfg.VolumeName)
	if volName == "" {
		volName = def.VolumeName
	}
	img := strings.TrimSpace(cfg.Image)
	if img == "" {
		img = def.Image
	}

	status := RegistryInstanceStatus{
		Target:        def.Target,
		Configured:    true,
		VMID:          cfg.VMID,
		VMName:        cfg.VMName,
		HostIP:        cfg.HostIP,
		Port:          port,
		ContainerPort: cPort,
		ContainerName: cName,
		VolumeName:    volName,
		Image:         img,
		RemoteURL:     cfg.RemoteURL,
		UpdatedAt:     cfg.UpdatedAt,
	}

	vmInfo, foundVM := vmByName[strings.ToLower(cfg.VMName)]
	if !foundVM {
		if gotVM, err := s.vmProvider.GetVM(cfg.VMName); err == nil {
			vmInfo = gotVM
			foundVM = true
		}
	}

	if foundVM {
		status.VMID = vmInfo.ID
		status.VMName = vmInfo.Name
		status.VMState = vmInfo.State
		if vmInfo.PrivateIP != "" && (status.HostIP == "" || strings.HasPrefix(status.HostIP, "10.0.2.") || strings.HasPrefix(status.HostIP, "10.0.3.")) {
			status.HostIP = vmInfo.PrivateIP
		}
	} else {
		status.VMState = "missing"
	}

	if status.HostIP != "" && status.Port > 0 {
		status.EndpointHost = fmt.Sprintf("%s:%d", status.HostIP, status.Port)
		status.EndpointURL = fmt.Sprintf("http://%s:%d", status.HostIP, status.Port)
		status.APIURL = fmt.Sprintf("http://%s:%d/v2/", status.HostIP, status.Port)
	}

	if !isVMRunningState(status.VMState) {
		status.ContainerState = "vm-stopped"
		status.Healthy = false
		status.StatusText = fmt.Sprintf("VM '%s' apagada", status.VMName)
		return status
	}

	// Consultar en una sola llamada SSH el estado del contenedor y si el endpoint HTTP /v2/ responde 200
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	probeScript := fmt.Sprintf(
		`C_INFO=$(sudo docker inspect --format '{{.Id}}|{{.State.Status}}|{{.HostConfig.RestartPolicy.Name}}' %s 2>/dev/null || echo "MISSING"); `+
			`HTTP_CODE=$(curl -s -o /dev/null -w '%%{http_code}' --max-time 3 http://127.0.0.1:%d/v2/ 2>/dev/null || echo "000"); `+
			`echo "PROBE|$C_INFO|$HTTP_CODE"`,
		EscapeShellArg(cName),
		port,
	)

	out, err := s.runner(cmdCtx, status.VMName, probeScript)
	if err != nil {
		status.ContainerState = "unknown"
		status.Healthy = false
		status.StatusText = "Sin respuesta SSH"
		status.Error = err.Error()
		return status
	}

	for _, rawLine := range strings.Split(strings.TrimSpace(out), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "PROBE|") {
			continue
		}
		rest := strings.TrimPrefix(line, "PROBE|")
		if strings.HasPrefix(rest, "MISSING|") {
			status.ContainerState = "missing"
			status.Healthy = false
			status.StatusText = "Contenedor no encontrado"
			return status
		}
		parts := strings.Split(rest, "|")
		if len(parts) >= 4 {
			cID := strings.TrimSpace(parts[0])
			if len(cID) > 12 {
				cID = cID[:12]
			}
			status.ContainerID = cID
			status.ContainerState = strings.ToLower(strings.TrimSpace(parts[1]))
			status.RestartPolicy = strings.TrimSpace(parts[2])
			httpCode := strings.TrimSpace(parts[3])
			status.Healthy = (status.ContainerState == "running") && (httpCode == "200" || httpCode == "401")
		}
	}

	switch {
	case status.ContainerState == "running" && status.Healthy:
		status.StatusText = "Activo (HTTP V2 OK)"
	case status.ContainerState == "running" && !status.Healthy:
		status.StatusText = "En ejecución (esperando endpoint /v2/)"
	case status.ContainerState == "exited" || status.ContainerState == "created" || status.ContainerState == "stopped":
		status.StatusText = "Detenido"
	default:
		status.StatusText = status.ContainerState
	}
	return status
}

func (s *RegistryService) inspectNodesRegistryStatus(ctx context.Context, vms []SwarmVMInfo, local RegistryInstanceStatus, cache RegistryInstanceStatus) []NodeRegistryConfigStatus {
	expectedMirror := ""
	if cache.Configured && cache.EndpointURL != "" {
		expectedMirror = strings.ToLower(strings.TrimRight(cache.EndpointURL, "/"))
	}
	expectedLocalInsecure := strings.ToLower(strings.TrimSpace(local.EndpointHost))
	expectedCacheInsecure := strings.ToLower(strings.TrimSpace(cache.EndpointHost))

	nodes := make([]NodeRegistryConfigStatus, 0, len(vms))
	for _, v := range vms {
		privIP := strings.TrimSpace(v.PrivateIP)
		if privIP == "" {
			privIP = strings.TrimSpace(v.IP)
		}
		item := NodeRegistryConfigStatus{
			VMID:      v.ID,
			VMName:    v.Name,
			VMState:   v.State,
			PrivateIP: privIP,
		}

		if !isVMRunningState(v.State) {
			item.NeedsConfiguration = true
			item.StatusLabel = "VM Apagada"
			nodes = append(nodes, item)
			continue
		}

		cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		rawDaemon, err := s.runner(cmdCtx, v.Name, "sudo cat /etc/docker/daemon.json 2>/dev/null || echo '{}'")
		cancel()

		if err != nil {
			item.NeedsConfiguration = true
			item.StatusLabel = "Sin conexión SSH"
			item.Error = err.Error()
			nodes = append(nodes, item)
			continue
		}

		var parsed struct {
			RegistryMirrors    []string `json:"registry-mirrors"`
			InsecureRegistries []string `json:"insecure-registries"`
		}
		_ = json.Unmarshal([]byte(strings.TrimSpace(rawDaemon)), &parsed)

		item.ConfiguredMirrors = parsed.RegistryMirrors
		item.InsecureRegistries = parsed.InsecureRegistries

		for _, m := range parsed.RegistryMirrors {
			cleanM := strings.ToLower(strings.TrimRight(strings.TrimSpace(m), "/"))
			if expectedMirror != "" && cleanM == expectedMirror {
				item.MirrorConfigured = true
			}
		}
		for _, ir := range parsed.InsecureRegistries {
			cleanIR := strings.ToLower(strings.TrimSpace(ir))
			if expectedLocalInsecure != "" && cleanIR == expectedLocalInsecure {
				item.LocalInsecure = true
			}
			if expectedCacheInsecure != "" && cleanIR == expectedCacheInsecure {
				item.CacheInsecure = true
			}
		}

		needs := false
		if expectedMirror != "" && !item.MirrorConfigured {
			needs = true
		}
		if expectedLocalInsecure != "" && !item.LocalInsecure {
			needs = true
		}
		if expectedCacheInsecure != "" && !item.CacheInsecure {
			needs = true
		}
		item.NeedsConfiguration = needs

		switch {
		case expectedMirror != "" && item.MirrorConfigured && (!local.Configured || item.LocalInsecure):
			item.StatusLabel = "Mirror y HTTP configurados"
		case expectedMirror != "" && item.MirrorConfigured:
			item.StatusLabel = "Mirror activo"
		case expectedLocalInsecure != "" && item.LocalInsecure && !cache.Configured:
			item.StatusLabel = "Registry HTTP confiado"
		default:
			item.StatusLabel = "Pendiente de configurar"
		}

		nodes = append(nodes, item)
	}
	return nodes
}

// DeployInstance despliega de forma idempotente el Registry local escribible ("local") o el Pull-through Cache ("cache").
func (s *RegistryService) DeployInstance(ctx context.Context, req RegistryDeployRequest) (*RegistryInstanceStatus, error) {
	target := strings.ToLower(strings.TrimSpace(req.Target))
	if target != RegistryTargetCache {
		target = RegistryTargetLocal
	}
	def := defaultInstanceConfig(target)

	vmName := strings.TrimSpace(req.VMName)
	if vmName == "" {
		return nil, fmt.Errorf("debes seleccionar una máquina virtual de destino")
	}

	port := req.Port
	if port <= 0 {
		port = def.Port
	}
	if port < 1024 || port > 65535 {
		return nil, fmt.Errorf("el puerto %d no es válido (debe estar entre 1024 y 65535)", port)
	}

	cName := strings.TrimSpace(req.ContainerName)
	if cName == "" {
		cName = def.ContainerName
	}
	if err := ValidateContainerName(cName); err != nil {
		return nil, err
	}

	volName := strings.TrimSpace(req.VolumeName)
	if volName == "" {
		volName = def.VolumeName
	}
	if err := ValidateVolumeName(volName); err != nil {
		return nil, err
	}

	remoteURL := strings.TrimSpace(req.RemoteURL)
	if target == RegistryTargetCache {
		if remoteURL == "" {
			remoteURL = DefaultCacheRemoteURL
		}
	} else {
		remoteURL = ""
	}

	// Verificar que no colisione con la otra instancia (local vs cache)
	persisted, _ := s.store.Load()
	other := persisted.Cache
	if target == RegistryTargetCache {
		other = persisted.Local
	}
	if other != nil && strings.EqualFold(other.VMName, vmName) {
		if other.Port == port {
			return nil, fmt.Errorf("el puerto %d en la VM %q ya está asignado a la otra instancia de Registry (%s); utiliza puertos distintos (ej. 5000 y 5001)", port, vmName, other.ContainerName)
		}
		if strings.EqualFold(other.ContainerName, cName) {
			return nil, fmt.Errorf("el nombre de contenedor %q ya está en uso por la otra instancia de Registry", cName)
		}
		if strings.EqualFold(other.VolumeName, volName) {
			return nil, fmt.Errorf("el volumen %q ya está en uso por la otra instancia de Registry; ambas instancias deben usar volúmenes independientes", volName)
		}
	}

	vmInfo, hostIP, err := s.resolveVMAccessibleIP(ctx, vmName, req.HostIP)
	if err != nil {
		return nil, err
	}

	endpointHost := fmt.Sprintf("%s:%d", hostIP, port)

	// Asegurar que la VM anfitriona confíe en el Registry HTTP antes de operar
	_ = s.ensureNodeDaemonConfig(ctx, vmInfo.Name, nil, []string{endpointHost}, false)

	// Script idempotente para crear volumen persistente y desplegar/iniciar el contenedor oficial registry:2.8.3
	envFlags := "-e REGISTRY_STORAGE_DELETE_ENABLED=true"
	if target == RegistryTargetCache {
		envFlags += " -e " + EscapeShellArg("REGISTRY_PROXY_REMOTEURL="+remoteURL)
	}

	deployScript := fmt.Sprintf(`set -e
VOL_NAME=%s
C_NAME=%s
IMAGE=%s
HOST_PORT=%d

if ! sudo docker volume inspect "$VOL_NAME" >/dev/null 2>&1; then
    sudo docker volume create "$VOL_NAME" >/dev/null
fi

NEED_CREATE=1
if sudo docker inspect "$C_NAME" >/dev/null 2>&1; then
    CUR_PORT=$(sudo docker inspect --format '{{(index (index .HostConfig.PortBindings "5000/tcp") 0).HostPort}}' "$C_NAME" 2>/dev/null || echo "")
    CUR_ENV=$(sudo docker inspect --format '{{json .Config.Env}}' "$C_NAME" 2>/dev/null || echo "")
    if [ "$CUR_PORT" = "$HOST_PORT" ]; then
        if [ "%s" = "cache" ] && echo "$CUR_ENV" | grep -q "REGISTRY_PROXY_REMOTEURL"; then
            NEED_CREATE=0
        elif [ "%s" = "local" ] && ! echo "$CUR_ENV" | grep -q "REGISTRY_PROXY_REMOTEURL"; then
            NEED_CREATE=0
        fi
    fi
    if [ "$NEED_CREATE" = "1" ]; then
        sudo docker rm -f "$C_NAME" >/dev/null 2>&1 || true
    fi
fi

if [ "$NEED_CREATE" = "1" ]; then
    sudo docker run -d \
        --name "$C_NAME" \
        --restart unless-stopped \
        -p "${HOST_PORT}:5000" \
        -v "${VOL_NAME}:/var/lib/registry" \
        %s \
        "$IMAGE" >/dev/null
else
    sudo docker start "$C_NAME" >/dev/null 2>&1 || true
fi

# Esperar hasta 10s a que el endpoint V2 responda HTTP 200
HTTP_OK=0
for i in 1 2 3 4 5; do
    CODE=$(curl -s -o /dev/null -w '%%{http_code}' --max-time 3 "http://127.0.0.1:${HOST_PORT}/v2/" 2>/dev/null || echo "000")
    if [ "$CODE" = "200" ] || [ "$CODE" = "401" ]; then
        HTTP_OK=1
        break
    fi
    sleep 1
done
echo "DEPLOY_OK|$HTTP_OK"
`,
		EscapeShellArg(volName),
		EscapeShellArg(cName),
		EscapeShellArg(DefaultRegistryImage),
		port,
		target,
		target,
		envFlags,
	)

	s.log.Info("Desplegando instancia de Registry en VM",
		"target", target,
		"vm", vmInfo.Name,
		"ip", hostIP,
		"port", port,
		"container", cName,
		"volume", volName,
	)

	out, runErr := s.runner(ctx, vmInfo.Name, deployScript)
	if runErr != nil {
		return nil, fmt.Errorf("error desplegando %s en la VM %q: %s (%w)", cName, vmInfo.Name, strings.TrimSpace(out), runErr)
	}

	nowISO := time.Now().Format(time.RFC3339)
	cfg := RegistryInstanceConfig{
		Target:        target,
		VMID:          vmInfo.ID,
		VMName:        vmInfo.Name,
		HostIP:        hostIP,
		Port:          port,
		ContainerPort: 5000,
		ContainerName: cName,
		VolumeName:    volName,
		Image:         DefaultRegistryImage,
		RemoteURL:     remoteURL,
		CreatedAt:     nowISO,
		UpdatedAt:     nowISO,
	}
	if err := s.store.SetInstance(cfg); err != nil {
		return nil, fmt.Errorf("instancia desplegada pero no se pudo guardar en registry.json: %w", err)
	}

	// Configurar automáticamente insecure-registries (HTTP sin TLS) en todas las VMs encendidas mediante reload sin interrumpir contenedores
	if vms, vErr := s.vmProvider.ListVMs(); vErr == nil {
		for _, v := range vms {
			if isVMRunningState(v.State) {
				_ = s.ensureNodeDaemonConfig(ctx, v.Name, nil, []string{endpointHost}, false)
			}
		}
	}

	status := s.inspectInstanceStatus(ctx, target, &cfg, map[string]SwarmVMInfo{strings.ToLower(vmInfo.Name): vmInfo})
	return &status, nil
}

// ControlInstance permite iniciar, detener, reiniciar o eliminar el contenedor del Registry local o de la Caché.
func (s *RegistryService) ControlInstance(ctx context.Context, req RegistryControlRequest) (*RegistryInstanceStatus, error) {
	target := strings.ToLower(strings.TrimSpace(req.Target))
	if target != RegistryTargetCache {
		target = RegistryTargetLocal
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "start" && action != "stop" && action != "restart" && action != "remove" {
		return nil, fmt.Errorf("acción %q no soportada (use start, stop, restart o remove)", req.Action)
	}

	persisted, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	cfg := persisted.Local
	if target == RegistryTargetCache {
		cfg = persisted.Cache
	}
	if cfg == nil || strings.TrimSpace(cfg.VMName) == "" {
		return nil, fmt.Errorf("la instancia %q aún no está desplegada", target)
	}

	var cmd string
	switch action {
	case "start":
		cmd = fmt.Sprintf("sudo docker start %s", EscapeShellArg(cfg.ContainerName))
	case "stop":
		cmd = fmt.Sprintf("sudo docker stop %s", EscapeShellArg(cfg.ContainerName))
	case "restart":
		cmd = fmt.Sprintf("sudo docker restart %s", EscapeShellArg(cfg.ContainerName))
	case "remove":
		cmd = fmt.Sprintf("sudo docker rm -f %s 2>/dev/null || true", EscapeShellArg(cfg.ContainerName))
	}

	if out, runErr := s.runner(ctx, cfg.VMName, cmd); runErr != nil {
		return nil, fmt.Errorf("error ejecutando %s en %s (%s): %s (%w)", action, cfg.ContainerName, cfg.VMName, strings.TrimSpace(out), runErr)
	}

	if action == "remove" {
		_ = s.store.ClearInstance(target)
		st := s.inspectInstanceStatus(ctx, target, nil, nil)
		return &st, nil
	}

	cfg.UpdatedAt = time.Now().Format(time.RFC3339)
	_ = s.store.SetInstance(*cfg)
	st := s.inspectInstanceStatus(ctx, target, cfg, nil)
	return &st, nil
}

// GetInstanceLogs obtiene los logs recientes del contenedor del Registry local o de la Caché.
func (s *RegistryService) GetInstanceLogs(ctx context.Context, target string, tail int) (string, error) {
	if tail <= 0 || tail > 2000 {
		tail = 200
	}
	persisted, err := s.store.Load()
	if err != nil {
		return "", err
	}
	cfg := persisted.Local
	if strings.EqualFold(target, RegistryTargetCache) {
		cfg = persisted.Cache
	}
	if cfg == nil || strings.TrimSpace(cfg.VMName) == "" {
		return "", fmt.Errorf("la instancia %q no está desplegada", target)
	}

	cmd := fmt.Sprintf("sudo docker logs --tail %d %s 2>&1", tail, EscapeShellArg(cfg.ContainerName))
	out, runErr := s.runner(ctx, cfg.VMName, cmd)
	if runErr != nil {
		return out, fmt.Errorf("error obteniendo logs de %s en %s: %w", cfg.ContainerName, cfg.VMName, runErr)
	}
	return out, nil
}

// ListRegistryImages consulta mediante la API HTTP V2 todos los repositorios, tags, digests y tamaños almacenados en el Registry indicado.
func (s *RegistryService) ListRegistryImages(ctx context.Context, target string) (*RegistryImagesResponse, error) {
	if strings.EqualFold(target, RegistryTargetCache) {
		target = RegistryTargetCache
	} else {
		target = RegistryTargetLocal
	}

	persisted, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	cfg := persisted.Local
	if target == RegistryTargetCache {
		cfg = persisted.Cache
	}
	if cfg == nil || strings.TrimSpace(cfg.VMName) == "" {
		return &RegistryImagesResponse{
			Target:       target,
			Repositories: []string{},
			Images:       []RegistryImageItem{},
			Error:        "El Registry aún no ha sido desplegado",
		}, nil
	}

	endpointHost := fmt.Sprintf("%s:%d", cfg.HostIP, cfg.Port)
	endpointURL := fmt.Sprintf("http://%s:%d", cfg.HostIP, cfg.Port)

	// Consultar el catálogo V2 desde la VM donde corre el Registry
	cmdCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	// Intentar primero inspección directa del almacenamiento local del contenedor (/var/lib/registry).
	// Esto es indispensable para Pull-through Cache (donde /v2/_catalog omite repositorios cuyo cliente ya tenía las capas
	// y donde /v2/<repo>/tags/list consulta todos los cientos de tags remotos de Docker Hub causando timeouts).
	if strings.TrimSpace(cfg.ContainerName) != "" {
		if fsRes, ok := s.listRegistryImagesFromContainerFS(cmdCtx, target, cfg, endpointHost, endpointURL); ok {
			return fsRes, nil
		}
	}

	catCmd := fmt.Sprintf(`curl -fsS --max-time 8 "http://127.0.0.1:%d/v2/_catalog?n=500"`, cfg.Port)
	catOut, catErr := s.runner(cmdCtx, cfg.VMName, catCmd)
	if catErr != nil {
		return &RegistryImagesResponse{
			Target:       target,
			EndpointHost: endpointHost,
			EndpointURL:  endpointURL,
			Healthy:      false,
			Repositories: []string{},
			Images:       []RegistryImageItem{},
			Error:        fmt.Sprintf("El endpoint V2 (%s/v2/) no responde: %v", endpointURL, catErr),
		}, nil
	}

	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(catOut)), &catalog); err != nil {
		return nil, fmt.Errorf("respuesta inválida de /v2/_catalog: %w", err)
	}

	sort.Strings(catalog.Repositories)
	var allImages []RegistryImageItem

	for _, repo := range catalog.Repositories {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}
		tagsCmd := fmt.Sprintf(`curl -fsS --max-time 6 "http://127.0.0.1:%d/v2/%s/tags/list"`, cfg.Port, repo)
		tagsOut, tagsErr := s.runner(cmdCtx, cfg.VMName, tagsCmd)
		if tagsErr != nil {
			continue
		}
		var tagList struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(tagsOut)), &tagList) != nil || len(tagList.Tags) == 0 {
			continue
		}
		sort.Strings(tagList.Tags)

		// Mapa digest -> tags dentro del mismo repositorio para detectar tags compartidos
		digestToTags := make(map[string][]string)
		repoItems := make([]RegistryImageItem, 0, len(tagList.Tags))

		for _, tag := range tagList.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			item := s.inspectManifestV2(cmdCtx, cfg.VMName, cfg.Port, endpointHost, repo, tag)
			if item.Digest != "" {
				digestToTags[item.Digest] = append(digestToTags[item.Digest], tag)
			}
			repoItems = append(repoItems, item)
		}

		for i := range repoItems {
			d := repoItems[i].Digest
			if d != "" && len(digestToTags[d]) > 1 {
				for _, otherTag := range digestToTags[d] {
					if otherTag != repoItems[i].Tag {
						repoItems[i].SharedTags = append(repoItems[i].SharedTags, otherTag)
					}
				}
			}
			allImages = append(allImages, repoItems[i])
		}
	}

	if allImages == nil {
		allImages = []RegistryImageItem{}
	}
	if catalog.Repositories == nil {
		catalog.Repositories = []string{}
	}

	return &RegistryImagesResponse{
		Target:       target,
		EndpointHost: endpointHost,
		EndpointURL:  endpointURL,
		Healthy:      true,
		Repositories: catalog.Repositories,
		Images:       allImages,
	}, nil
}

// listRegistryImagesFromContainerFS inspecciona en una sola llamada SSH los repositorios, tags y manifiestos
// realmente almacenados en el volumen del contenedor (/var/lib/registry/docker/registry/v2).
func (s *RegistryService) listRegistryImagesFromContainerFS(
	ctx context.Context,
	target string,
	cfg *RegistryInstanceConfig,
	endpointHost, endpointURL string,
) (*RegistryImagesResponse, bool) {
	scanScript := fmt.Sprintf(`HTTP_CODE=$(curl -s -o /dev/null -w '%%{http_code}' --max-time 5 "http://127.0.0.1:%d/v2/" 2>/dev/null || echo "000")
echo "HTTP_V2|$HTTP_CODE"
if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "401" ]; then
    echo "FS_SCAN_DONE"
    exit 0
fi
sudo docker exec %s sh -c '
ROOT="/var/lib/registry/docker/registry/v2"
if [ -d "$ROOT/repositories" ]; then
    for link_file in $(find "$ROOT/repositories" -type f -name "link" 2>/dev/null | grep "/_manifests/tags/.*/current/link$"); do
        rel="${link_file#$ROOT/repositories/}"
        repo="${rel%%%%/_manifests/tags/*}"
        rest="${rel#*/_manifests/tags/}"
        tag="${rest%%%%/current/link}"
        digest=$(cat "$link_file" 2>/dev/null | tr -d " \t\r\n")
        if [ -z "$digest" ]; then continue; fi
        hex="${digest#sha256:}"
        if [ ! -f "$ROOT/repositories/$repo/_manifests/revisions/sha256/$hex/link" ]; then continue; fi
        p2=$(echo "$hex" | cut -c1-2)
        blob="$ROOT/blobs/sha256/$p2/$hex/data"
        b64=""
        if [ -f "$blob" ]; then
            b64=$(base64 "$blob" 2>/dev/null | tr -d "\r\n")
        fi
        has_layers=0
        if [ -d "$ROOT/repositories/$repo/_layers" ]; then
            has_layers=1
        fi
        echo "FS_ITEM|$repo|$tag|$digest|$has_layers|$b64"
    done
fi
echo "FS_SCAN_DONE"
' 2>/dev/null || true
`, cfg.Port, EscapeShellArg(cfg.ContainerName))

	out, err := s.runner(ctx, cfg.VMName, scanScript)
	if err != nil || !strings.Contains(out, "FS_SCAN_DONE") {
		return nil, false
	}

	httpCode := "000"
	type rawFSItem struct {
		repo          string
		tag           string
		digest        string
		hasLayers     bool
		manifestBytes []byte
	}
	byRepo := make(map[string][]rawFSItem)
	repoSet := make(map[string]bool)

	for _, rawLine := range strings.Split(strings.TrimSpace(out), "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "HTTP_V2|") {
			httpCode = strings.TrimSpace(strings.TrimPrefix(line, "HTTP_V2|"))
			continue
		}
		if !strings.HasPrefix(line, "FS_ITEM|") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "FS_ITEM|"), "|", 5)
		if len(parts) < 5 {
			continue
		}
		repo := strings.TrimSpace(parts[0])
		tag := strings.TrimSpace(parts[1])
		digest := strings.TrimSpace(parts[2])
		hasLayers := strings.TrimSpace(parts[3]) == "1"
		b64 := strings.TrimSpace(parts[4])
		if repo == "" || tag == "" {
			continue
		}
		var mBytes []byte
		if b64 != "" {
			if decoded, decErr := base64.StdEncoding.DecodeString(b64); decErr == nil {
				mBytes = decoded
			}
		}
		repoSet[repo] = true
		byRepo[repo] = append(byRepo[repo], rawFSItem{
			repo:          repo,
			tag:           tag,
			digest:        digest,
			hasLayers:     hasLayers,
			manifestBytes: mBytes,
		})
	}

	if httpCode != "200" && httpCode != "401" {
		return &RegistryImagesResponse{
			Target:       target,
			EndpointHost: endpointHost,
			EndpointURL:  endpointURL,
			Healthy:      false,
			Repositories: []string{},
			Images:       []RegistryImageItem{},
			Error:        fmt.Sprintf("El endpoint V2 (%s/v2/) no responde (HTTP %s)", endpointURL, httpCode),
		}, true
	}

	repos := make([]string, 0, len(repoSet))
	for r := range repoSet {
		repos = append(repos, r)
	}
	sort.Strings(repos)

	var allImages []RegistryImageItem
	for _, repo := range repos {
		rawItems := byRepo[repo]
		sort.Slice(rawItems, func(i, j int) bool {
			return rawItems[i].tag < rawItems[j].tag
		})

		digestToTags := make(map[string][]string)
		repoItems := make([]RegistryImageItem, 0, len(rawItems))

		for _, raw := range rawItems {
			item := s.buildRegistryItemFromFS(ctx, target, cfg, endpointHost, raw.repo, raw.tag, raw.digest, raw.hasLayers, raw.manifestBytes)
			if item.Digest != "" {
				digestToTags[item.Digest] = append(digestToTags[item.Digest], item.Tag)
			}
			repoItems = append(repoItems, item)
		}

		for i := range repoItems {
			d := repoItems[i].Digest
			if d != "" && len(digestToTags[d]) > 1 {
				for _, otherTag := range digestToTags[d] {
					if otherTag != repoItems[i].Tag {
						repoItems[i].SharedTags = append(repoItems[i].SharedTags, otherTag)
					}
				}
			}
			allImages = append(allImages, repoItems[i])
		}
	}

	if allImages == nil {
		allImages = []RegistryImageItem{}
	}
	return &RegistryImagesResponse{
		Target:       target,
		EndpointHost: endpointHost,
		EndpointURL:  endpointURL,
		Healthy:      true,
		Repositories: repos,
		Images:       allImages,
	}, true
}

func (s *RegistryService) buildRegistryItemFromFS(
	ctx context.Context,
	target string,
	cfg *RegistryInstanceConfig,
	endpointHost, repo, tag, digest string,
	hasLayers bool,
	manifestBytes []byte,
) RegistryImageItem {
	if len(manifestBytes) == 0 {
		return s.inspectManifestV2(ctx, cfg.VMName, cfg.Port, endpointHost, repo, tag)
	}

	fullRef := fmt.Sprintf("%s/%s:%s", endpointHost, repo, tag)
	item := RegistryImageItem{
		Repository:    repo,
		Tag:           tag,
		Digest:        digest,
		FullReference: fullRef,
		Size:          "n/a",
	}

	var manifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
		Config        struct {
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
		} `json:"config"`
		Layers []struct {
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
		} `json:"layers"`
		Manifests []struct {
			Digest   string `json:"digest"`
			Size     int64  `json:"size"`
			Platform struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			} `json:"platform"`
		} `json:"manifests"`
	}

	if json.Unmarshal(manifestBytes, &manifest) != nil {
		return item
	}
	if manifest.MediaType != "" {
		item.MediaType = manifest.MediaType
	}

	if len(manifest.Layers) > 0 || manifest.Config.Size > 0 {
		var total int64 = manifest.Config.Size
		for _, l := range manifest.Layers {
			total += l.Size
		}
		item.SizeBytes = total
		item.Size = FormatByteSize(total)
		item.LayersCount = len(manifest.Layers)

		// Si estamos en Pull-through Cache y el repositorio solo tenía el manifiesto (porque la VM cliente ya poseía las capas),
		// calentar el blob de configuración para que registry:2.8.3 registre _layers y lo incluya en /v2/_catalog.
		if target == RegistryTargetCache && !hasLayers && strings.HasPrefix(manifest.Config.Digest, "sha256:") {
			warmCmd := fmt.Sprintf(`curl -fsS -o /dev/null --max-time 5 "http://127.0.0.1:%d/v2/%s/blobs/%s" 2>/dev/null || true`,
				cfg.Port, repo, manifest.Config.Digest)
			_, _ = s.runner(ctx, cfg.VMName, warmCmd)
		}
		return item
	}

	// Si es un Manifest List / OCI Index (multi-arquitectura), resolver el manifiesto hijo linux/amd64
	if len(manifest.Manifests) > 0 {
		chosenDigest := manifest.Manifests[0].Digest
		for _, m := range manifest.Manifests {
			if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" && m.Digest != "" {
				chosenDigest = m.Digest
				break
			}
		}
		if strings.HasPrefix(chosenDigest, "sha256:") {
			subHex := strings.TrimPrefix(chosenDigest, "sha256:")
			if len(subHex) >= 2 {
				subCmd := fmt.Sprintf(
					`sudo docker exec %s cat "/var/lib/registry/docker/registry/v2/blobs/sha256/%s/%s/data" 2>/dev/null || curl -fsS --max-time 6 -H %s "http://127.0.0.1:%d/v2/%s/manifests/%s"`,
					EscapeShellArg(cfg.ContainerName),
					subHex[:2],
					subHex,
					EscapeShellArg("Accept: "+registryAcceptHeaders),
					cfg.Port,
					repo,
					chosenDigest,
				)
				if subOut, subErr := s.runner(ctx, cfg.VMName, subCmd); subErr == nil {
					var subManifest struct {
						Config struct {
							Size   int64  `json:"size"`
							Digest string `json:"digest"`
						} `json:"config"`
						Layers []struct {
							Size int64 `json:"size"`
						} `json:"layers"`
					}
					if json.Unmarshal([]byte(strings.TrimSpace(subOut)), &subManifest) == nil && len(subManifest.Layers) > 0 {
						var total int64 = subManifest.Config.Size
						for _, l := range subManifest.Layers {
							total += l.Size
						}
						item.SizeBytes = total
						item.Size = FormatByteSize(total)
						item.LayersCount = len(subManifest.Layers)

						if target == RegistryTargetCache && !hasLayers && strings.HasPrefix(subManifest.Config.Digest, "sha256:") {
							warmCmd := fmt.Sprintf(`curl -fsS -o /dev/null --max-time 5 "http://127.0.0.1:%d/v2/%s/blobs/%s" 2>/dev/null || true`,
								cfg.Port, repo, subManifest.Config.Digest)
							_, _ = s.runner(ctx, cfg.VMName, warmCmd)
						}
					}
				}
			}
		}
	}

	return item
}

// inspectManifestV2 consulta el manifiesto de <repo>:<tag> en la API V2 soportando esquemas Docker V2 y OCI.
func (s *RegistryService) inspectManifestV2(ctx context.Context, vmName string, port int, endpointHost, repo, tag string) RegistryImageItem {
	fullRef := fmt.Sprintf("%s/%s:%s", endpointHost, repo, tag)
	item := RegistryImageItem{
		Repository:    repo,
		Tag:           tag,
		FullReference: fullRef,
		Size:          "n/a",
	}

	manifestCmd := fmt.Sprintf(
		`curl -fsSi --max-time 6 -H %s "http://127.0.0.1:%d/v2/%s/manifests/%s"`,
		EscapeShellArg("Accept: "+registryAcceptHeaders),
		port,
		repo,
		tag,
	)
	out, err := s.runner(ctx, vmName, manifestCmd)
	if err != nil || strings.TrimSpace(out) == "" {
		return item
	}

	headerPart, bodyPart := splitHTTPResponse(out)
	for _, hLine := range strings.Split(headerPart, "\n") {
		hLine = strings.TrimSpace(hLine)
		lower := strings.ToLower(hLine)
		if strings.HasPrefix(lower, "docker-content-digest:") {
			item.Digest = strings.TrimSpace(hLine[len("docker-content-digest:"):])
		} else if strings.HasPrefix(lower, "content-type:") {
			item.MediaType = strings.TrimSpace(hLine[len("content-type:"):])
		}
	}

	var manifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
		Config        struct {
			Size int64 `json:"size"`
		} `json:"config"`
		Layers []struct {
			Size int64 `json:"size"`
		} `json:"layers"`
		Manifests []struct {
			Digest   string `json:"digest"`
			Size     int64  `json:"size"`
			Platform struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			} `json:"platform"`
		} `json:"manifests"`
	}

	if json.Unmarshal([]byte(strings.TrimSpace(bodyPart)), &manifest) != nil {
		return item
	}

	if len(manifest.Layers) > 0 || manifest.Config.Size > 0 {
		var total int64 = manifest.Config.Size
		for _, l := range manifest.Layers {
			total += l.Size
		}
		item.SizeBytes = total
		item.Size = FormatByteSize(total)
		item.LayersCount = len(manifest.Layers)
		return item
	}

	// Si es un Manifest List / OCI Index (multi-arquitectura), inspeccionar el manifiesto hijo principal para calcular el tamaño
	if len(manifest.Manifests) > 0 {
		chosenDigest := manifest.Manifests[0].Digest
		for _, m := range manifest.Manifests {
			if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" && m.Digest != "" {
				chosenDigest = m.Digest
				break
			}
		}
		if chosenDigest != "" {
			subCmd := fmt.Sprintf(
				`curl -fsS --max-time 6 -H %s "http://127.0.0.1:%d/v2/%s/manifests/%s"`,
				EscapeShellArg("Accept: "+registryAcceptHeaders),
				port,
				repo,
				chosenDigest,
			)
			if subOut, subErr := s.runner(ctx, vmName, subCmd); subErr == nil {
				var subManifest struct {
					Config struct {
						Size int64 `json:"size"`
					} `json:"config"`
					Layers []struct {
						Size int64 `json:"size"`
					} `json:"layers"`
				}
				if json.Unmarshal([]byte(strings.TrimSpace(subOut)), &subManifest) == nil && len(subManifest.Layers) > 0 {
					var total int64 = subManifest.Config.Size
					for _, l := range subManifest.Layers {
						total += l.Size
					}
					item.SizeBytes = total
					item.Size = FormatByteSize(total)
					item.LayersCount = len(subManifest.Layers)
				}
			}
		}
	}

	return item
}

func splitHTTPResponse(raw string) (string, string) {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	idx := strings.Index(normalized, "\n\n")
	if idx == -1 {
		return "", normalized
	}
	return normalized[:idx], normalized[idx+2:]
}

// DeleteRegistryImage elimina un manifiesto del Registry local escribible verificando que no comprometa otros tags compartidos.
func (s *RegistryService) DeleteRegistryImage(ctx context.Context, repo, tag string, force bool) error {
	repo = strings.TrimSpace(repo)
	tag = strings.TrimSpace(tag)
	if err := ValidateImageRepository(repo); err != nil {
		return err
	}
	if err := ValidateImageTag(tag); err != nil {
		return err
	}

	persisted, err := s.store.Load()
	if err != nil || persisted.Local == nil || strings.TrimSpace(persisted.Local.VMName) == "" {
		return fmt.Errorf("el Registry local no está desplegado")
	}
	cfg := persisted.Local
	endpointHost := fmt.Sprintf("%s:%d", cfg.HostIP, cfg.Port)

	// Inspeccionar los tags actuales del repositorio para comprobar si otro tag comparte el mismo digest
	tagsCmd := fmt.Sprintf(`curl -fsS --max-time 6 "http://127.0.0.1:%d/v2/%s/tags/list"`, cfg.Port, repo)
	tagsOut, err := s.runner(ctx, cfg.VMName, tagsCmd)
	if err != nil {
		return fmt.Errorf("no se pudo consultar el repositorio %q en el Registry local: %w", repo, err)
	}
	var tagList struct {
		Tags []string `json:"tags"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(tagsOut)), &tagList)

	targetItem := s.inspectManifestV2(ctx, cfg.VMName, cfg.Port, endpointHost, repo, tag)
	if targetItem.Digest == "" {
		return fmt.Errorf("no se encontró el manifiesto o digest para '%s:%s' en el Registry local", repo, tag)
	}

	var sharedWith []string
	for _, t := range tagList.Tags {
		if t == tag {
			continue
		}
		other := s.inspectManifestV2(ctx, cfg.VMName, cfg.Port, endpointHost, repo, t)
		if other.Digest == targetItem.Digest {
			sharedWith = append(sharedWith, t)
		}
	}

	if len(sharedWith) > 0 && !force {
		return fmt.Errorf("el manifiesto de '%s:%s' (%s) es compartido con otro(s) tag(s) del mismo repositorio (%s). Marca la casilla de confirmación si deseas eliminar el manifiesto para todos esos tags",
			repo, tag, targetItem.Digest[:19], strings.Join(sharedWith, ", "))
	}

	delScript := fmt.Sprintf(
		`CODE=$(curl -s -o /dev/null -w '%%{http_code}' -X DELETE --max-time 8 "http://127.0.0.1:%d/v2/%s/manifests/%s"); `+
			`if [ "$CODE" != "202" ] && [ "$CODE" != "200" ]; then echo "DELETE_HTTP_$CODE"; exit 1; fi; `+
			`sudo docker exec %s bin/registry garbage-collect /etc/docker/registry/config.yml --delete-untagged=true >/dev/null 2>&1 || true; `+
			`echo "DELETE_OK"`,
		cfg.Port,
		repo,
		targetItem.Digest,
		EscapeShellArg(cfg.ContainerName),
	)

	out, delErr := s.runner(ctx, cfg.VMName, delScript)
	if delErr != nil {
		return fmt.Errorf("no se pudo eliminar '%s:%s' del Registry local: %s (%w)", repo, tag, strings.TrimSpace(out), delErr)
	}
	return nil
}

// PushToLocalRegistry etiqueta una imagen existente en una VM y la sube al Registry local escribible.
func (s *RegistryService) PushToLocalRegistry(ctx context.Context, req RegistryPushRequest) (*RegistryPushResponse, error) {
	vmName := strings.TrimSpace(req.VMName)
	if vmName == "" {
		return nil, fmt.Errorf("debes seleccionar la máquina virtual donde se encuentra la imagen origen")
	}
	sourceImage := strings.TrimSpace(req.SourceImage)
	if sourceImage == "" {
		return nil, fmt.Errorf("debes seleccionar una imagen local de origen")
	}
	if err := ValidateImageName(sourceImage); err != nil {
		return nil, err
	}

	repo := strings.TrimSpace(req.Repository)
	tag := strings.TrimSpace(req.Tag)
	if repo == "" {
		parsedRepo, parsedTag, pErr := ParseImageReference(sourceImage)
		if pErr != nil {
			return nil, fmt.Errorf("especifica un nombre de repositorio válido: %w", pErr)
		}
		repo = parsedRepo
		if tag == "" {
			tag = parsedTag
		}
	}
	if tag == "" {
		tag = "latest"
	}
	if err := ValidateImageRepository(repo); err != nil {
		return nil, err
	}
	if err := ValidateImageTag(tag); err != nil {
		return nil, err
	}

	persisted, err := s.store.Load()
	if err != nil || persisted.Local == nil || strings.TrimSpace(persisted.Local.VMName) == "" {
		return nil, fmt.Errorf("el Registry local escribible no está desplegado; despliégalo primero en la pestaña Registry Local")
	}
	localCfg := persisted.Local
	endpointHost := fmt.Sprintf("%s:%d", localCfg.HostIP, localCfg.Port)
	targetRef := fmt.Sprintf("%s/%s:%s", endpointHost, repo, tag)

	// Garantizar que el nodo origen confíe en el Registry HTTP
	_ = s.ensureNodeDaemonConfig(ctx, vmName, nil, []string{endpointHost}, false)

	pushCmd := fmt.Sprintf(
		"sudo docker tag %s %s 2>&1 && sudo docker push %s 2>&1",
		EscapeShellArg(sourceImage),
		EscapeShellArg(targetRef),
		EscapeShellArg(targetRef),
	)

	s.log.Info("Subiendo imagen al Registry local", "vm", vmName, "source", sourceImage, "target", targetRef)
	out, pushErr := s.runner(ctx, vmName, pushCmd)
	if pushErr != nil {
		return &RegistryPushResponse{
			Success:         false,
			VMName:          vmName,
			SourceImage:     sourceImage,
			TargetReference: targetRef,
			Repository:      repo,
			Tag:             tag,
			Output:          out,
		}, fmt.Errorf("error al subir %q al Registry local (%s): %s", targetRef, endpointHost, extractLastLine(out, pushErr))
	}

	return &RegistryPushResponse{
		Success:         true,
		VMName:          vmName,
		SourceImage:     sourceImage,
		TargetReference: targetRef,
		Repository:      repo,
		Tag:             tag,
		Output:          out,
	}, nil
}

// PullFromLocalRegistry descarga una imagen desde el Registry local escribible hacia la VM seleccionada.
func (s *RegistryService) PullFromLocalRegistry(ctx context.Context, req RegistryPullRequest) (*RegistryPullResponse, error) {
	vmName := strings.TrimSpace(req.VMName)
	if vmName == "" {
		return nil, fmt.Errorf("debes seleccionar la máquina virtual de destino")
	}

	repo := strings.TrimSpace(req.Repository)
	tag := strings.TrimSpace(req.Tag)
	if repo == "" && strings.TrimSpace(req.ImageRef) != "" {
		var err error
		repo, tag, err = ParseImageReference(req.ImageRef)
		if err != nil {
			return nil, err
		}
	}
	if tag == "" {
		tag = "latest"
	}
	if err := ValidateImageRepository(repo); err != nil {
		return nil, err
	}
	if err := ValidateImageTag(tag); err != nil {
		return nil, err
	}

	persisted, err := s.store.Load()
	if err != nil || persisted.Local == nil || strings.TrimSpace(persisted.Local.VMName) == "" {
		return nil, fmt.Errorf("el Registry local no está desplegado")
	}
	localCfg := persisted.Local
	endpointHost := fmt.Sprintf("%s:%d", localCfg.HostIP, localCfg.Port)
	fullRef := fmt.Sprintf("%s/%s:%s", endpointHost, repo, tag)
	shortTag := fmt.Sprintf("%s:%s", repo, tag)

	// Garantizar que el nodo destino confíe en el Registry HTTP
	_ = s.ensureNodeDaemonConfig(ctx, vmName, nil, []string{endpointHost}, false)

	pullCmd := fmt.Sprintf("sudo docker pull %s 2>&1", EscapeShellArg(fullRef))
	if req.RetagLocal {
		pullCmd += fmt.Sprintf(" && sudo docker tag %s %s 2>&1", EscapeShellArg(fullRef), EscapeShellArg(shortTag))
	}

	s.log.Info("Descargando imagen desde el Registry local", "vm", vmName, "image", fullRef)
	out, pullErr := s.runner(ctx, vmName, pullCmd)
	if pullErr != nil {
		return &RegistryPullResponse{
			Success:         false,
			VMName:          vmName,
			PulledReference: fullRef,
			Source:          "local-registry",
			SourceLabel:     fmt.Sprintf("Registry Local (%s)", endpointHost),
			Output:          out,
		}, fmt.Errorf("error descargando %q en la VM %q: %s", fullRef, vmName, extractLastLine(out, pullErr))
	}

	res := &RegistryPullResponse{
		Success:         true,
		VMName:          vmName,
		PulledReference: fullRef,
		Source:          "local-registry",
		SourceLabel:     fmt.Sprintf("Registry Local (%s)", endpointHost),
		Output:          out,
	}
	if req.RetagLocal {
		res.LocalTag = shortTag
	}
	return res, nil
}

// SmartPullImage ejecuta la lógica inteligente de descarga:
// PRIORIDAD 1: Consulta la API V2 del Registry local escribible. Si el manifiesto existe, descarga desde el Registry local sin acudir a Docker Hub.
// PRIORIDAD 2: Si no existe en el Registry local, descarga desde Docker Hub aprovechando el Pull-through Cache si está configurado.
func (s *RegistryService) SmartPullImage(ctx context.Context, req SmartPullRequest) (*SmartPullResponse, error) {
	vmName := strings.TrimSpace(req.VMName)
	if vmName == "" {
		return nil, fmt.Errorf("debes seleccionar una máquina virtual de destino")
	}

	rawImage := strings.TrimSpace(req.Image)
	persisted, _ := s.store.Load()

	// Si el usuario pegó una referencia que incluye explícitamente el prefijo del Registry local (ej. 10.250.0.5:5000/mysql:8.4), quitarlo para normalizar repo:tag
	if persisted.Local != nil && persisted.Local.HostIP != "" && persisted.Local.Port > 0 {
		prefix := fmt.Sprintf("%s:%d/", persisted.Local.HostIP, persisted.Local.Port)
		rawImage = strings.TrimPrefix(rawImage, prefix)
	}

	repo, tag, err := ParseImageReference(rawImage)
	if err != nil {
		return nil, err
	}
	normalizedImage := fmt.Sprintf("%s:%s", repo, tag)

	// =========================================================================
	// PRIORIDAD 1: Comprobar en el Registry local escribible mediante API V2
	// =========================================================================
	if persisted.Local != nil && strings.TrimSpace(persisted.Local.VMName) != "" {
		localCfg := persisted.Local
		localEndpoint := fmt.Sprintf("%s:%d", localCfg.HostIP, localCfg.Port)

		healthy, matchedRepo, checkErr := s.checkManifestInRegistry(ctx, localCfg, repo, tag)
		if !healthy {
			errMsg := fmt.Sprintf("El Registry local (%s en %s) no responde (%v).", localEndpoint, localCfg.VMName, checkErr)
			if !req.AllowFallback {
				return &SmartPullResponse{
					Success:                  false,
					VMName:                   vmName,
					RequestedImage:           normalizedImage,
					LocalRegistryChecked:     true,
					LocalRegistryUnreachable: true,
					LocalRegistryError:       errMsg,
					CanFallbackToDockerHub:   true,
					Source:                   "local-registry-unreachable",
					SourceLabel:              fmt.Sprintf("Registry Local inaccesible (%s)", localEndpoint),
				}, fmt.Errorf("%s", errMsg)
			}
		} else if matchedRepo != "" {
			// ¡El manifiesto existe en el Registry local! Descargar directamente desde el Registry local sin acudir a Docker Hub.
			_ = s.ensureNodeDaemonConfig(ctx, vmName, nil, []string{localEndpoint}, false)
			fullLocalRef := fmt.Sprintf("%s/%s:%s", localEndpoint, matchedRepo, tag)

			pullCmd := fmt.Sprintf(
				"sudo docker pull %s 2>&1 && sudo docker tag %s %s 2>&1",
				EscapeShellArg(fullLocalRef),
				EscapeShellArg(fullLocalRef),
				EscapeShellArg(normalizedImage),
			)
			out, pullErr := s.runner(ctx, vmName, pullCmd)
			if pullErr != nil {
				return &SmartPullResponse{
					Success:              false,
					VMName:               vmName,
					RequestedImage:       normalizedImage,
					PulledReference:      fullLocalRef,
					FoundInLocalRegistry: true,
					LocalRegistryChecked: true,
					Source:               "local-registry",
					SourceLabel:          fmt.Sprintf("Registry Local (%s)", localEndpoint),
					Output:               out,
				}, fmt.Errorf("la imagen '%s' existe en el Registry local (%s) pero falló su descarga en '%s': %s", normalizedImage, localEndpoint, vmName, extractLastLine(out, pullErr))
			}

			return &SmartPullResponse{
				Success:              true,
				VMName:               vmName,
				RequestedImage:       normalizedImage,
				PulledReference:      fullLocalRef,
				LocalTag:             normalizedImage,
				FoundInLocalRegistry: true,
				LocalRegistryChecked: true,
				Source:               "local-registry",
				SourceLabel:          fmt.Sprintf("Registry Local (%s) — Prioridad 1", localEndpoint),
				Output:               out,
			}, nil
		}
	}

	// =========================================================================
	// PRIORIDAD 2: No existe en el Registry local -> Docker Hub (vía Caché si aplica)
	// =========================================================================
	cacheActiveOnNode := false
	wasCachedBefore := false
	cacheEndpoint := ""

	if persisted.Cache != nil && strings.TrimSpace(persisted.Cache.VMName) != "" {
		cacheCfg := persisted.Cache
		cacheEndpoint = fmt.Sprintf("http://%s:%d", cacheCfg.HostIP, cacheCfg.Port)
		// Comprobar si el nodo destino tiene configurado este mirror en daemon.json
		if daemonOut, dErr := s.runner(ctx, vmName, "sudo cat /etc/docker/daemon.json 2>/dev/null || echo '{}'"); dErr == nil {
			if strings.Contains(strings.ToLower(daemonOut), strings.ToLower(fmt.Sprintf("%s:%d", cacheCfg.HostIP, cacheCfg.Port))) {
				cacheActiveOnNode = true
			}
		}
		if cacheActiveOnNode {
			if healthy, matchedCacheRepo, _ := s.checkManifestInRegistry(ctx, cacheCfg, repo, tag); healthy && matchedCacheRepo != "" {
				wasCachedBefore = true
			}
		}
	}

	pullHubCmd := fmt.Sprintf("sudo docker pull %s 2>&1", EscapeShellArg(normalizedImage))
	out, pullErr := s.runner(ctx, vmName, pullHubCmd)
	if pullErr != nil {
		return &SmartPullResponse{
			Success:              false,
			VMName:               vmName,
			RequestedImage:       normalizedImage,
			LocalRegistryChecked: persisted.Local != nil,
			UsedPullThroughCache: cacheActiveOnNode,
			Source:               "dockerhub-error",
			SourceLabel:          "Docker Hub",
			Output:               out,
		}, fmt.Errorf("la imagen '%s' no se encontró en el Registry local y falló su descarga desde Docker Hub en '%s': %s", normalizedImage, vmName, extractLastLine(out, pullErr))
	}

	isCachedAfter := false
	if cacheActiveOnNode && persisted.Cache != nil {
		if healthy, matchedCacheRepo, _ := s.checkManifestInRegistry(ctx, persisted.Cache, repo, tag); healthy && matchedCacheRepo != "" {
			isCachedAfter = true
		} else if healthy && strings.TrimSpace(persisted.Cache.ContainerName) != "" {
			// Si el daemon Docker del nodo ya tenía las capas preinstaladas y no forzó la escritura en el mirror,
			// calentar el manifiesto en el Pull-through Cache para garantizar que quede registrado en caché.
			warmRepo := repo
			if !strings.Contains(repo, "/") {
				warmRepo = "library/" + repo
			}
			warmCmd := fmt.Sprintf(
				`curl -fsS -o /dev/null --max-time 8 -H %s "http://127.0.0.1:%d/v2/%s/manifests/%s" 2>/dev/null || true`,
				EscapeShellArg("Accept: "+registryAcceptHeaders),
				persisted.Cache.Port,
				warmRepo,
				tag,
			)
			_, _ = s.runner(ctx, persisted.Cache.VMName, warmCmd)
			if h2, m2, _ := s.checkManifestInRegistry(ctx, persisted.Cache, repo, tag); h2 && m2 != "" {
				isCachedAfter = true
			}
		}
	}

	source := "dockerhub-direct"
	sourceLabel := "Docker Hub (Directo)"
	if cacheActiveOnNode && isCachedAfter {
		if wasCachedBefore {
			source = "dockerhub-cache-hit"
			sourceLabel = fmt.Sprintf("Pull-through Cache de Docker Hub (%s — Servida desde caché)", cacheEndpoint)
		} else {
			source = "dockerhub-cache-miss"
			sourceLabel = fmt.Sprintf("Docker Hub vía Pull-through Cache (%s — Almacenada en caché para próximos pulls)", cacheEndpoint)
		}
	} else if cacheActiveOnNode {
		source = "dockerhub-cache"
		sourceLabel = fmt.Sprintf("Docker Hub con mirror configurado (%s)", cacheEndpoint)
	}

	return &SmartPullResponse{
		Success:              true,
		VMName:               vmName,
		RequestedImage:       normalizedImage,
		PulledReference:      normalizedImage,
		LocalTag:             normalizedImage,
		FoundInLocalRegistry: false,
		LocalRegistryChecked: persisted.Local != nil,
		UsedPullThroughCache: cacheActiveOnNode && isCachedAfter,
		CacheHit:             wasCachedBefore && isCachedAfter,
		Source:               source,
		SourceLabel:          sourceLabel,
		Output:               out,
	}, nil
}

// checkManifestInRegistry verifica mediante la API V2 si el Registry está activo (healthy) y si contiene el manifiesto <repo>:<tag> (o library/<repo>:<tag>).
func (s *RegistryService) checkManifestInRegistry(ctx context.Context, cfg *RegistryInstanceConfig, repo, tag string) (bool, string, error) {
	if cfg == nil || strings.TrimSpace(cfg.VMName) == "" {
		return false, "", fmt.Errorf("instancia no configurada")
	}

	candidates := []string{repo}
	if !strings.Contains(repo, "/") {
		candidates = append(candidates, "library/"+repo)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	// Comprobar primero que /v2/ responda 200
	pingCmd := fmt.Sprintf(`curl -s -o /dev/null -w '%%{http_code}' --max-time 4 "http://127.0.0.1:%d/v2/"`, cfg.Port)
	pingOut, pingErr := s.runner(cmdCtx, cfg.VMName, pingCmd)
	code := strings.TrimSpace(pingOut)
	if pingErr != nil || (code != "200" && code != "401") {
		if pingErr != nil {
			return false, "", pingErr
		}
		return false, "", fmt.Errorf("HTTP %s en /v2/", code)
	}

	if strings.EqualFold(cfg.Target, RegistryTargetCache) && strings.TrimSpace(cfg.ContainerName) != "" {
		for _, candidateRepo := range candidates {
			fsCheckCmd := fmt.Sprintf(
				`sudo docker exec %s sh -c 'if [ -f "/var/lib/registry/docker/registry/v2/repositories/%s/_manifests/tags/%s/current/link" ]; then echo "FS_CACHED|1"; else echo "FS_CACHED|0"; fi' 2>/dev/null || true`,
				EscapeShellArg(cfg.ContainerName),
				candidateRepo,
				tag,
			)
			fsOut, fsErr := s.runner(cmdCtx, cfg.VMName, fsCheckCmd)
			if fsErr == nil && strings.Contains(fsOut, "FS_CACHED|") {
				if strings.Contains(fsOut, "FS_CACHED|1") {
					return true, candidateRepo, nil
				}
				continue
			}
			checkCmd := fmt.Sprintf(
				`curl -s -o /dev/null -w '%%{http_code}' --max-time 5 -H %s "http://127.0.0.1:%d/v2/%s/manifests/%s"`,
				EscapeShellArg("Accept: "+registryAcceptHeaders),
				cfg.Port,
				candidateRepo,
				tag,
			)
			out, err := s.runner(cmdCtx, cfg.VMName, checkCmd)
			if err == nil && strings.TrimSpace(out) == "200" {
				return true, candidateRepo, nil
			}
		}
		return true, "", nil
	}

	for _, candidateRepo := range candidates {
		checkCmd := fmt.Sprintf(
			`curl -s -o /dev/null -w '%%{http_code}' --max-time 5 -H %s "http://127.0.0.1:%d/v2/%s/manifests/%s"`,
			EscapeShellArg("Accept: "+registryAcceptHeaders),
			cfg.Port,
			candidateRepo,
			tag,
		)
		out, err := s.runner(cmdCtx, cfg.VMName, checkCmd)
		if err == nil && strings.TrimSpace(out) == "200" {
			return true, candidateRepo, nil
		}
	}

	return true, "", nil
}

// ConfigureNodesMirror aplica de forma segura `registry-mirrors` e `insecure-registries` en `/etc/docker/daemon.json` de los nodos seleccionados,
// preservando todas las propiedades existentes.
func (s *RegistryService) ConfigureNodesMirror(ctx context.Context, req ConfigureNodesRegistryRequest) (*ConfigureNodesRegistryResponse, error) {
	if !req.Confirmed {
		return nil, fmt.Errorf("se requiere confirmación explícita para aplicar los cambios en el daemon Docker de los nodos")
	}

	persisted, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	var mirrors []string
	var insecure []string

	if persisted.Cache != nil && persisted.Cache.HostIP != "" && persisted.Cache.Port > 0 {
		mirrors = append(mirrors, fmt.Sprintf("http://%s:%d", persisted.Cache.HostIP, persisted.Cache.Port))
		insecure = append(insecure, fmt.Sprintf("%s:%d", persisted.Cache.HostIP, persisted.Cache.Port))
	}
	if persisted.Local != nil && persisted.Local.HostIP != "" && persisted.Local.Port > 0 {
		insecure = append(insecure, fmt.Sprintf("%s:%d", persisted.Local.HostIP, persisted.Local.Port))
	}

	if len(mirrors) == 0 && len(insecure) == 0 {
		return nil, fmt.Errorf("no hay ningún Registry local ni Pull-through Cache desplegado todavía")
	}

	targetVMs := req.VMNames
	if len(targetVMs) == 0 {
		vms, err := s.vmProvider.ListVMs()
		if err != nil {
			return nil, fmt.Errorf("error listando VMs: %w", err)
		}
		for _, v := range vms {
			if isVMRunningState(v.State) {
				targetVMs = append(targetVMs, v.Name)
			}
		}
	}

	if len(targetVMs) == 0 {
		return nil, fmt.Errorf("no hay máquinas virtuales en ejecución seleccionadas para configurar")
	}

	results := make([]NodeConfigureResult, 0, len(targetVMs))
	allOk := true

	for _, vmName := range targetVMs {
		vmName = strings.TrimSpace(vmName)
		if vmName == "" {
			continue
		}
		err := s.ensureNodeDaemonConfig(ctx, vmName, mirrors, insecure, req.RestartDocker)
		method := "reload (SIGHUP sin detener contenedores)"
		if req.RestartDocker {
			method = "restart (reinicio completo de docker.service)"
		}
		if err != nil {
			allOk = false
			results = append(results, NodeConfigureResult{
				VMName:  vmName,
				Success: false,
				Method:  method,
				Message: fmt.Sprintf("Error en '%s': %v", vmName, err),
				Error:   err.Error(),
			})
		} else {
			results = append(results, NodeConfigureResult{
				VMName:  vmName,
				Success: true,
				Method:  method,
				Message: fmt.Sprintf("Configuración aplicada en '%s' (%s)", vmName, method),
			})
		}
	}

	msg := fmt.Sprintf("Configuración de Registry/Mirror aplicada en %d nodo(s).", len(results))
	if !allOk {
		msg = "Se completó la operación con errores en uno o más nodos."
	}

	return &ConfigureNodesRegistryResponse{
		Success: allOk,
		Results: results,
		Message: msg,
	}, nil
}

// ensureNodeDaemonConfig lee `/etc/docker/daemon.json` en la VM indicada, combina los mirrors e insecure-registries preservando las demás propiedades,
// y recarga (o reinicia si se solicita explícitamente) el servicio Docker.
func (s *RegistryService) ensureNodeDaemonConfig(ctx context.Context, vmName string, mirrorsToAdd []string, insecureToAdd []string, restartDocker bool) error {
	cmdCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	existingRaw, err := s.runner(cmdCtx, vmName, "sudo cat /etc/docker/daemon.json 2>/dev/null || echo '{}'")
	if err != nil {
		return fmt.Errorf("leyendo /etc/docker/daemon.json en %q: %w", vmName, err)
	}

	mergedJSON, changed, err := MergeDaemonJSON(existingRaw, mirrorsToAdd, insecureToAdd)
	if err != nil {
		return err
	}

	if !changed && !restartDocker {
		return nil
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(mergedJSON))
	reloadCmd := "sudo systemctl reload docker 2>/dev/null || sudo kill -HUP $(pidof dockerd) 2>/dev/null || true"
	if restartDocker {
		reloadCmd = "sudo systemctl restart docker"
	}

	applyScript := fmt.Sprintf(
		"sudo mkdir -p /etc/docker && echo %s | base64 -d | sudo tee /etc/docker/daemon.json >/dev/null && %s",
		EscapeShellArg(b64),
		reloadCmd,
	)

	out, applyErr := s.runner(cmdCtx, vmName, applyScript)
	if applyErr != nil {
		return fmt.Errorf("aplicando /etc/docker/daemon.json en %q: %s (%w)", vmName, strings.TrimSpace(out), applyErr)
	}
	return nil
}

func extractLastLine(out string, fallbackErr error) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l != "" {
			return l
		}
	}
	if fallbackErr != nil {
		return fallbackErr.Error()
	}
	return "error desconocido"
}
