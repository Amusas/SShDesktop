package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"
)

// CommandRunner define la función para ejecutar comandos de forma remota en una VM
type CommandRunner func(ctx context.Context, vmName string, command string) (string, error)

// Service gestiona el ciclo de vida y operaciones de contenedores Docker en VMs Linux
type Service struct {
	runner CommandRunner
	log    *slog.Logger
}

// NewService crea una nueva instancia del servicio Docker
func NewService(runner CommandRunner, log *slog.Logger) *Service {
	return &Service{
		runner: runner,
		log:    log,
	}
}

// CheckDockerStatus comprueba si el daemon de Docker está activo y obtiene métricas en la VM
func (s *Service) CheckDockerStatus(ctx context.Context, vmName string) (*DockerStatus, error) {
	if strings.TrimSpace(vmName) == "" {
		return nil, fmt.Errorf("el nombre de la VM no puede estar vacío")
	}

	cmd := "sudo docker info --format '{{json .}}' 2>/dev/null"
	out, err := s.runner(ctx, vmName, cmd)
	if err != nil || strings.TrimSpace(out) == "" {
		// Fallback simple comprobando versión
		vCmd := "sudo docker version --format '{{.Server.Version}}' 2>/dev/null"
		vOut, vErr := s.runner(ctx, vmName, vCmd)
		if vErr != nil || strings.TrimSpace(vOut) == "" {
			return &DockerStatus{
				Available: false,
				Error:     "Docker no está disponible o el servicio docker.service está detenido en esta VM",
			}, nil
		}
		return &DockerStatus{
			Available: true,
			Version:   strings.TrimSpace(vOut),
		}, nil
	}

	var raw struct {
		ServerVersion string `json:"ServerVersion"`
		Containers    int    `json:"Containers"`
		ContainersRunning int `json:"ContainersRunning"`
		Images        int    `json:"Images"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &raw); err != nil {
		return &DockerStatus{
			Available: true,
			Version:   "Detectado",
		}, nil
	}

	return &DockerStatus{
		Available:         true,
		Version:           raw.ServerVersion,
		ContainersRunning: raw.ContainersRunning,
		ContainersTotal:   raw.Containers,
		ImagesTotal:       raw.Images,
	}, nil
}

// ListContainers obtiene la lista de contenedores en una VM específica
func (s *Service) ListContainers(ctx context.Context, vmName string) ([]Container, error) {
	if strings.TrimSpace(vmName) == "" {
		return nil, fmt.Errorf("el nombre de la VM no puede estar vacío")
	}

	format := `{"id":{{json .ID}},"name":{{json .Names}},"image":{{json .Image}},"command":{{json .Command}},"createdAt":{{json .CreatedAt}},"state":{{json .State}},"status":{{json .Status}},"ports":{{json .Ports}}}`
	cmd := fmt.Sprintf("sudo docker ps -a --format '%s'", format)

	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		return nil, fmt.Errorf("listando contenedores en VM '%s': %w", vmName, err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var containers []Container
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c Container
		if err := json.Unmarshal([]byte(line), &c); err == nil {
			c.VMName = vmName
			// Normalizar nombre de contenedor quitando prefijo '/'
			c.Name = strings.TrimPrefix(c.Name, "/")
			containers = append(containers, c)
		}
	}

	return containers, nil
}

// InspectContainer obtiene todos los detalles de configuración y estado de un contenedor
func (s *Service) InspectContainer(ctx context.Context, vmName string, containerID string) (*ContainerInspect, error) {
	if err := ValidateContainerName(containerID); err != nil {
		// Puede ser un ID hexadecimal
		if len(containerID) < 6 || strings.ContainsAny(containerID, " ;&|`$") {
			return nil, fmt.Errorf("identificador de contenedor no válido: %s", containerID)
		}
	}

	cmd := fmt.Sprintf("sudo docker inspect %s", EscapeShellArg(containerID))
	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		return nil, fmt.Errorf("inspeccionando contenedor '%s' en VM '%s': %w", containerID, vmName, err)
	}

	type dockerInspectItem struct {
		Id      string `json:"Id"`
		Name    string `json:"Name"`
		Created string `json:"Created"`
		State   struct {
			Status     string `json:"Status"`
			Running    bool   `json:"Running"`
			Paused     bool   `json:"Paused"`
			Restarting bool   `json:"Restarting"`
			ExitCode   int    `json:"ExitCode"`
			StartedAt  string `json:"StartedAt"`
			FinishedAt string `json:"FinishedAt"`
			Error      string `json:"Error"`
		} `json:"State"`
		Config struct {
			Image  string            `json:"Image"`
			Env    []string          `json:"Env"`
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
		HostConfig struct {
			RestartPolicy struct {
				Name string `json:"Name"`
			} `json:"RestartPolicy"`
			PortBindings map[string][]struct {
				HostPort string `json:"HostPort"`
			} `json:"PortBindings"`
			Binds []string `json:"Binds"`
		} `json:"HostConfig"`
		NetworkSettings struct {
			IPAddress string `json:"IPAddress"`
			Ports     map[string][]struct {
				HostPort string `json:"HostPort"`
			} `json:"Ports"`
			Networks map[string]struct {
				IPAddress  string `json:"IPAddress"`
				Gateway    string `json:"Gateway"`
				MacAddress string `json:"MacAddress"`
				NetworkID  string `json:"NetworkID"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}

	var items []dockerInspectItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &items); err != nil || len(items) == 0 {
		return nil, fmt.Errorf("fallo al parsear inspección de contenedor: %w", err)
	}

	item := items[0]
	inspect := &ContainerInspect{
		ID:            item.Id,
		Name:          strings.TrimPrefix(item.Name, "/"),
		VMName:        vmName,
		Created:       item.Created,
		Image:         item.Config.Image,
		RestartPolicy: item.HostConfig.RestartPolicy.Name,
		IPAddress:     item.NetworkSettings.IPAddress,
		Env:           item.Config.Env,
		Labels:        item.Config.Labels,
		State: InspectState{
			Status:     item.State.Status,
			Running:    item.State.Running,
			Paused:     item.State.Paused,
			Restarting: item.State.Restarting,
			ExitCode:   item.State.ExitCode,
			StartedAt:  item.State.StartedAt,
			FinishedAt: item.State.FinishedAt,
			Error:      item.State.Error,
		},
	}

	// Mapear redes
	for netName, netData := range item.NetworkSettings.Networks {
		inspect.Networks = append(inspect.Networks, ContainerNetworkInfo{
			Name:       netName,
			IPAddress:  netData.IPAddress,
			Gateway:    netData.Gateway,
			MacAddress: netData.MacAddress,
			NetworkID:  netData.NetworkID,
		})
		if inspect.IPAddress == "" && netData.IPAddress != "" {
			inspect.IPAddress = netData.IPAddress
		}
	}

	// Mapear puertos
	portsMap := item.NetworkSettings.Ports
	if len(portsMap) == 0 {
		portsMap = item.HostConfig.PortBindings
	}
	for cPortKey, bindings := range portsMap {
		var cPort int
		var proto string = "tcp"
		parts := strings.Split(cPortKey, "/")
		fmt.Sscanf(parts[0], "%d", &cPort)
		if len(parts) > 1 {
			proto = parts[1]
		}
		for _, b := range bindings {
			var hPort int
			fmt.Sscanf(b.HostPort, "%d", &hPort)
			inspect.Ports = append(inspect.Ports, PortMapping{
				HostPort:      hPort,
				ContainerPort: cPort,
				Protocol:      proto,
			})
		}
	}

	// Mapear montajes
	for _, bind := range item.HostConfig.Binds {
		parts := strings.Split(bind, ":")
		if len(parts) >= 2 {
			ro := len(parts) >= 3 && parts[2] == "ro"
			inspect.Mounts = append(inspect.Mounts, VolumeMount{
				HostPath:      parts[0],
				ContainerPath: parts[1],
				ReadOnly:      ro,
			})
		}
	}

	return inspect, nil
}

// CreateContainer valida y lanza un nuevo contenedor en la VM especificada
func (s *Service) CreateContainer(ctx context.Context, req CreateContainerRequest) (*Container, error) {
	if strings.TrimSpace(req.VMName) == "" {
		return nil, fmt.Errorf("el nombre de la máquina virtual (VM) es obligatorio")
	}
	if err := ValidateContainerName(req.Name); err != nil {
		return nil, err
	}
	if err := ValidateImageName(req.Image); err != nil {
		return nil, err
	}
	if err := ValidateRestartPolicy(req.RestartPolicy); err != nil {
		return nil, err
	}

	var args []string
	args = append(args, "sudo", "docker", "run", "-d", "--name", EscapeShellArg(req.Name))

	policy := req.RestartPolicy
	if policy == "" {
		policy = "unless-stopped"
	}
	args = append(args, "--restart", EscapeShellArg(policy))

	var allNets []string
	if strings.TrimSpace(req.Network) != "" {
		allNets = append(allNets, strings.TrimSpace(req.Network))
	}
	for _, n := range req.Networks {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		exists := false
		for _, prev := range allNets {
			if strings.EqualFold(prev, n) {
				exists = true
				break
			}
		}
		if !exists {
			allNets = append(allNets, n)
		}
	}

	for _, n := range allNets {
		if err := ValidateNetworkName(n); err != nil {
			return nil, fmt.Errorf("nombre de red inválido: %w", err)
		}
	}

	if len(allNets) > 0 {
		args = append(args, "--network", EscapeShellArg(allNets[0]))
	}

	for _, p := range req.Ports {
		if err := ValidatePort(p.HostPort); err != nil {
			return nil, fmt.Errorf("puerto host inválido: %w", err)
		}
		if err := ValidatePort(p.ContainerPort); err != nil {
			return nil, fmt.Errorf("puerto contenedor inválido: %w", err)
		}
		proto := strings.ToLower(strings.TrimSpace(p.Protocol))
		if proto == "" {
			proto = "tcp"
		}
		if proto != "tcp" && proto != "udp" {
			return nil, fmt.Errorf("protocolo no soportado (debe ser tcp o udp): %s", proto)
		}
		args = append(args, "-p", EscapeShellArg(fmt.Sprintf("%d:%d/%s", p.HostPort, p.ContainerPort, proto)))
	}

	for _, v := range req.Volumes {
		if err := ValidateVolumePath(v.HostPath); err != nil {
			return nil, fmt.Errorf("ruta host de volumen inválida: %w", err)
		}
		if err := ValidateVolumePath(v.ContainerPath); err != nil {
			return nil, fmt.Errorf("ruta en contenedor de volumen inválida: %w", err)
		}
		mode := "rw"
		if v.ReadOnly {
			mode = "ro"
		}
		args = append(args, "-v", EscapeShellArg(fmt.Sprintf("%s:%s:%s", v.HostPath, v.ContainerPath, mode)))
	}

	for _, e := range req.Env {
		if strings.TrimSpace(e.Key) == "" {
			continue
		}
		if err := ValidateEnvKey(e.Key); err != nil {
			return nil, fmt.Errorf("variable de entorno inválida: %w", err)
		}
		args = append(args, "-e", EscapeShellArg(fmt.Sprintf("%s=%s", e.Key, e.Value)))
	}

	args = append(args, EscapeShellArg(req.Image))

	if strings.TrimSpace(req.Command) != "" {
		// Validar que el comando no contenga metacaracteres peligrosos directos
		args = append(args, EscapeShellArg(req.Command))
	}

	runCmd := strings.Join(args, " ")
	s.log.Info("creando contenedor Docker en VM", "vm", req.VMName, "name", req.Name, "image", req.Image)

	out, err := s.runner(ctx, req.VMName, runCmd)
	if err != nil {
		return nil, fmt.Errorf("creando contenedor '%s' en VM '%s': %w", req.Name, req.VMName, err)
	}

	// Conectar redes adicionales si fueron especificadas
	if len(allNets) > 1 {
		for _, addNet := range allNets[1:] {
			connectCmd := fmt.Sprintf("sudo docker network connect %s %s", EscapeShellArg(addNet), EscapeShellArg(req.Name))
			if _, connErr := s.runner(ctx, req.VMName, connectCmd); connErr != nil {
				s.log.Warn("no se pudo conectar contenedor a red adicional", "vm", req.VMName, "container", req.Name, "network", addNet, "error", connErr)
			}
		}
	}

	cid := strings.TrimSpace(out)
	if len(cid) > 12 {
		cid = cid[:12]
	}

	return &Container{
		ID:        cid,
		Name:      req.Name,
		Image:     req.Image,
		State:     "running",
		Status:    "Up Less than a second",
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		VMName:    req.VMName,
	}, nil
}

// StartContainer arranca un contenedor detenido en la VM
func (s *Service) StartContainer(ctx context.Context, vmName string, containerID string) error {
	if err := ValidateContainerName(containerID); err != nil {
		if len(containerID) < 6 || strings.ContainsAny(containerID, " ;&|`$") {
			return fmt.Errorf("identificador de contenedor no válido: %s", containerID)
		}
	}
	cmd := fmt.Sprintf("sudo docker start %s", EscapeShellArg(containerID))
	_, err := s.runner(ctx, vmName, cmd)
	return err
}

// StopContainer detiene un contenedor en ejecución de forma ordenada
func (s *Service) StopContainer(ctx context.Context, vmName string, containerID string) error {
	if err := ValidateContainerName(containerID); err != nil {
		if len(containerID) < 6 || strings.ContainsAny(containerID, " ;&|`$") {
			return fmt.Errorf("identificador de contenedor no válido: %s", containerID)
		}
	}
	cmd := fmt.Sprintf("sudo docker stop -t 10 %s", EscapeShellArg(containerID))
	_, err := s.runner(ctx, vmName, cmd)
	return err
}

// RestartContainer reinicia un contenedor en la VM
func (s *Service) RestartContainer(ctx context.Context, vmName string, containerID string) error {
	if err := ValidateContainerName(containerID); err != nil {
		if len(containerID) < 6 || strings.ContainsAny(containerID, " ;&|`$") {
			return fmt.Errorf("identificador de contenedor no válido: %s", containerID)
		}
	}
	cmd := fmt.Sprintf("sudo docker restart -t 10 %s", EscapeShellArg(containerID))
	_, err := s.runner(ctx, vmName, cmd)
	return err
}

// DeleteContainer elimina un contenedor en la VM (con opción de forzado)
func (s *Service) DeleteContainer(ctx context.Context, vmName string, containerID string, force bool) error {
	if err := ValidateContainerName(containerID); err != nil {
		if len(containerID) < 6 || strings.ContainsAny(containerID, " ;&|`$") {
			return fmt.Errorf("identificador de contenedor no válido: %s", containerID)
		}
	}
	forceFlag := ""
	if force {
		forceFlag = "-f "
	}
	cmd := fmt.Sprintf("sudo docker rm %s%s", forceFlag, EscapeShellArg(containerID))
	_, err := s.runner(ctx, vmName, cmd)
	return err
}

// GetLogs obtiene los registros de un contenedor
func (s *Service) GetLogs(ctx context.Context, vmName string, containerID string, tail int) (string, error) {
	if err := ValidateContainerName(containerID); err != nil {
		if len(containerID) < 6 || strings.ContainsAny(containerID, " ;&|`$") {
			return "", fmt.Errorf("identificador de contenedor no válido: %s", containerID)
		}
	}
	if tail <= 0 || tail > 2000 {
		tail = 200
	}
	cmd := fmt.Sprintf("sudo docker logs --tail %d --timestamps %s 2>&1", tail, EscapeShellArg(containerID))
	out, err := s.runner(ctx, vmName, cmd)
	if err != nil && strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("obteniendo logs de contenedor '%s': %w", containerID, err)
	}
	return out, nil
}

