package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

// ErrWorkerUnreachable indica que el Worker está apagado o sin conexión SSH y requiere confirmación para eliminación forzada.
var ErrWorkerUnreachable = errors.New("el nodo Worker no está accesible por SSH")

// SwarmVMProvider abstrae las operaciones de consulta y red sobre las máquinas virtuales existentes en Vessel.
type SwarmVMProvider interface {
	ListVMs() ([]SwarmVMInfo, error)
	GetVM(nameOrID string) (SwarmVMInfo, error)
	EnsureVMNetwork(ctx context.Context, name string) (SwarmVMInfo, error)
	CheckVMConnectivity(ctx context.Context, sourceVM, targetIP string, port int) (bool, string, error)
}

// SwarmService administra los clusters Docker Swarm sobre las VMs existentes de Vessel.
type SwarmService struct {
	runner     CommandRunner
	vmProvider SwarmVMProvider
	store      *ClusterStore
	log        *slog.Logger
}

// NewSwarmService crea una nueva instancia del servicio de clusters Docker Swarm.
func NewSwarmService(runner CommandRunner, vmProvider SwarmVMProvider, store *ClusterStore, logger *slog.Logger) *SwarmService {
	if logger == nil {
		logger = slog.Default()
	}
	if store == nil {
		store = NewClusterStore("clusters.json")
	}
	return &SwarmService{
		runner:     runner,
		vmProvider: vmProvider,
		store:      store,
		log:        logger,
	}
}

// Store devuelve el almacén de persistencia de clusters.
func (s *SwarmService) Store() *ClusterStore {
	return s.store
}

// dockerNodeLSItem modela cada línea JSON de `docker node ls --format '{{json .}}'`.
type dockerNodeLSItem struct {
	ID            string `json:"ID"`
	Hostname      string `json:"Hostname"`
	Status        string `json:"Status"`
	Availability  string `json:"Availability"`
	ManagerStatus string `json:"ManagerStatus"`
	EngineVersion string `json:"EngineVersion"`
}

// nodeInspectInfo contiene los datos clave de un nodo obtenidos con docker info.
type nodeDockerInfo struct {
	ServerVersion  string
	LocalNodeState string
	ClusterID      string
	NodeID         string
	Hostname       string
}

func isVMRunningState(state string) bool {
	st := strings.ToLower(strings.TrimSpace(state))
	return st == "running" || st == "ejecutando"
}

// validateVesselNetIP verifica que una dirección IP sea IPv4 válida y pertenezca a la red compartida vessel-net (y nunca a 10.0.2.x / 10.0.3.x).
func validateVesselNetIP(ipStr, expectedCIDR string) error {
	cleanIP := strings.TrimSpace(ipStr)
	if cleanIP == "" {
		return fmt.Errorf("la VM no tiene dirección IPv4 asignada en la red vessel-net")
	}
	parsed := net.ParseIP(cleanIP)
	if parsed == nil || parsed.To4() == nil {
		return fmt.Errorf("dirección IP inválida %q (se requiere IPv4)", cleanIP)
	}
	if parsed.IsLoopback() || parsed.IsUnspecified() {
		return fmt.Errorf("dirección IP %q no es válida para comunicación entre nodos", cleanIP)
	}
	// Rechazar explícitamente los rangos de Adaptador 2 NAT individual de VirtualBox
	if strings.HasPrefix(cleanIP, "10.0.2.") || strings.HasPrefix(cleanIP, "10.0.3.") {
		return fmt.Errorf("la IP %s pertenece al adaptador NAT de gestión SSH y no a la red privada compartida vessel-net", cleanIP)
	}

	cidr := strings.TrimSpace(expectedCIDR)
	if cidr == "" || strings.HasPrefix(cidr, "10.0.2.") || strings.HasPrefix(cidr, "10.0.3.") {
		cidr = DefaultSwarmNetworkCIDR
	}
	_, subnet, err := net.ParseCIDR(cidr)
	if err == nil && subnet != nil {
		if !subnet.Contains(parsed) {
			return fmt.Errorf("la IP %s no pertenece a la subred interna %s (%s)", cleanIP, DefaultSwarmNetworkName, cidr)
		}
	}
	return nil
}

// inspectNodeDocker consulta en una sola llamada SSH el estado de Docker y de Swarm de una VM.
func (s *SwarmService) inspectNodeDocker(ctx context.Context, vmName string) (nodeDockerInfo, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	cmd := "sudo docker info --format '{{.ServerVersion}}|{{.Swarm.LocalNodeState}}|{{.Swarm.Cluster.ID}}|{{.Swarm.NodeID}}|{{.Name}}'"
	out, err := s.runner(cmdCtx, vmName, cmd)
	if err != nil {
		return nodeDockerInfo{}, fmt.Errorf("no se pudo consultar Docker en la VM %q: %w", vmName, err)
	}

	line := strings.TrimSpace(out)
	if idx := strings.LastIndex(line, "\n"); idx != -1 {
		line = strings.TrimSpace(line[idx+1:])
	}
	parts := strings.Split(line, "|")
	if len(parts) < 4 || strings.TrimSpace(parts[0]) == "" {
		return nodeDockerInfo{}, fmt.Errorf("respuesta inesperada de Docker Engine en %q: %s", vmName, line)
	}

	info := nodeDockerInfo{
		ServerVersion:  strings.TrimSpace(parts[0]),
		LocalNodeState: strings.ToLower(strings.TrimSpace(parts[1])),
		ClusterID:      strings.TrimSpace(parts[2]),
		NodeID:         strings.TrimSpace(parts[3]),
	}
	if len(parts) >= 5 {
		info.Hostname = strings.TrimSpace(parts[4])
	}
	return info, nil
}

