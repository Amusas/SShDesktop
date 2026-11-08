package docker

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ClusterStatusActive   = "Active"
	ClusterStatusCreating = "Creating"
	ClusterStatusDegraded = "Degraded"
	ClusterStatusError    = "Error"

	StepStatusPending    = "pending"
	StepStatusInProgress = "in_progress"
	StepStatusCompleted  = "completed"
	StepStatusFailed     = "failed"
	StepStatusWarning    = "warning"
	StepStatusSkipped    = "skipped"

	NodeRoleManager = "manager"
	NodeRoleWorker  = "worker"

	NodeSwarmReady        = "Ready"
	NodeSwarmDown         = "Down"
	NodeSwarmDisconnected = "Disconnected"
	NodeSwarmError        = "Error"
	NodeSwarmPending      = "Pending"
	NodeSwarmUnknown      = "Unknown"

	DefaultSwarmNetworkName = "vessel-net"
	DefaultSwarmNetworkCIDR = "10.250.0.0/24"
	SwarmManagementPort     = 2377
)

var validClusterNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}$`)

// ValidateClusterName verifica que el nombre del cluster sea seguro y válido.
func ValidateClusterName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("el nombre del cluster es obligatorio")
	}
	if !validClusterNameRegex.MatchString(trimmed) {
		return fmt.Errorf("nombre de cluster inválido %q: usa letras, números, guiones (-), guiones bajos (_) o puntos (máx. 63 caracteres)", name)
	}
	return nil
}

// SwarmVMInfo contiene la información de una máquina virtual de Vessel necesaria para Swarm y la UI.
type SwarmVMInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	TemplateID    string `json:"templateId,omitempty"`
	TemplateName  string `json:"templateName,omitempty"`
	OS            string `json:"os"`
	Cores         int    `json:"cores"`
	MemoryMB      int    `json:"memoryMB"`
	State         string `json:"state"`
	IP            string `json:"ip"`
	PrivateIP     string `json:"privateIp"`
	NetworkName   string `json:"networkName"`
	NetworkCIDR   string `json:"networkCidr"`
	MACAddress1   string `json:"macAddress1,omitempty"`
	MACAddress2   string `json:"macAddress2,omitempty"`
	Adapter1State string `json:"adapter1State,omitempty"`
	Adapter2State string `json:"adapter2State,omitempty"`
	Host          string `json:"host"`
	HostPort      int    `json:"hostPort"`
	GuestPort     int    `json:"guestPort"`
	SSHUser       string `json:"sshUser"`
	CreatedAt     string `json:"createdAt"`
}

// SwarmNode representa un nodo (Manager o Worker) dentro de un cluster Docker Swarm en Vessel.
type SwarmNode struct {
	VMID              string       `json:"vmId"`
	VMName            string       `json:"vmName"`
	NodeID            string       `json:"nodeId"`
	Hostname          string       `json:"hostname"`
	Role              string       `json:"role"` // "manager" | "worker"
	IsLeader          bool         `json:"isLeader"`
	InternalIP        string       `json:"internalIp"`
	VMState           string       `json:"vmState"`
	SwarmStatus       string       `json:"swarmStatus"`  // "Ready" | "Down" | "Disconnected" | "Error" | "Pending"
	Availability      string       `json:"availability"` // "Active" | "Drain" | "Pause"
	EngineVersion     string       `json:"engineVersion,omitempty"`
	JoinedAt          string       `json:"joinedAt,omitempty"`
	JoinError         string       `json:"joinError,omitempty"`
	NeedsLocalCleanup bool         `json:"needsLocalCleanup,omitempty"`
	VM                *SwarmVMInfo `json:"vm,omitempty"`
}

// SwarmCluster representa un cluster Docker Swarm persistido y administrado en Vessel.
type SwarmCluster struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	SwarmClusterID    string      `json:"swarmClusterId"`
	Status            string      `json:"status"` // "Active" | "Creating" | "Degraded" | "Error"
	StatusReason      string      `json:"statusReason,omitempty"`
	ManagerVMID       string      `json:"managerVmId"`
	ManagerVMName     string      `json:"managerVmName"`
	ManagerIP         string      `json:"managerIp"`
	WorkerVMIDs       []string    `json:"workerVmIds"`
	WorkerVMNames     []string    `json:"workerVmNames"`
	Nodes             []SwarmNode `json:"nodes"`
	NetworkName       string      `json:"networkName"`
	NetworkCIDR       string      `json:"networkCidr"`
	WorkersConnected  int         `json:"workersConnected"`
	WorkersTotal      int         `json:"workersTotal"`
	TotalNodes        int         `json:"totalNodes"`
	SingleNodeWarning string      `json:"singleNodeWarning,omitempty"`
	CreatedAt         string      `json:"createdAt"`
	UpdatedAt         string      `json:"updatedAt"`
	LastSyncAt        string      `json:"lastSyncAt,omitempty"`
}

// RecomputeCounters recalcula los contadores resumidos (WorkersConnected, WorkersTotal, TotalNodes) del cluster.
func (c *SwarmCluster) RecomputeCounters() {
	workersTotal := 0
	workersConnected := 0
	totalNodes := len(c.Nodes)
	var workerIDs []string
	var workerNames []string

	for _, n := range c.Nodes {
		if strings.EqualFold(n.Role, NodeRoleWorker) {
			workersTotal++
			if n.VMID != "" {
				workerIDs = append(workerIDs, n.VMID)
			}
			if n.VMName != "" {
				workerNames = append(workerNames, n.VMName)
			}
			if strings.EqualFold(n.SwarmStatus, NodeSwarmReady) &&
				(strings.EqualFold(n.VMState, "running") || strings.EqualFold(n.VMState, "ejecutando")) &&
				n.JoinError == "" {
				workersConnected++
			}
		}
	}

	c.WorkersTotal = workersTotal
	c.WorkersConnected = workersConnected
	c.TotalNodes = totalNodes
	if workerIDs != nil {
		c.WorkerVMIDs = workerIDs
	} else {
		c.WorkerVMIDs = []string{}
	}
	if workerNames != nil {
		c.WorkerVMNames = workerNames
	} else {
		c.WorkerVMNames = []string{}
	}
}

// VMSwarmEligibility describe si una VM existente en Vessel es elegible para ser Manager o Worker.
type VMSwarmEligibility struct {
	VMID            string `json:"vmId"`
	VMName          string `json:"vmName"`
	TemplateName    string `json:"templateName,omitempty"`
	Cores           int    `json:"cores"`
	MemoryMB        int    `json:"memoryMB"`
	State           string `json:"state"`
	PrivateIP       string `json:"privateIp"`
	NetworkName     string `json:"networkName"`
	NetworkCIDR     string `json:"networkCidr"`
	DockerAvailable bool   `json:"dockerAvailable"`
	DockerVersion   string `json:"dockerVersion,omitempty"`
	SwarmLocalState string `json:"swarmLocalState,omitempty"`
	Eligible        bool   `json:"eligible"`
	Reason          string `json:"reason"`
	ClusterID       string `json:"clusterId,omitempty"`
	ClusterName     string `json:"clusterName,omitempty"`
	NodeRole        string `json:"nodeRole,omitempty"`
	ExternalSwarm   bool   `json:"externalSwarm,omitempty"`
}

// CreateClusterRequest contiene los datos solicitados para crear un cluster Docker Swarm.
type CreateClusterRequest struct {
	Name      string   `json:"name"`
	ManagerVM string   `json:"managerVm"`
	WorkerVMs []string `json:"workerVms"`
}

// ClusterProgressEvent representa un evento de progreso paso a paso durante la creación de un cluster.
type ClusterProgressEvent struct {
	Stage   string        `json:"stage"`
	Step    int           `json:"step"`
	Total   int           `json:"total"`
	Status  string        `json:"status"` // "pending" | "in_progress" | "completed" | "failed" | "warning" | "skipped"
	Message string        `json:"message"`
	NodeVM  string        `json:"nodeVm,omitempty"`
	Error   string        `json:"error,omitempty"`
	Cluster *SwarmCluster `json:"cluster,omitempty"`
}

// RemoveNodeResult contiene el resultado de retirar un nodo (Worker o Manager) de un cluster.
type RemoveNodeResult struct {
	ClusterDeleted    bool          `json:"clusterDeleted"`
	Cluster           *SwarmCluster `json:"cluster,omitempty"`
	RemovedVM         string        `json:"removedVm"`
	Role              string        `json:"role"`
	Forced            bool          `json:"forced"`
	Warning           string        `json:"warning,omitempty"`
	Message           string        `json:"message"`
	WorkerUnreachable bool          `json:"workerUnreachable,omitempty"`
}