// ListImages obtiene la lista de imágenes Docker disponibles en una VM
func (s *Service) ListImages(ctx context.Context, vmName string) ([]Image, error) {
	if strings.TrimSpace(vmName) == "" {
		return nil, fmt.Errorf("el nombre de la VM no puede estar vacío")
	}

	format := `{"id":{{json .ID}},"repository":{{json .Repository}},"tag":{{json .Tag}},"size":{{json .Size}},"createdAt":{{json .CreatedAt}}}`
	cmd := fmt.Sprintf("sudo docker image ls --format '%s'", format)

	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		return nil, fmt.Errorf("listando imágenes en VM '%s': %w", vmName, err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var images []Image
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var img Image
		if err := json.Unmarshal([]byte(line), &img); err == nil {
			img.VMName = vmName
			images = append(images, img)
		}
	}

	return images, nil
}

// BuildImage crea una imagen a partir de un Dockerfile y archivos de contexto dentro de la VM Linux
func (s *Service) BuildImage(ctx context.Context, req BuildImageRequest) (*BuildImageResponse, error) {
	if strings.TrimSpace(req.VMName) == "" {
		return nil, fmt.Errorf("el nombre de la máquina virtual (VM) es obligatorio")
	}
	if err := ValidateImageRepository(req.ImageName); err != nil {
		return nil, err
	}
	tag := strings.TrimSpace(req.Tag)
	if tag == "" {
		tag = "latest"
	}
	if err := ValidateImageTag(tag); err != nil {
		return nil, err
	}
	if err := ValidateDockerfile(req.Dockerfile); err != nil {
		return nil, err
	}

	// Validar y preparar archivos adicionales del contexto de compilación
	type preparedFile struct {
		relPath string
		content string
	}
	var validFiles []preparedFile
	for _, f := range req.Files {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		cleanName := path.Clean(strings.ReplaceAll(name, "\\", "/"))
		if cleanName == "." || cleanName == "/" || cleanName == ".." || strings.HasPrefix(cleanName, "../") || strings.HasPrefix(cleanName, "/") {
			return nil, fmt.Errorf("nombre de archivo adjunto inválido o inseguro: '%s'", f.Name)
		}
		if strings.EqualFold(cleanName, "Dockerfile") {
			return nil, fmt.Errorf("el archivo 'Dockerfile' debe definirse en el contenido principal, no en los archivos adjuntos")
		}
		// Validar contenido Base64
		if _, err := base64.StdEncoding.DecodeString(f.Content); err != nil {
			return nil, fmt.Errorf("el contenido del archivo adjunto '%s' no es base64 válido: %w", f.Name, err)
		}
		validFiles = append(validFiles, preparedFile{
			relPath: cleanName,
			content: f.Content,
		})
	}

	fullTag := fmt.Sprintf("%s:%s", strings.TrimSpace(req.ImageName), tag)

	// Crear directorio temporal seguro en la VM
	buildID := fmt.Sprintf("docker-build-%d", time.Now().UnixNano())
	buildDir := fmt.Sprintf("/tmp/%s", buildID)

	// Garantizar limpieza del directorio temporal al finalizar
	defer func() {
		cleanupCmd := fmt.Sprintf("rm -rf %s", EscapeShellArg(buildDir))
		_, _ = s.runner(context.Background(), req.VMName, cleanupCmd)
	}()

	// Escribir el Dockerfile usando base64 para evitar inyección o corrupción de caracteres
	b64Content := base64.StdEncoding.EncodeToString([]byte(req.Dockerfile))
	writeCmd := fmt.Sprintf("mkdir -p %s && echo %s | base64 -d > %s/Dockerfile",
		EscapeShellArg(buildDir),
		EscapeShellArg(b64Content),
		EscapeShellArg(buildDir),
	)

	if _, err := s.runner(ctx, req.VMName, writeCmd); err != nil {
		return nil, fmt.Errorf("preparando contexto de compilación en VM '%s': %w", req.VMName, err)
	}

	// Escribir los archivos adicionales en el contexto de compilación dentro de la VM
	for _, vf := range validFiles {
		targetPath := fmt.Sprintf("%s/%s", buildDir, vf.relPath)
		targetDir := path.Dir(targetPath)
		writeTargetCmd := fmt.Sprintf("mkdir -p %s && echo %s | base64 -d > %s",
			EscapeShellArg(targetDir),
			EscapeShellArg(vf.content),
			EscapeShellArg(targetPath),
		)
		if _, err := s.runner(ctx, req.VMName, writeTargetCmd); err != nil {
			return nil, fmt.Errorf("escribiendo archivo '%s' en el contexto de compilación: %w", vf.relPath, err)
		}
	}

	// Ejecutar docker build
	buildCmd := fmt.Sprintf("sudo docker build -t %s %s 2>&1", EscapeShellArg(fullTag), EscapeShellArg(buildDir))
	s.log.Info("construyendo imagen Docker en VM", "vm", req.VMName, "tag", fullTag, "attached_files", len(validFiles))

	out, err := s.runner(ctx, req.VMName, buildCmd)
	if err != nil {
		userErrMsg := AnalyzeBuildError(out, fullTag)
		return &BuildImageResponse{
			Success: false,
			Image:   fullTag,
			Output:  out,
		}, fmt.Errorf("%s", userErrMsg)
	}

	return &BuildImageResponse{
		Success: true,
		Image:   fullTag,
		Output:  out,
	}, nil
}