// GetVMEligibility evalúa todas las VMs existentes en Vessel e indica cuáles son elegibles para crear o unirse a un cluster Swarm.
func (s *SwarmService) GetVMEligibility(ctx context.Context) ([]VMSwarmEligibility, error) {
	vms, err := s.vmProvider.ListVMs()
	if err != nil {
		return nil, fmt.Errorf("error listando máquinas virtuales: %w", err)
	}

	results := make([]VMSwarmEligibility, 0, len(vms))
	for _, vm := range vms {
		netName := vm.NetworkName
		if netName == "" {
			netName = DefaultSwarmNetworkName
		}
		netCIDR := vm.NetworkCIDR
		if netCIDR == "" {
			netCIDR = DefaultSwarmNetworkCIDR
		}
		privIP := strings.TrimSpace(vm.PrivateIP)
		if privIP == "" {
			privIP = strings.TrimSpace(vm.IP)
		}

		item := VMSwarmEligibility{
			VMID:         vm.ID,
			VMName:       vm.Name,
			TemplateName: vm.TemplateName,
			Cores:        vm.Cores,
			MemoryMB:     vm.MemoryMB,
			State:        vm.State,
			PrivateIP:    privIP,
			NetworkName:  netName,
			NetworkCIDR:  netCIDR,
		}

		// 1. Verificar si ya pertenece a un cluster registrado en Vessel
		if cluster, node, found := s.store.FindClusterByVM(vm.ID, vm.Name); found {
			roleLabel := "Worker"
			if strings.EqualFold(node.Role, NodeRoleManager) {
				roleLabel = "Manager"
			}
			item.Eligible = false
			item.ClusterID = cluster.ID
			item.ClusterName = cluster.Name
			item.NodeRole = node.Role
			item.Reason = fmt.Sprintf("Ya pertenece al cluster '%s' como %s", cluster.Name, roleLabel)
			results = append(results, item)
			continue
		}

		// 2. Verificar si la VM está encendida
		if !isVMRunningState(vm.State) {
			stateLabel := vm.State
			if stateLabel == "" {
				stateLabel = "poweroff"
			}
			item.Eligible = false
			item.Reason = fmt.Sprintf("La máquina virtual está apagada (%s)", stateLabel)
			results = append(results, item)
			continue
		}

		// 3. Asegurar y validar dirección IPv4 en vessel-net
		if validateVesselNetIP(privIP, netCIDR) != nil {
			if refreshed, netErr := s.vmProvider.EnsureVMNetwork(ctx, vm.Name); netErr == nil {
				if refreshed.PrivateIP != "" {
					privIP = refreshed.PrivateIP
				} else if refreshed.IP != "" {
					privIP = refreshed.IP
				}
				item.PrivateIP = privIP
			}
		}
		if ipErr := validateVesselNetIP(privIP, netCIDR); ipErr != nil {
			item.Eligible = false
			item.Reason = ipErr.Error()
			results = append(results, item)
			continue
		}

		// 4. Verificar conexión SSH, Docker activo y que no pertenezca a otro Swarm externo
		dInfo, dErr := s.inspectNodeDocker(ctx, vm.Name)
		if dErr != nil {
			item.Eligible = false
			item.DockerAvailable = false
			item.Reason = "Sin conexión SSH o Docker Engine no está en ejecución"
			results = append(results, item)
			continue
		}

		item.DockerAvailable = true
		item.DockerVersion = dInfo.ServerVersion
		item.SwarmLocalState = dInfo.LocalNodeState

		if dInfo.LocalNodeState != "" && dInfo.LocalNodeState != "inactive" {
			item.Eligible = false
			item.ExternalSwarm = true
			item.Reason = fmt.Sprintf("Ya pertenece a un cluster Docker Swarm activo (estado: %s)", dInfo.LocalNodeState)
			results = append(results, item)
			continue
		}

		item.Eligible = true
		item.Reason = fmt.Sprintf("Elegible · %s (%s) · Docker %s", privIP, netName, dInfo.ServerVersion)
		results = append(results, item)
	}

	return results, nil
}