// AnalyzeBuildError analiza la salida de docker build para generar un mensaje de error claro y descriptivo
func AnalyzeBuildError(output string, fullTag string) string {
	trimmed := strings.TrimSpace(output)
	lower := strings.ToLower(trimmed)

	// Detección de falta de archivos para COPY o ADD
	if strings.Contains(lower, "copy failed") ||
		strings.Contains(lower, "add failed") ||
		(strings.Contains(lower, "failed to compute cache key") && (strings.Contains(lower, "not found") || strings.Contains(lower, "no such file"))) ||
		(strings.Contains(lower, "failed to solve") && (strings.Contains(lower, "not found") || strings.Contains(lower, "no such file"))) ||
		strings.Contains(lower, "file not found in build context") ||
		(strings.Contains(lower, "stat ") && strings.Contains(lower, "no such file or directory")) {
		return fmt.Sprintf("Error compilando imagen '%s': falta un archivo en el contexto de compilación requerido por COPY o ADD en el Dockerfile. Asegúrate de adjuntar todos los archivos necesarios.", fullTag)
	}

	// Si hay un error general, extraer la última línea informativa
	lines := strings.Split(trimmed, "\n")
	var lastLine string
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l != "" {
			lastLine = l
			break
		}
	}
	if lastLine != "" {
		return fmt.Sprintf("Error compilando imagen '%s': %s", fullTag, lastLine)
	}
	return fmt.Sprintf("Error compilando imagen '%s'", fullTag)
}

// DeleteImage elimina una imagen Docker en la VM (con opción de forzado)
func (s *Service) DeleteImage(ctx context.Context, vmName string, imageRef string, force bool) error {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return fmt.Errorf("identificador de imagen no puede estar vacío")
	}
	if dangerousCharsRegex.MatchString(imageRef) {
		return fmt.Errorf("identificador de imagen contiene caracteres inválidos: %s", imageRef)
	}

	forceFlag := ""
	if force {
		forceFlag = "-f "
	}
	cmd := fmt.Sprintf("sudo docker rmi %s%s 2>&1", forceFlag, EscapeShellArg(imageRef))
	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		outTrim := strings.TrimSpace(out)
		if outTrim != "" {
			return fmt.Errorf("%s", outTrim)
		}
		return err
	}
	return nil
}

// CommitContainer crea una nueva imagen Docker a partir del estado actual de un contenedor en la VM
func (s *Service) CommitContainer(ctx context.Context, req CommitContainerRequest) (*CommitContainerResponse, error) {
	if strings.TrimSpace(req.VMName) == "" {
		return nil, fmt.Errorf("el nombre de la máquina virtual (VM) es obligatorio")
	}
	containerRef := strings.TrimSpace(req.ContainerID)
	if containerRef == "" {
		return nil, fmt.Errorf("el identificador o nombre del contenedor es obligatorio")
	}
	if dangerousCharsRegex.MatchString(containerRef) {
		return nil, fmt.Errorf("identificador de contenedor contiene caracteres inválidos: %s", containerRef)
	}

	if err := ValidateImageRepository(req.ImageName); err != nil {
		return nil, err
	}
	tag := strings.TrimSpace(req.Tag)
	if tag == "" {
		tag = "latest"
	}
	if err := ValidateImageTag(tag); err != nil {
		return nil, err
	}

	fullTag := fmt.Sprintf("%s:%s", strings.TrimSpace(req.ImageName), tag)

	var args []string
	args = append(args, "sudo", "docker", "commit")
	if strings.TrimSpace(req.Comment) != "" {
		if !dangerousCharsRegex.MatchString(req.Comment) {
			args = append(args, "-m", EscapeShellArg(req.Comment))
		}
	}
	if strings.TrimSpace(req.Author) != "" {
		if !dangerousCharsRegex.MatchString(req.Author) {
			args = append(args, "-a", EscapeShellArg(req.Author))
		}
	}
	args = append(args, EscapeShellArg(containerRef), EscapeShellArg(fullTag))

	cmd := strings.Join(args, " ")
	s.log.Info("creando imagen Docker desde contenedor en VM", "vm", req.VMName, "container", containerRef, "image", fullTag)

	out, err := s.runner(ctx, req.VMName, cmd+" 2>&1")
	if err != nil {
		return &CommitContainerResponse{
			Success: false,
			Image:   fullTag,
			Output:  out,
		}, fmt.Errorf("error creando imagen a partir del contenedor '%s': %s", containerRef, strings.TrimSpace(out))
	}

	return &CommitContainerResponse{
		Success: true,
		Image:   fullTag,
		ImageID: strings.TrimSpace(out),
		Output:  out,
	}, nil
}