// CreateClusterWithProgress ejecuta el flujo real de 11 pasos para crear un cluster Docker Swarm y reporta cada etapa.
func (s *SwarmService) CreateClusterWithProgress(ctx context.Context, req CreateClusterRequest, onProgress func(ClusterProgressEvent)) (*SwarmCluster, error) {
	const totalSteps = 11
	emit := func(step int, stage, status, message string, extra ...func(*ClusterProgressEvent)) {
		if onProgress == nil {
			return
		}
		ev := ClusterProgressEvent{
			Stage:   stage,
			Step:    step,
			Total:   totalSteps,
			Status:  status,
			Message: message,
		}
		for _, fn := range extra {
			fn(&ev)
		}
		onProgress(ev)
	}

	clusterName := strings.TrimSpace(req.Name)
	managerRef := strings.TrimSpace(req.ManagerVM)

	// =========================================================================
	// PASO 1: Validando la configuración del cluster
	// =========================================================================
	emit(1, "validate_config", StepStatusInProgress, "Validando la configuración del cluster...")
	s.log.Info("Iniciando creación de cluster Docker Swarm", "cluster", clusterName, "manager", managerRef, "workersCount", len(req.WorkerVMs))

	if err := ValidateClusterName(clusterName); err != nil {
		emit(1, "validate_config", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
		return nil, err
	}
	if _, exists := s.store.Find(clusterName); exists {
		err := fmt.Errorf("ya existe un cluster llamado %q en Vessel", clusterName)
		emit(1, "validate_config", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
		return nil, err
	}
	if managerRef == "" {
		err := fmt.Errorf("debes seleccionar una máquina virtual para actuar como Manager")
		emit(1, "validate_config", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
		return nil, err
	}

	seenWorkers := make(map[string]bool)
	cleanWorkerRefs := make([]string, 0, len(req.WorkerVMs))
	for _, w := range req.WorkerVMs {
		wTrim := strings.TrimSpace(w)
		if wTrim == "" {
			continue
		}
		if strings.EqualFold(wTrim, managerRef) {
			err := fmt.Errorf("la máquina virtual %q fue seleccionada como Manager y no puede seleccionarse también como Worker", wTrim)
			emit(1, "validate_config", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
			return nil, err
		}
		key := strings.ToLower(wTrim)
		if seenWorkers[key] {
			err := fmt.Errorf("la máquina virtual Worker %q está duplicada en la selección", wTrim)
			emit(1, "validate_config", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
			return nil, err
		}
		seenWorkers[key] = true
		cleanWorkerRefs = append(cleanWorkerRefs, wTrim)
	}
	emit(1, "validate_config", StepStatusCompleted, "Configuración del cluster validada correctamente.")

	// =========================================================================
	// PASO 2: Verificando disponibilidad de las VMs seleccionadas
	// =========================================================================
	emit(2, "verify_vms", StepStatusInProgress, "Verificando disponibilidad y estado de las VMs seleccionadas...")

	managerVM, err := s.vmProvider.GetVM(managerRef)
	if err != nil {
		err = fmt.Errorf("no se encontró la VM Manager %q: %w", managerRef, err)
		emit(2, "verify_vms", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
		return nil, err
	}
	if existingCluster, _, inCluster := s.store.FindClusterByVM(managerVM.ID, managerVM.Name); inCluster {
		err = fmt.Errorf("la VM Manager %q ya pertenece al cluster %q", managerVM.Name, existingCluster.Name)
		emit(2, "verify_vms", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
		return nil, err
	}
	if !isVMRunningState(managerVM.State) {
		err = fmt.Errorf("la VM Manager %q no está en ejecución (estado actual: %s)", managerVM.Name, managerVM.State)
		emit(2, "verify_vms", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
		return nil, err
	}

	workerVMs := make([]SwarmVMInfo, 0, len(cleanWorkerRefs))
	for _, wRef := range cleanWorkerRefs {
		wVM, wErr := s.vmProvider.GetVM(wRef)
		if wErr != nil {
			err = fmt.Errorf("no se encontró la VM Worker %q: %w", wRef, wErr)
			emit(2, "verify_vms", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
			return nil, err
		}
		if wVM.ID == managerVM.ID || strings.EqualFold(wVM.Name, managerVM.Name) {
			err = fmt.Errorf("la VM %q no puede ser Manager y Worker simultáneamente", wVM.Name)
			emit(2, "verify_vms", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
			return nil, err
		}
		if existingCluster, _, inCluster := s.store.FindClusterByVM(wVM.ID, wVM.Name); inCluster {
			err = fmt.Errorf("la VM Worker %q ya pertenece al cluster %q", wVM.Name, existingCluster.Name)
			emit(2, "verify_vms", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
			return nil, err
		}
		if !isVMRunningState(wVM.State) {
			err = fmt.Errorf("la VM Worker %q no está en ejecución (estado actual: %s)", wVM.Name, wVM.State)
			emit(2, "verify_vms", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
			return nil, err
		}
		workerVMs = append(workerVMs, wVM)
	}
	emit(2, "verify_vms", StepStatusCompleted, fmt.Sprintf("VMs verificadas: Manager '%s' y %d Worker(s) en ejecución.", managerVM.Name, len(workerVMs)))

	// =========================================================================
	// PASO 3: Comprobando conexión SSH con los nodos
	// =========================================================================
	emit(3, "check_ssh_docker", StepStatusInProgress, "Comprobando conexión SSH y estado de Docker Engine en los nodos...")

	allVMs := append([]SwarmVMInfo{managerVM}, workerVMs...)
	nodeDockerMap := make(map[string]nodeDockerInfo, len(allVMs))
	for _, vm := range allVMs {
		dInfo, dErr := s.inspectNodeDocker(ctx, vm.Name)
		if dErr != nil {
			err = fmt.Errorf("falló la verificación SSH/Docker en la VM %q: %w", vm.Name, dErr)
			emit(3, "check_ssh_docker", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = vm.Name
				e.Error = err.Error()
			})
			return nil, err
		}
		if dInfo.LocalNodeState != "" && dInfo.LocalNodeState != "inactive" {
			err = fmt.Errorf("la VM %q ya forma parte de un cluster Docker Swarm activo (LocalNodeState=%s)", vm.Name, dInfo.LocalNodeState)
			emit(3, "check_ssh_docker", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = vm.Name
				e.Error = err.Error()
			})
			return nil, err
		}
		nodeDockerMap[vm.Name] = dInfo
	}
	emit(3, "check_ssh_docker", StepStatusCompleted, "Conexión SSH y Docker Engine verificados en todos los nodos.")

	// =========================================================================
	// PASO 4: Detectando las direcciones IP de vessel-net
	// =========================================================================
	emit(4, "detect_ips", StepStatusInProgress, "Detectando y validando direcciones IPv4 en la red compartida vessel-net...")

	nodeIPs := make(map[string]string, len(allVMs))
	usedIPs := make(map[string]string, len(allVMs))

	for i, vm := range allVMs {
		refreshedVM, netErr := s.vmProvider.EnsureVMNetwork(ctx, vm.Name)
		if netErr == nil {
			vm = refreshedVM
			if i == 0 {
				managerVM = refreshedVM
			} else {
				workerVMs[i-1] = refreshedVM
			}
		}
		ip := strings.TrimSpace(vm.PrivateIP)
		if ip == "" {
			ip = strings.TrimSpace(vm.IP)
		}
		if ipErr := validateVesselNetIP(ip, vm.NetworkCIDR); ipErr != nil {
			err = fmt.Errorf("error de red en la VM %q: %w", vm.Name, ipErr)
			emit(4, "detect_ips", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = vm.Name
				e.Error = err.Error()
			})
			return nil, err
		}
		if otherVM, dup := usedIPs[ip]; dup {
			err = fmt.Errorf("conflicto de dirección IP en vessel-net: las VMs %q y %q comparten la misma IP %s", otherVM, vm.Name, ip)
			emit(4, "detect_ips", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = vm.Name
				e.Error = err.Error()
			})
			return nil, err
		}
		usedIPs[ip] = vm.Name
		nodeIPs[vm.Name] = ip
	}
	managerIP := nodeIPs[managerVM.Name]
	emit(4, "detect_ips", StepStatusCompleted, fmt.Sprintf("Direcciones vessel-net confirmadas (Manager %s: %s).", managerVM.Name, managerIP))

	// =========================================================================
	// PASO 5: Comprobando conectividad entre los nodos
	// =========================================================================
	var singleNodeWarning string
	if len(workerVMs) == 0 {
		singleNodeWarning = "El cluster fue creado únicamente con el nodo Manager y aún no tiene capacidad distribuida."
		emit(5, "check_connectivity", StepStatusSkipped, "Cluster de un solo nodo (sin Workers): se omite prueba entre nodos. El cluster no tendrá capacidad distribuida todavía.")
	} else {
		emit(5, "check_connectivity", StepStatusInProgress, "Comprobando conectividad de red interna (vessel-net) entre Manager y Workers...")
		for _, wVM := range workerVMs {
			wIP := nodeIPs[wVM.Name]
			ok, detail, connErr := s.vmProvider.CheckVMConnectivity(ctx, wVM.Name, managerIP, 0)
			if connErr != nil || !ok {
				msg := fmt.Sprintf("el Worker %q (%s) no tiene conectividad en vessel-net hacia el Manager %q (%s)", wVM.Name, wIP, managerVM.Name, managerIP)
				if detail != "" {
					msg += ": " + detail
				} else if connErr != nil {
					msg += ": " + connErr.Error()
				}
				err = errors.New(msg)
				emit(5, "check_connectivity", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
					e.NodeVM = wVM.Name
					e.Error = err.Error()
				})
				return nil, err
			}
		}
		emit(5, "check_connectivity", StepStatusCompleted, "Conectividad IP en vessel-net verificada entre todos los nodos.")
	}

	// =========================================================================
	// PASO 6: Verificando disponibilidad de los puertos de Docker Swarm
	// =========================================================================
	emit(6, "check_ports", StepStatusInProgress, "Verificando disponibilidad de puertos Swarm (2377/tcp, 7946/tcp+udp, 4789/udp)...")
	portWarning := ""
	for _, vm := range allVMs {
		fwNote, pErr := s.verifyAndPrepareSwarmPorts(ctx, vm.Name)
		if pErr != nil {
			err = fmt.Errorf("error verificando puertos de Docker Swarm en la VM %q: %w", vm.Name, pErr)
			emit(6, "check_ports", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = vm.Name
				e.Error = err.Error()
			})
			return nil, err
		}
		if fwNote != "" {
			if portWarning != "" {
				portWarning += "; "
			}
			portWarning += fmt.Sprintf("%s: %s", vm.Name, fwNote)
		}
	}
	if portWarning != "" {
		emit(6, "check_ports", StepStatusWarning, fmt.Sprintf("Puertos Swarm verificados (%s). Nota: la recepción UDP 7946/4789 se valida parcialmente antes de iniciar el daemon Swarm.", portWarning))
	} else {
		emit(6, "check_ports", StepStatusCompleted, "Puertos 2377/tcp, 7946/tcp, 7946/udp y 4789/udp disponibles en todos los nodos (UDP verificado parcialmente a nivel de socket/firewall).")
	}

	// =========================================================================
	// PASO 7: Inicializando Docker Swarm en el Manager
	// =========================================================================
	emit(7, "swarm_init", StepStatusInProgress, fmt.Sprintf("Inicializando Docker Swarm en el Manager '%s' (%s)...", managerVM.Name, managerIP))

	initCmd := fmt.Sprintf("sudo docker swarm init --advertise-addr %s --data-path-addr %s",
		EscapeShellArg(managerIP), EscapeShellArg(managerIP))
	if out, initErr := s.runner(ctx, managerVM.Name, initCmd); initErr != nil {
		err = fmt.Errorf("falló 'docker swarm init' en el Manager %q (%s): %s (%w)", managerVM.Name, managerIP, strings.TrimSpace(out), initErr)
		s.log.Error("Error inicializando Docker Swarm en Manager", "vm", managerVM.Name, "ip", managerIP, "error", err)
		emit(7, "swarm_init", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
			e.NodeVM = managerVM.Name
			e.Error = err.Error()
		})
		return nil, err
	}

	mgrDockerInfo, err := s.inspectNodeDocker(ctx, managerVM.Name)
	if err != nil || mgrDockerInfo.LocalNodeState != "active" {
		if err == nil {
			err = fmt.Errorf("el Manager %q no quedó en estado Swarm activo (estado: %s)", managerVM.Name, mgrDockerInfo.LocalNodeState)
		}
		emit(7, "swarm_init", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
			e.NodeVM = managerVM.Name
			e.Error = err.Error()
		})
		return nil, err
	}

	// Verificar accesibilidad del puerto 2377/tcp desde los Workers una vez que el Manager está escuchando
	for _, wVM := range workerVMs {
		ok, detail, connErr := s.vmProvider.CheckVMConnectivity(ctx, wVM.Name, managerIP, SwarmManagementPort)
		if connErr != nil || !ok {
			err = fmt.Errorf("el puerto de gestión Swarm %s:%d no es accesible desde el Worker %q (%s)", managerIP, SwarmManagementPort, wVM.Name, detail)
			emit(7, "swarm_init", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = wVM.Name
				e.Error = err.Error()
			})
			return nil, err
		}
	}

	emit(7, "swarm_init", StepStatusCompleted, fmt.Sprintf("Manager '%s' inicializado como Leader en %s:2377.", managerVM.Name, managerIP))

	// =========================================================================
	// PASO 8: Generando y obteniendo el token de incorporación de Workers
	// =========================================================================
	var workerJoinToken string
	if len(workerVMs) == 0 {
		emit(8, "join_token", StepStatusSkipped, "No se seleccionaron Workers; se omite la solicitud del token de incorporación.")
	} else {
		emit(8, "join_token", StepStatusInProgress, "Obteniendo token de incorporación de Workers desde el Manager...")
		tokenOut, tokenErr := s.runner(ctx, managerVM.Name, "sudo docker swarm join-token -q worker")
		if tokenErr != nil {
			err = fmt.Errorf("no se pudo obtener el token de incorporación de Workers en el Manager %q: %w", managerVM.Name, tokenErr)
			emit(8, "join_token", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = managerVM.Name
				e.Error = err.Error()
			})
			return nil, err
		}
		workerJoinToken = strings.TrimSpace(tokenOut)
		if idx := strings.LastIndex(workerJoinToken, "\n"); idx != -1 {
			workerJoinToken = strings.TrimSpace(workerJoinToken[idx+1:])
		}
		if workerJoinToken == "" {
			err = fmt.Errorf("el Manager %q devolvió un token de incorporación vacío", managerVM.Name)
			emit(8, "join_token", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) {
				e.NodeVM = managerVM.Name
				e.Error = err.Error()
			})
			return nil, err
		}
		emit(8, "join_token", StepStatusCompleted, "Token de incorporación obtenido de forma segura en memoria.")
	}

	// =========================================================================
	// PASO 9: Incorporando los Workers al cluster
	// =========================================================================
	nowISO := time.Now().Format(time.RFC3339)
	mgrCopy := managerVM
	nodes := make([]SwarmNode, 0, 1+len(workerVMs))
	nodes = append(nodes, SwarmNode{
		VMID:          managerVM.ID,
		VMName:        managerVM.Name,
		NodeID:        mgrDockerInfo.NodeID,
		Hostname:      mgrDockerInfo.Hostname,
		Role:          NodeRoleManager,
		IsLeader:      true,
		InternalIP:    managerIP,
		VMState:       managerVM.State,
		SwarmStatus:   NodeSwarmReady,
		Availability:  "Active",
		EngineVersion: mgrDockerInfo.ServerVersion,
		JoinedAt:      nowISO,
		VM:            &mgrCopy,
	})

	var failedWorkers []string
	if len(workerVMs) == 0 {
		emit(9, "join_workers", StepStatusSkipped, "Sin nodos Worker por incorporar.")
	} else {
		emit(9, "join_workers", StepStatusInProgress, fmt.Sprintf("Incorporando %d nodo(s) Worker al cluster...", len(workerVMs)))
		managerAddr := fmt.Sprintf("%s:%d", managerIP, SwarmManagementPort)

		for idx, wVM := range workerVMs {
			wIP := nodeIPs[wVM.Name]
			wCopy := wVM
			wDocker := nodeDockerMap[wVM.Name]

			emit(9, "join_workers", StepStatusInProgress,
				fmt.Sprintf("Uniendo Worker %d/%d: '%s' (%s) → %s...", idx+1, len(workerVMs), wVM.Name, wIP, managerAddr),
				func(e *ClusterProgressEvent) { e.NodeVM = wVM.Name })

			joinCmd := fmt.Sprintf("sudo docker swarm join --token %s --advertise-addr %s --data-path-addr %s %s",
				EscapeShellArg(workerJoinToken),
				EscapeShellArg(wIP),
				EscapeShellArg(wIP),
				EscapeShellArg(managerAddr),
			)

			out, joinErr := s.runner(ctx, wVM.Name, joinCmd)
			if joinErr != nil {
				sanitizedOut := sanitizeToken(strings.TrimSpace(out), workerJoinToken)
				sanitizedErr := sanitizeToken(joinErr.Error(), workerJoinToken)
				errMsg := fmt.Sprintf("falló incorporación del Worker '%s' (%s): %s (%s)", wVM.Name, wIP, sanitizedOut, sanitizedErr)
				s.log.Warn("Fallo parcial incorporando Worker al cluster Swarm", "cluster", clusterName, "worker", wVM.Name, "ip", wIP, "error", errMsg)
				failedWorkers = append(failedWorkers, wVM.Name)
				nodes = append(nodes, SwarmNode{
					VMID:          wVM.ID,
					VMName:        wVM.Name,
					Hostname:      wDocker.Hostname,
					Role:          NodeRoleWorker,
					InternalIP:    wIP,
					VMState:       wVM.State,
					SwarmStatus:   NodeSwarmError,
					Availability:  "Unknown",
					EngineVersion: wDocker.ServerVersion,
					JoinError:     errMsg,
					VM:            &wCopy,
				})
				continue
			}

			wPostInfo, _ := s.inspectNodeDocker(ctx, wVM.Name)
			nodes = append(nodes, SwarmNode{
				VMID:          wVM.ID,
				VMName:        wVM.Name,
				NodeID:        wPostInfo.NodeID,
				Hostname:      wPostInfo.Hostname,
				Role:          NodeRoleWorker,
				InternalIP:    wIP,
				VMState:       wVM.State,
				SwarmStatus:   NodeSwarmReady,
				Availability:  "Active",
				EngineVersion: wDocker.ServerVersion,
				JoinedAt:      time.Now().Format(time.RFC3339),
				VM:            &wCopy,
			})
		}

		if len(failedWorkers) > 0 {
			warnMsg := fmt.Sprintf("Incorporación parcial: %d/%d Workers unidos. Fallaron: %s (puedes reintentar desde los detalles del cluster).",
				len(workerVMs)-len(failedWorkers), len(workerVMs), strings.Join(failedWorkers, ", "))
			emit(9, "join_workers", StepStatusWarning, warnMsg)
		} else {
			emit(9, "join_workers", StepStatusCompleted, fmt.Sprintf("Todos los Workers (%d) se unieron correctamente al cluster.", len(workerVMs)))
		}
	}

	// =========================================================================
	// PASO 10: Verificando el estado de los nodos
	// =========================================================================
	emit(10, "verify_nodes", StepStatusInProgress, "Verificando el estado real de los nodos en Docker Swarm (docker node ls)...")

	netName := managerVM.NetworkName
	if netName == "" {
		netName = DefaultSwarmNetworkName
	}
	netCIDR := managerVM.NetworkCIDR
	if netCIDR == "" {
		netCIDR = DefaultSwarmNetworkCIDR
	}

	cluster := &SwarmCluster{
		ID:                fmt.Sprintf("cluster-%d", time.Now().UnixNano()),
		Name:              clusterName,
		SwarmClusterID:    mgrDockerInfo.ClusterID,
		Status:            ClusterStatusActive,
		ManagerVMID:       managerVM.ID,
		ManagerVMName:     managerVM.Name,
		ManagerIP:         managerIP,
		Nodes:             nodes,
		NetworkName:       netName,
		NetworkCIDR:       netCIDR,
		SingleNodeWarning: singleNodeWarning,
		CreatedAt:         nowISO,
		UpdatedAt:         nowISO,
		LastSyncAt:        nowISO,
	}

	if err := s.refreshNodesFromManager(ctx, cluster); err != nil {
		s.log.Warn("Advertencia al consultar docker node ls tras crear cluster", "cluster", clusterName, "error", err)
		emit(10, "verify_nodes", StepStatusWarning, fmt.Sprintf("Nodos verificados con advertencia: %v", err))
	} else {
		if len(failedWorkers) > 0 {
			cluster.Status = ClusterStatusDegraded
			cluster.StatusReason = fmt.Sprintf("Fallo al incorporar Worker(s): %s", strings.Join(failedWorkers, ", "))
			emit(10, "verify_nodes", StepStatusWarning, fmt.Sprintf("Cluster en estado Degraded: %d nodo(s) Ready, %d Worker(s) con error de unión.", cluster.WorkersConnected+1, len(failedWorkers)))
		} else {
			emit(10, "verify_nodes", StepStatusCompleted, fmt.Sprintf("Estado verificado: %d nodo(s) Ready en el cluster.", len(cluster.Nodes)))
		}
	}
	cluster.RecomputeCounters()

	// =========================================================================
	// PASO 11: Guardando la información del cluster en Vessel
	// =========================================================================
	emit(11, "save_cluster", StepStatusInProgress, "Guardando la información del cluster en Vessel...")

	if err := s.store.Add(*cluster); err != nil {
		err = fmt.Errorf("el cluster Swarm se inicializó pero no se pudo guardar en clusters.json: %w", err)
		emit(11, "save_cluster", StepStatusFailed, err.Error(), func(e *ClusterProgressEvent) { e.Error = err.Error() })
		return nil, err
	}

	emit(11, "save_cluster", StepStatusCompleted, fmt.Sprintf("Cluster '%s' guardado exitosamente.", cluster.Name))
	emit(11, "done", StepStatusCompleted, fmt.Sprintf("¡Cluster Docker Swarm '%s' listo!", cluster.Name), func(e *ClusterProgressEvent) {
		e.Cluster = cluster
	})

	s.log.Info("Cluster Docker Swarm creado exitosamente",
		"id", cluster.ID,
		"name", cluster.Name,
		"status", cluster.Status,
		"manager", cluster.ManagerVMName,
		"workersConnected", cluster.WorkersConnected,
		"totalNodes", cluster.TotalNodes,
	)
	return cluster, nil
}

// verifyAndPrepareSwarmPorts comprueba en una VM que los puertos de Swarm no estén ocupados por otro proceso ajeno a Docker y que UFW (si está activo) permita 2377/tcp, 7946/tcp, 7946/udp y 4789/udp.
func (s *SwarmService) verifyAndPrepareSwarmPorts(ctx context.Context, vmName string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	script := `set -e
if command -v ss >/dev/null 2>&1; then
   CONFLICT=$(sudo ss -tlnp 2>/dev/null | grep -E ':(2377|7946)\s' | grep -v -E 'dockerd|docker' || true)
  if [ -n "$CONFLICT" ]; then
    echo "PORT_CONFLICT:$CONFLICT"
    exit 1
  fi
fi
if command -v ufw >/dev/null 2>&1; then
  UFW_STATE=$(sudo ufw status 2>/dev/null | head -n 1 || true)
  if echo "$UFW_STATE" | grep -qi "active"; then
    sudo ufw allow 2377/tcp >/dev/null 2>&1 || true
    sudo ufw allow 7946/tcp >/dev/null 2>&1 || true
    sudo ufw allow 7946/udp >/dev/null 2>&1 || true
    sudo ufw allow 4789/udp >/dev/null 2>&1 || true
    echo "UFW_RULES_UPDATED"
  fi
fi
echo "PORTS_OK"`

	out, err := s.runner(cmdCtx, vmName, script)
	if err != nil {
		if strings.Contains(out, "PORT_CONFLICT:") {
			return "", fmt.Errorf("los puertos 2377/tcp o 7946/tcp están ocupados por otro servicio: %s", strings.TrimSpace(out))
		}
		return "", fmt.Errorf("no se pudieron verificar los puertos de Swarm: %s (%w)", strings.TrimSpace(out), err)
	}
	if strings.Contains(out, "UFW_RULES_UPDATED") {
		return "reglas UFW habilitadas explícitamente para 2377/tcp, 7946/tcp+udp y 4789/udp", nil
	}
	return "", nil
}

// refreshNodesFromManager consulta `docker node ls` en el Manager y sincroniza el estado real de los nodos del cluster.
func (s *SwarmService) refreshNodesFromManager(ctx context.Context, cluster *SwarmCluster) error {
	cmdCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	out, err := s.runner(cmdCtx, cluster.ManagerVMName, "sudo docker node ls --format '{{json .}}'")
	if err != nil {
		return err
	}

	lsByNodeID := make(map[string]dockerNodeLSItem)
	lsByHost := make(map[string]dockerNodeLSItem)

	for _, rawLine := range strings.Split(out, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var item dockerNodeLSItem
		if jsonErr := json.Unmarshal([]byte(line), &item); jsonErr == nil && item.ID != "" {
			lsByNodeID[item.ID] = item
			lsByHost[strings.ToLower(strings.TrimSpace(item.Hostname))] = item
		}
	}

	allReady := true
	var reasons []string

	for i := range cluster.Nodes {
		n := &cluster.Nodes[i]
		// Actualizar datos de la VM desde el proveedor
		if vmInfo, vmErr := s.vmProvider.GetVM(n.VMName); vmErr == nil {
			n.VMState = vmInfo.State
			if vmInfo.PrivateIP != "" {
				n.InternalIP = vmInfo.PrivateIP
			}
			vmCopy := vmInfo
			n.VM = &vmCopy
		}

		var matched dockerNodeLSItem
		var found bool
		if n.NodeID != "" {
			matched, found = lsByNodeID[n.NodeID]
		}
		if !found && n.Hostname != "" {
			matched, found = lsByHost[strings.ToLower(n.Hostname)]
		}
		if !found && n.VMName != "" {
			matched, found = lsByHost[strings.ToLower(n.VMName)]
		}

		if found {
			n.NodeID = matched.ID
			if matched.Hostname != "" {
				n.Hostname = matched.Hostname
			}
			n.Availability = matched.Availability
			n.SwarmStatus = matched.Status
			if matched.EngineVersion != "" {
				n.EngineVersion = matched.EngineVersion
			}
			if strings.EqualFold(n.Role, NodeRoleManager) {
				n.IsLeader = strings.EqualFold(matched.ManagerStatus, "Leader")
			}
			n.JoinError = ""
		}

		if !isVMRunningState(n.VMState) {
			n.SwarmStatus = NodeSwarmDown
			allReady = false
			reasons = append(reasons, fmt.Sprintf("VM '%s' apagada", n.VMName))
		} else if n.JoinError != "" {
			n.SwarmStatus = NodeSwarmError
			allReady = false
			reasons = append(reasons, fmt.Sprintf("Worker '%s' con fallo de incorporación", n.VMName))
		} else if !strings.EqualFold(n.SwarmStatus, NodeSwarmReady) {
			allReady = false
			reasons = append(reasons, fmt.Sprintf("Nodo '%s' en estado %s", n.VMName, n.SwarmStatus))
		}
	}

	nowISO := time.Now().Format(time.RFC3339)
	cluster.LastSyncAt = nowISO
	cluster.UpdatedAt = nowISO
	if allReady {
		cluster.Status = ClusterStatusActive
		cluster.StatusReason = ""
	} else {
		cluster.Status = ClusterStatusDegraded
		cluster.StatusReason = strings.Join(reasons, "; ")
	}
	cluster.RecomputeCounters()
	return nil
}

// SyncCluster sincroniza el estado persistido en Vessel con el estado real de las VMs y de Docker Swarm.
func (s *SwarmService) SyncCluster(ctx context.Context, cluster *SwarmCluster) {
	if cluster == nil {
		return
	}

	// Refrescar estado de cada VM primero
	managerRunning := false
	for i := range cluster.Nodes {
		n := &cluster.Nodes[i]
		vmLookup := n.VMName
		if vmLookup == "" {
			vmLookup = n.VMID
		}
		if vmInfo, err := s.vmProvider.GetVM(vmLookup); err == nil {
			n.VMID = vmInfo.ID
			n.VMName = vmInfo.Name
			n.VMState = vmInfo.State
			if vmInfo.PrivateIP != "" {
				n.InternalIP = vmInfo.PrivateIP
			}
			vmCopy := vmInfo
			n.VM = &vmCopy
			if strings.EqualFold(n.Role, NodeRoleManager) {
				managerRunning = isVMRunningState(vmInfo.State)
				cluster.ManagerVMID = vmInfo.ID
				cluster.ManagerVMName = vmInfo.Name
				if vmInfo.PrivateIP != "" {
					cluster.ManagerIP = vmInfo.PrivateIP
				}
			}
		} else {
			n.VMState = "missing"
			n.SwarmStatus = NodeSwarmDisconnected
		}
	}

	if !managerRunning {
		for i := range cluster.Nodes {
			if !isVMRunningState(cluster.Nodes[i].VMState) {
				cluster.Nodes[i].SwarmStatus = NodeSwarmDown
			} else {
				cluster.Nodes[i].SwarmStatus = NodeSwarmUnknown
			}
		}
		cluster.Status = ClusterStatusDegraded
		cluster.StatusReason = fmt.Sprintf("La VM Manager '%s' está apagada", cluster.ManagerVMName)
		cluster.UpdatedAt = time.Now().Format(time.RFC3339)
		cluster.RecomputeCounters()
		_ = s.store.Update(*cluster)
		return
	}

	mgrInfo, err := s.inspectNodeDocker(ctx, cluster.ManagerVMName)
	if err != nil {
		cluster.Status = ClusterStatusError
		cluster.StatusReason = fmt.Sprintf("No se pudo conectar por SSH/Docker al Manager '%s'", cluster.ManagerVMName)
		cluster.RecomputeCounters()
		_ = s.store.Update(*cluster)
		return
	}
	if mgrInfo.LocalNodeState != "active" {
		cluster.Status = ClusterStatusError
		cluster.StatusReason = fmt.Sprintf("El Manager '%s' ya no tiene Docker Swarm activo (estado: %s)", cluster.ManagerVMName, mgrInfo.LocalNodeState)
		cluster.RecomputeCounters()
		_ = s.store.Update(*cluster)
		return
	}
	if mgrInfo.ClusterID != "" {
		cluster.SwarmClusterID = mgrInfo.ClusterID
	}

	if err := s.refreshNodesFromManager(ctx, cluster); err != nil {
		cluster.Status = ClusterStatusDegraded
		cluster.StatusReason = fmt.Sprintf("No se pudo listar los nodos desde el Manager '%s': %v", cluster.ManagerVMName, err)
	}
	if len(cluster.Nodes) == 1 {
		cluster.SingleNodeWarning = "El cluster opera únicamente con el nodo Manager y aún no tiene capacidad distribuida."
	} else {
		cluster.SingleNodeWarning = ""
	}
	cluster.RecomputeCounters()
	_ = s.store.Update(*cluster)
}

// ListClusters devuelve todos los clusters registrados sincronizando su estado actual.
func (s *SwarmService) ListClusters(ctx context.Context) ([]SwarmCluster, error) {
	clusters, err := s.store.Load()
	if err != nil {
		return nil, fmt.Errorf("error cargando clusters: %w", err)
	}
	for i := range clusters {
		s.SyncCluster(ctx, &clusters[i])
	}
	return clusters, nil
}

// GetCluster obtiene el detalle actualizado de un cluster por su ID o nombre.
func (s *SwarmService) GetCluster(ctx context.Context, idOrName string) (*SwarmCluster, error) {
	cluster, found := s.store.Find(idOrName)
	if !found {
		return nil, fmt.Errorf("cluster %q no encontrado", idOrName)
	}
	s.SyncCluster(ctx, &cluster)
	return &cluster, nil
}

// RetryOrAddWorker reintenta incorporar un Worker con error (o incorpora una VM elegible como Worker) a un cluster existente.
func (s *SwarmService) RetryOrAddWorker(ctx context.Context, clusterIDOrName, vmIDOrName string) (*SwarmCluster, error) {
	cluster, found := s.store.Find(clusterIDOrName)
	if !found {
		return nil, fmt.Errorf("cluster %q no encontrado", clusterIDOrName)
	}

	mgrVM, err := s.vmProvider.GetVM(cluster.ManagerVMName)
	if err != nil || !isVMRunningState(mgrVM.State) {
		return nil, fmt.Errorf("la VM Manager %q debe estar encendida para incorporar un Worker", cluster.ManagerVMName)
	}

	workerVM, err := s.vmProvider.GetVM(vmIDOrName)
	if err != nil {
		return nil, fmt.Errorf("no se encontró la VM Worker %q: %w", vmIDOrName, err)
	}
	if !isVMRunningState(workerVM.State) {
		return nil, fmt.Errorf("la VM Worker %q está apagada; iníciala antes de incorporarla al cluster", workerVM.Name)
	}

	if otherCluster, _, inOther := s.store.FindClusterByVM(workerVM.ID, workerVM.Name); inOther && otherCluster.ID != cluster.ID {
		return nil, fmt.Errorf("la VM %q ya pertenece al cluster %q", workerVM.Name, otherCluster.Name)
	}

	if refreshed, netErr := s.vmProvider.EnsureVMNetwork(ctx, workerVM.Name); netErr == nil {
		workerVM = refreshed
	}
	workerIP := strings.TrimSpace(workerVM.PrivateIP)
	if workerIP == "" {
		workerIP = strings.TrimSpace(workerVM.IP)
	}
	if ipErr := validateVesselNetIP(workerIP, workerVM.NetworkCIDR); ipErr != nil {
		return nil, fmt.Errorf("error de red en Worker %q: %w", workerVM.Name, ipErr)
	}

	mgrIP := strings.TrimSpace(cluster.ManagerIP)
	if mgrIP == "" {
		mgrIP = strings.TrimSpace(mgrVM.PrivateIP)
	}
	if ipErr := validateVesselNetIP(mgrIP, mgrVM.NetworkCIDR); ipErr != nil {
		return nil, fmt.Errorf("error de red en Manager %q: %w", mgrVM.Name, ipErr)
	}

	if _, pErr := s.verifyAndPrepareSwarmPorts(ctx, workerVM.Name); pErr != nil {
		return nil, pErr
	}

	// Si el Worker quedó en estado Swarm pendiente o roto, limpiarlo antes de reintentar
	wDocker, err := s.inspectNodeDocker(ctx, workerVM.Name)
	if err != nil {
		return nil, err
	}
	if wDocker.LocalNodeState != "" && wDocker.LocalNodeState != "inactive" {
		if wDocker.ClusterID == cluster.SwarmClusterID && wDocker.ClusterID != "" {
			// Ya está unido a este cluster; solo sincronizar
			s.SyncCluster(ctx, &cluster)
			return &cluster, nil
		}
		return nil, fmt.Errorf("la VM %q ya pertenece a otro estado Swarm (%s)", workerVM.Name, wDocker.LocalNodeState)
	}

	tokenOut, err := s.runner(ctx, mgrVM.Name, "sudo docker swarm join-token -q worker")
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el token de incorporación desde el Manager %q: %w", mgrVM.Name, err)
	}
	token := strings.TrimSpace(tokenOut)
	if idx := strings.LastIndex(token, "\n"); idx != -1 {
		token = strings.TrimSpace(token[idx+1:])
	}
	if token == "" {
		return nil, fmt.Errorf("el Manager devolvió un token de incorporación vacío")
	}

	managerAddr := fmt.Sprintf("%s:%d", mgrIP, SwarmManagementPort)
	joinCmd := fmt.Sprintf("sudo docker swarm join --token %s --advertise-addr %s --data-path-addr %s %s",
		EscapeShellArg(token),
		EscapeShellArg(workerIP),
		EscapeShellArg(workerIP),
		EscapeShellArg(managerAddr),
	)
	if out, joinErr := s.runner(ctx, workerVM.Name, joinCmd); joinErr != nil {
		sanitizedOut := sanitizeToken(strings.TrimSpace(out), token)
		sanitizedErr := sanitizeToken(joinErr.Error(), token)
		return nil, fmt.Errorf("falló la incorporación del Worker %q (%s): %s (%s)", workerVM.Name, workerIP, sanitizedOut, sanitizedErr)
	}

	wPost, _ := s.inspectNodeDocker(ctx, workerVM.Name)
	wCopy := workerVM
	updatedExisting := false
	for i := range cluster.Nodes {
		if cluster.Nodes[i].VMID == workerVM.ID || strings.EqualFold(cluster.Nodes[i].VMName, workerVM.Name) {
			cluster.Nodes[i].NodeID = wPost.NodeID
			cluster.Nodes[i].Hostname = wPost.Hostname
			cluster.Nodes[i].InternalIP = workerIP
			cluster.Nodes[i].VMState = workerVM.State
			cluster.Nodes[i].SwarmStatus = NodeSwarmReady
			cluster.Nodes[i].Availability = "Active"
			cluster.Nodes[i].EngineVersion = wDocker.ServerVersion
			cluster.Nodes[i].JoinError = ""
			cluster.Nodes[i].JoinedAt = time.Now().Format(time.RFC3339)
			cluster.Nodes[i].VM = &wCopy
			updatedExisting = true
			break
		}
	}
	if !updatedExisting {
		cluster.Nodes = append(cluster.Nodes, SwarmNode{
			VMID:          workerVM.ID,
			VMName:        workerVM.Name,
			NodeID:        wPost.NodeID,
			Hostname:      wPost.Hostname,
			Role:          NodeRoleWorker,
			InternalIP:    workerIP,
			VMState:       workerVM.State,
			SwarmStatus:   NodeSwarmReady,
			Availability:  "Active",
			EngineVersion: wDocker.ServerVersion,
			JoinedAt:      time.Now().Format(time.RFC3339),
			VM:            &wCopy,
		})
	}

	s.SyncCluster(ctx, &cluster)
	return &cluster, nil
}