// ListVolumes obtiene la lista de volúmenes Docker en la VM
func (s *Service) ListVolumes(ctx context.Context, vmName string) ([]Volume, error) {
	if strings.TrimSpace(vmName) == "" {
		return nil, fmt.Errorf("el nombre de la VM no puede estar vacío")
	}

	format := `{"name":{{json .Name}},"driver":{{json .Driver}},"mountpoint":{{json .Mountpoint}}}`
	cmd := fmt.Sprintf("sudo docker volume ls --format '%s'", format)

	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		return nil, fmt.Errorf("listando volúmenes en VM '%s': %w", vmName, err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var volumes []Volume
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var v Volume
		if err := json.Unmarshal([]byte(line), &v); err == nil {
			v.VMName = vmName
			volumes = append(volumes, v)
		}
	}

	// Obtener volúmenes en uso para marcar estado
	inUseCmd := `sudo docker ps -q | xargs -r sudo docker inspect -f '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{println}}{{end}}{{end}}'`
	inUseOut, _ := s.runner(ctx, vmName, inUseCmd)
	inUseSet := make(map[string]bool)
	for _, name := range strings.Split(strings.TrimSpace(inUseOut), "\n") {
		name = strings.TrimSpace(name)
		if name != "" {
			inUseSet[name] = true
		}
	}
	
	for i := range volumes {
		if inUseSet[volumes[i].Name] {
			volumes[i].Status = "En uso"
		} else {
			volumes[i].Status = "Sin utilizar"
		}
	}

	return volumes, nil
}

// CreateVolume crea un volumen en la VM
func (s *Service) CreateVolume(ctx context.Context, req CreateVolumeRequest) (*Volume, error) {
	if strings.TrimSpace(req.VMName) == "" {
		return nil, fmt.Errorf("el nombre de la VM es obligatorio")
	}
	if err := ValidateVolumeName(req.Name); err != nil {
		return nil, err
	}

	driver := strings.TrimSpace(req.Driver)
	if driver == "" {
		driver = "local"
	}

	cmd := fmt.Sprintf("sudo docker volume create --driver %s --name %s 2>&1", EscapeShellArg(driver), EscapeShellArg(req.Name))
	s.log.Info("creando volumen Docker en VM", "vm", req.VMName, "name", req.Name, "driver", driver)

	out, err := s.runner(ctx, req.VMName, cmd)
	if err != nil {
		return nil, fmt.Errorf("error creando volumen '%s': %s", req.Name, strings.TrimSpace(out))
	}

	return &Volume{
		Name: req.Name,
		Driver: driver,
		Status: "Sin utilizar",
		VMName: req.VMName,
	}, nil
}

// DeleteVolume elimina un volumen Docker en la VM
func (s *Service) DeleteVolume(ctx context.Context, vmName string, volumeName string) error {
	if err := ValidateVolumeName(volumeName); err != nil {
		return err
	}

	cmd := fmt.Sprintf("sudo docker volume rm %s 2>&1", EscapeShellArg(volumeName))
	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		outStr := strings.TrimSpace(out)
		if strings.Contains(outStr, "in use") {
			return fmt.Errorf("el volumen está asociado a uno o más contenedores y no puede ser eliminado")
		}
		return fmt.Errorf("error eliminando volumen: %s", outStr)
	}

	return nil
}