// RemoveNodeFromCluster retira de forma controlada un nodo Worker o el Manager de un cluster Docker Swarm.
func (s *SwarmService) RemoveNodeFromCluster(ctx context.Context, clusterIDOrName, vmIDOrName string, force bool) (*RemoveNodeResult, error) {
	cluster, found := s.store.Find(clusterIDOrName)
	if !found {
		return nil, fmt.Errorf("cluster %q no encontrado", clusterIDOrName)
	}

	targetRef := strings.TrimSpace(vmIDOrName)
	var targetNode *SwarmNode
	targetIdx := -1
	for i := range cluster.Nodes {
		n := &cluster.Nodes[i]
		if n.VMID == targetRef || strings.EqualFold(n.VMName, targetRef) {
			targetNode = n
			targetIdx = i
			break
		}
	}
	if targetNode == nil {
		return nil, fmt.Errorf("la máquina virtual %q no pertenece al cluster %q", targetRef, cluster.Name)
	}

	// =========================================================================
	// Caso 1: Eliminación del Manager
	// =========================================================================
	if strings.EqualFold(targetNode.Role, NodeRoleManager) {
		activeWorkers := 0
		for _, n := range cluster.Nodes {
			if strings.EqualFold(n.Role, NodeRoleWorker) {
				activeWorkers++
			}
		}
		if activeWorkers > 0 {
			return nil, fmt.Errorf("no se puede eliminar el único Manager (%q) mientras existan %d Worker(s) en el cluster %q. Retira primero todos los Workers", targetNode.VMName, activeWorkers, cluster.Name)
		}

		mgrVM, vmErr := s.vmProvider.GetVM(targetNode.VMName)
		mgrRunning := vmErr == nil && isVMRunningState(mgrVM.State)
		var warningMsg string

		if mgrRunning {
			s.log.Info("Ejecutando docker swarm leave --force en el último nodo Manager", "cluster", cluster.Name, "manager", targetNode.VMName)
			out, leaveErr := s.runner(ctx, targetNode.VMName, "sudo docker swarm leave --force")
			if leaveErr != nil && !strings.Contains(strings.ToLower(out), "not part of a swarm") {
				if !force {
					return nil, fmt.Errorf("no se pudo retirar el Manager %q de Docker Swarm: %s (%w)", targetNode.VMName, strings.TrimSpace(out), leaveErr)
				}
				warningMsg = fmt.Sprintf("Se eliminó el registro del cluster forzadamente, pero 'docker swarm leave --force' devolvió: %s", strings.TrimSpace(out))
			} else {
				// Verificar que efectivamente salió de Swarm
				if dInfo, dErr := s.inspectNodeDocker(ctx, targetNode.VMName); dErr == nil && dInfo.LocalNodeState == "active" {
					return nil, fmt.Errorf("la VM Manager %q continúa reportando LocalNodeState=active tras ejecutar swarm leave", targetNode.VMName)
				}
			}
		} else {
			if !force {
				return &RemoveNodeResult{
					RemovedVM:         targetNode.VMName,
					Role:              NodeRoleManager,
					WorkerUnreachable: true,
					Message:           fmt.Sprintf("La VM Manager '%s' está apagada o inaccesible. ¿Deseas forzar la eliminación del cluster en Vessel? (Cuando la VM encienda deberás ejecutar 'sudo docker swarm leave --force').", targetNode.VMName),
				}, ErrWorkerUnreachable
			}
			warningMsg = fmt.Sprintf("La VM Manager '%s' estaba apagada cuando se eliminó el cluster. Cuando vuelva a encenderse, ejecuta 'sudo docker swarm leave --force' si continúa en modo Swarm.", targetNode.VMName)
		}

		if err := s.store.Delete(cluster.ID); err != nil {
			return nil, fmt.Errorf("error eliminando el registro del cluster %q: %w", cluster.Name, err)
		}

		s.log.Info("Manager retirado y cluster eliminado de Vessel", "cluster", cluster.Name, "manager", targetNode.VMName, "forced", force)
		return &RemoveNodeResult{
			ClusterDeleted: true,
			RemovedVM:      targetNode.VMName,
			Role:           NodeRoleManager,
			Forced:         force,
			Warning:        warningMsg,
			Message:        fmt.Sprintf("El Manager '%s' salió de Docker Swarm y el cluster '%s' fue eliminado de Vessel.", targetNode.VMName, cluster.Name),
		}, nil
	}

	// =========================================================================
	// Caso 2: Eliminación de un Worker
	// =========================================================================
	// 1. Comprobar disponibilidad del Manager
	mgrVM, mgrErr := s.vmProvider.GetVM(cluster.ManagerVMName)
	if mgrErr != nil || !isVMRunningState(mgrVM.State) {
		return nil, fmt.Errorf("el nodo Manager %q debe estar encendido y accesible para retirar un Worker del cluster", cluster.ManagerVMName)
	}

	// Sincronizar NodeID desde el Manager antes de operar
	_ = s.refreshNodesFromManager(ctx, &cluster)
	targetNode = &cluster.Nodes[targetIdx]
	nodeID := strings.TrimSpace(targetNode.NodeID)

	// Si el Worker había fallado durante la unión inicial y ni siquiera está registrado en `docker node ls`
	if nodeID == "" && targetNode.JoinError != "" {
		if wVM, wErr := s.vmProvider.GetVM(targetNode.VMName); wErr == nil && isVMRunningState(wVM.State) {
			_, _ = s.runner(ctx, targetNode.VMName, "sudo docker swarm leave --force 2>/dev/null || true")
		}
		cluster.Nodes = append(cluster.Nodes[:targetIdx], cluster.Nodes[targetIdx+1:]...)
		s.SyncCluster(ctx, &cluster)
		return &RemoveNodeResult{
			ClusterDeleted: false,
			Cluster:        &cluster,
			RemovedVM:      targetNode.VMName,
			Role:           NodeRoleWorker,
			Forced:         force,
			Message:        fmt.Sprintf("El Worker '%s' (con unión fallida previa) fue retirado del cluster '%s'.", targetNode.VMName, cluster.Name),
		}, nil
	}

	// 2. Comprobar estado y conectividad SSH del Worker
	wVM, wErr := s.vmProvider.GetVM(targetNode.VMName)
	workerReachable := false
	if wErr == nil && isVMRunningState(wVM.State) {
		if _, dErr := s.inspectNodeDocker(ctx, targetNode.VMName); dErr == nil {
			workerReachable = true
		}
	}

	if !workerReachable && !force {
		return &RemoveNodeResult{
			RemovedVM:         targetNode.VMName,
			Role:              NodeRoleWorker,
			WorkerUnreachable: true,
			Message:           fmt.Sprintf("El Worker '%s' está apagado o sin conexión SSH. Puedes confirmar la eliminación forzada desde el Manager (cuando la VM vuelva a encenderse podría requerir limpieza local con 'sudo docker swarm leave --force').", targetNode.VMName),
		}, ErrWorkerUnreachable
	}

	var warningMsg string

	// 3. Poner el nodo en modo drain desde el Manager (si tiene NodeID en el Manager)
	if nodeID != "" {
		drainCmd := fmt.Sprintf("sudo docker node update --availability drain %s", EscapeShellArg(nodeID))
		if out, drainErr := s.runner(ctx, cluster.ManagerVMName, drainCmd); drainErr != nil {
			s.log.Warn("Advertencia al poner nodo en drain", "cluster", cluster.Name, "worker", targetNode.VMName, "nodeId", nodeID, "output", strings.TrimSpace(out))
		}
	}

	// 4. Hacer que el Worker abandone el cluster (si está accesible)
	if workerReachable {
		out, leaveErr := s.runner(ctx, targetNode.VMName, "sudo docker swarm leave")
		if leaveErr != nil {
			// Intentar con --force por si Docker exige forzar la salida
			outForce, forceErr := s.runner(ctx, targetNode.VMName, "sudo docker swarm leave --force")
			if forceErr != nil && !strings.Contains(strings.ToLower(outForce), "not part of a swarm") {
				if !force {
					return nil, fmt.Errorf("error ejecutando 'docker swarm leave' en el Worker %q: %s (%w)", targetNode.VMName, strings.TrimSpace(out), leaveErr)
				}
				warningMsg = fmt.Sprintf("El Worker '%s' reportó un error al ejecutar 'docker swarm leave' (%s); se forzó su baja en el Manager.", targetNode.VMName, strings.TrimSpace(outForce))
			}
		}
	} else {
		warningMsg = fmt.Sprintf("El Worker '%s' fue eliminado forzadamente desde el Manager mientras estaba desconectado. Cuando la VM vuelva a encenderse, ejecuta 'sudo docker swarm leave --force' si aún conserva estado Swarm local.", targetNode.VMName)
	}

	// 5. Eliminar el registro del nodo en el Manager
	if nodeID != "" {
		rmCmd := fmt.Sprintf("sudo docker node rm %s", EscapeShellArg(nodeID))
		if force || !workerReachable {
			rmCmd = fmt.Sprintf("sudo docker node rm --force %s", EscapeShellArg(nodeID))
		}
		if out, rmErr := s.runner(ctx, cluster.ManagerVMName, rmCmd); rmErr != nil {
			// Reintentar con --force si el Manager aún detecta el nodo en transición a Down
			rmForceCmd := fmt.Sprintf("sudo docker node rm --force %s", EscapeShellArg(nodeID))
			if out2, rmErr2 := s.runner(ctx, cluster.ManagerVMName, rmForceCmd); rmErr2 != nil && !strings.Contains(strings.ToLower(out2), "no such node") {
				return nil, fmt.Errorf("no se pudo eliminar el nodo %q (%s) en el Manager %q: %s (%w)", targetNode.VMName, nodeID, cluster.ManagerVMName, strings.TrimSpace(out), rmErr)
			}
		}

		// 6. Verificar que el nodo ya no figure en `docker node ls`
		lsOut, lsErr := s.runner(ctx, cluster.ManagerVMName, "sudo docker node ls --format '{{.ID}}'")
		if lsErr == nil {
			for _, idLine := range strings.Split(lsOut, "\n") {
				if strings.TrimSpace(idLine) == nodeID {
					return nil, fmt.Errorf("el nodo %q (%s) aún aparece en 'docker node ls' después de solicitar su eliminación", targetNode.VMName, nodeID)
				}
			}
		}
	}

	// 7. Actualizar el estado persistido en Vessel
	removedName := targetNode.VMName
	cluster.Nodes = append(cluster.Nodes[:targetIdx], cluster.Nodes[targetIdx+1:]...)
	s.SyncCluster(ctx, &cluster)

	s.log.Info("Worker eliminado del cluster Docker Swarm",
		"cluster", cluster.Name,
		"worker", removedName,
		"nodeId", nodeID,
		"forced", force || !workerReachable,
	)

	return &RemoveNodeResult{
		ClusterDeleted: false,
		Cluster:        &cluster,
		RemovedVM:      removedName,
		Role:           NodeRoleWorker,
		Forced:         force || !workerReachable,
		Warning:        warningMsg,
		Message:        fmt.Sprintf("La máquina virtual '%s' fue retirada del cluster '%s' correctamente (la VM y sus datos se conservan intactos).", removedName, cluster.Name),
	}, nil
}

// sanitizeToken oculta cualquier aparición del token Swarm en salidas de error o logs.
func sanitizeToken(text, token string) string {
	if token == "" || text == "" {
		return text
	}
	return strings.ReplaceAll(text, token, "[TOKEN-REDACTADO]")
}