// ListNetworks lista las redes de la VM con información resumida y cantidad de contenedores
func (s *Service) ListNetworks(ctx context.Context, vmName string) ([]Network, error) {
	if strings.TrimSpace(vmName) == "" {
		return nil, fmt.Errorf("el nombre de la VM es obligatorio")
	}

	// Consultar redes en formato JSON por línea
	format := `{"id":{{json .ID}},"name":{{json .Name}},"driver":{{json .Driver}},"scope":{{json .Scope}}}`
	cmd := fmt.Sprintf("sudo docker network ls --format '%s'", format)

	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		return nil, fmt.Errorf("listando redes en VM '%s': %w", vmName, err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var networks []Network
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var netItem Network
		if err := json.Unmarshal([]byte(line), &netItem); err == nil {
			netItem.VMName = vmName
			if len(netItem.ID) > 12 {
				netItem.ID = netItem.ID[:12]
			}
			networks = append(networks, netItem)
		}
	}

	// Obtener detalles IPAM y contenedores conectados mediante docker network inspect masivo
	if len(networks) > 0 {
		var netNames []string
		for _, n := range networks {
			netNames = append(netNames, EscapeShellArg(n.Name))
		}
		inspectCmd := fmt.Sprintf("sudo docker network inspect %s 2>/dev/null", strings.Join(netNames, " "))
		inspOut, _ := s.runner(ctx, vmName, inspectCmd)
		if strings.TrimSpace(inspOut) != "" {
			var inspectDetails []struct {
				Name       string `json:"Name"`
				Internal   bool   `json:"Internal"`
				Attachable bool   `json:"Attachable"`
				IPAM       struct {
					Config []struct {
						Subnet  string `json:"Subnet"`
						Gateway string `json:"Gateway"`
						IPRange string `json:"IPRange"`
					} `json:"Config"`
				} `json:"IPAM"`
				Containers map[string]struct {
					Name string `json:"Name"`
				} `json:"Containers"`
			}
			if err := json.Unmarshal([]byte(inspOut), &inspectDetails); err == nil {
				detailMap := make(map[string]struct {
					subnet     string
					gateway    string
					ipRange    string
					internal   bool
					attachable bool
					count      int
					cNames     []string
				})
				for _, d := range inspectDetails {
					sub := ""
					gw := ""
					ipr := ""
					if len(d.IPAM.Config) > 0 {
						sub = d.IPAM.Config[0].Subnet
						gw = d.IPAM.Config[0].Gateway
						ipr = d.IPAM.Config[0].IPRange
					}
					var cNames []string
					for _, c := range d.Containers {
						cNames = append(cNames, c.Name)
					}
					detailMap[d.Name] = struct {
						subnet     string
						gateway    string
						ipRange    string
						internal   bool
						attachable bool
						count      int
						cNames     []string
					}{
						subnet:     sub,
						gateway:    gw,
						ipRange:    ipr,
						internal:   d.Internal,
						attachable: d.Attachable,
						count:      len(d.Containers),
						cNames:     cNames,
					}
				}

				for i := range networks {
					if info, ok := detailMap[networks[i].Name]; ok {
						networks[i].Subnet = info.subnet
						networks[i].Gateway = info.gateway
						networks[i].IPRange = info.ipRange
						networks[i].Internal = info.internal
						networks[i].Attachable = info.attachable
						networks[i].Containers = info.count
						networks[i].ContainerList = info.cNames
					}
				}
			}
		}
	}

	return networks, nil
}

// InspectNetwork devuelve los detalles completos de una red Docker
func (s *Service) InspectNetwork(ctx context.Context, vmName string, networkID string) (*Network, error) {
	if strings.TrimSpace(vmName) == "" {
		return nil, fmt.Errorf("el nombre de la VM es obligatorio")
	}
	if err := ValidateNetworkName(networkID); err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("sudo docker network inspect %s", EscapeShellArg(networkID))
	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		return nil, fmt.Errorf("inspeccionando red '%s' en VM '%s': %w", networkID, vmName, err)
	}

	var inspectList []struct {
		Id         string `json:"Id"`
		Name       string `json:"Name"`
		Driver     string `json:"Driver"`
		Scope      string `json:"Scope"`
		Internal   bool   `json:"Internal"`
		Attachable bool   `json:"Attachable"`
		Options    map[string]string `json:"Options"`
		IPAM       struct {
			Driver string `json:"Driver"`
			Config []struct {
				Subnet  string `json:"Subnet"`
				Gateway string `json:"Gateway"`
				IPRange string `json:"IPRange"`
			} `json:"Config"`
		} `json:"IPAM"`
		Containers map[string]struct {
			Name        string `json:"Name"`
			EndpointID  string `json:"EndpointID"`
			MacAddress  string `json:"MacAddress"`
			IPv4Address string `json:"IPv4Address"`
		} `json:"Containers"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &inspectList); err != nil || len(inspectList) == 0 {
		return nil, fmt.Errorf("fallo al parsear inspección de red: %w", err)
	}

	item := inspectList[0]
	netDetail := &Network{
		ID:         item.Id,
		Name:       item.Name,
		Driver:     item.Driver,
		Scope:      item.Scope,
		Internal:   item.Internal,
		Attachable: item.Attachable,
		Options:    item.Options,
		VMName:     vmName,
	}

	if len(item.IPAM.Config) > 0 {
		netDetail.Subnet = item.IPAM.Config[0].Subnet
		netDetail.Gateway = item.IPAM.Config[0].Gateway
		netDetail.IPRange = item.IPAM.Config[0].IPRange
	}

	netDetail.Containers = len(item.Containers)
	for _, c := range item.Containers {
		netDetail.ContainerList = append(netDetail.ContainerList, c.Name)
	}

	return netDetail, nil
}

// CreateNetwork crea una red en la VM especificada
func (s *Service) CreateNetwork(ctx context.Context, req CreateNetworkRequest) (*Network, error) {
	if strings.TrimSpace(req.VMName) == "" {
		return nil, fmt.Errorf("el nombre de la VM es obligatorio")
	}
	if err := ValidateNetworkName(req.Name); err != nil {
		return nil, err
	}
	if err := ValidateNetworkDriver(req.Driver); err != nil {
		return nil, err
	}

	// Validar IPAM si fue provisto
	if strings.TrimSpace(req.Subnet) != "" {
		if _, err := ValidateCIDR(req.Subnet); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(req.Gateway) != "" {
		if err := ValidateGatewaySubnet(req.Gateway, req.Subnet); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(req.IPRange) != "" {
		if err := ValidateIPRange(req.IPRange, req.Subnet); err != nil {
			return nil, err
		}
	}

	driver := strings.TrimSpace(req.Driver)
	if driver == "" {
		driver = "bridge"
	}

	var args []string
	args = append(args, "sudo", "docker", "network", "create", "--driver", EscapeShellArg(driver))

	if strings.TrimSpace(req.Subnet) != "" {
		args = append(args, "--subnet", EscapeShellArg(req.Subnet))
	}
	if strings.TrimSpace(req.Gateway) != "" {
		args = append(args, "--gateway", EscapeShellArg(req.Gateway))
	}
	if strings.TrimSpace(req.IPRange) != "" {
		args = append(args, "--ip-range", EscapeShellArg(req.IPRange))
	}
	if req.Internal {
		args = append(args, "--internal")
	}
	if req.Attachable {
		args = append(args, "--attachable")
	}

	args = append(args, EscapeShellArg(req.Name))
	cmd := strings.Join(args, " ") + " 2>&1"

	s.log.Info("creando red Docker en VM", "vm", req.VMName, "name", req.Name, "driver", driver)

	out, err := s.runner(ctx, req.VMName, cmd)
	if err != nil {
		outStr := strings.TrimSpace(out)
		if strings.Contains(outStr, "already exists") {
			return nil, fmt.Errorf("ya existe una red con el nombre '%s'", req.Name)
		}
		return nil, fmt.Errorf("error creando red '%s': %s", req.Name, outStr)
	}

	id := strings.TrimSpace(out)
	if len(id) > 12 {
		id = id[:12]
	}

	return &Network{
		ID:         id,
		Name:       req.Name,
		Driver:     driver,
		Scope:      "local",
		Subnet:     req.Subnet,
		Gateway:    req.Gateway,
		IPRange:    req.IPRange,
		Internal:   req.Internal,
		Attachable: req.Attachable,
		Containers: 0,
		VMName:     req.VMName,
	}, nil
}

// DeleteNetwork elimina una red Docker en la VM asegurando que no sea una de las redes predeterminadas
func (s *Service) DeleteNetwork(ctx context.Context, vmName string, networkName string) error {
	netLower := strings.ToLower(strings.TrimSpace(networkName))
	if netLower == "bridge" || netLower == "host" || netLower == "none" {
		return fmt.Errorf("no está permitido eliminar las redes predeterminadas del sistema ('%s')", networkName)
	}
	if err := ValidateNetworkName(networkName); err != nil {
		return err
	}

	cmd := fmt.Sprintf("sudo docker network rm %s 2>&1", EscapeShellArg(networkName))
	out, err := s.runner(ctx, vmName, cmd)
	if err != nil {
		outStr := strings.TrimSpace(out)
		if strings.Contains(outStr, "has active endpoints") || strings.Contains(outStr, "in use") {
			return fmt.Errorf("la red '%s' tiene contenedores conectados y no puede ser eliminada", networkName)
		}
		return fmt.Errorf("error eliminando red: %s", outStr)
	}

	return nil
}


