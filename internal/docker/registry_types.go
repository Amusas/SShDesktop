package docker

const (
	// DefaultRegistryImage es la versión estable y explícita del Distribution Registry oficial.
	DefaultRegistryImage = "registry:2.8.3"

	// Instancia 1: Registry local escribible
	DefaultLocalRegistryContainer = "vessel-registry"
	DefaultLocalRegistryVolume    = "vessel-registry-data"
	DefaultLocalRegistryPort      = 5000

	// Instancia 2: Pull-through Cache de Docker Hub (independiente)
	DefaultCacheRegistryContainer = "vessel-registry-cache"
	DefaultCacheRegistryVolume    = "vessel-registry-cache-data"
	DefaultCacheRegistryPort      = 5001
	DefaultCacheRemoteURL         = "https://registry-1.docker.io"

	RegistryTargetLocal = "local"
	RegistryTargetCache = "cache"
)

// RegistryInstanceConfig almacena la configuración persistida de una instancia de Registry (local o cache).
type RegistryInstanceConfig struct {
	Target        string `json:"target"`        // "local" o "cache"
	VMID          string `json:"vmId"`          // ID de la VM anfitriona
	VMName        string `json:"vmName"`        // Nombre de la VM anfitriona
	HostIP        string `json:"hostIp"`        // IP accesible por los demás nodos (vessel-net)
	Port          int    `json:"port"`          // Puerto publicado en el host (ej. 5000 o 5001)
	ContainerPort int    `json:"containerPort"` // Puerto interno del contenedor (5000)
	ContainerName string `json:"containerName"` // ej. "vessel-registry" o "vessel-registry-cache"
	VolumeName    string `json:"volumeName"`    // ej. "vessel-registry-data" o "vessel-registry-cache-data"
	Image         string `json:"image"`         // ej. "registry:2.8.3"
	RemoteURL     string `json:"remoteUrl,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// RegistryPersistedState representa el archivo registry.json almacenado por Vessel.
type RegistryPersistedState struct {
	Local *RegistryInstanceConfig `json:"local,omitempty"`
	Cache *RegistryInstanceConfig `json:"cache,omitempty"`
}

// RegistryInstanceStatus expone la configuración y el estado en vivo de una instancia de Registry.
type RegistryInstanceStatus struct {
	Target         string `json:"target"`                   // "local" o "cache"
	Configured     bool   `json:"configured"`               // Si existe configuración registrada o detectada
	VMID           string `json:"vmId,omitempty"`           // ID de la VM
	VMName         string `json:"vmName,omitempty"`         // Nombre de la VM donde reside
	VMState        string `json:"vmState,omitempty"`        // running, poweroff, etc.
	HostIP         string `json:"hostIp,omitempty"`         // IP accesible desde otros nodos
	Port           int    `json:"port"`                     // 5000 o 5001
	ContainerPort  int    `json:"containerPort"`            // 5000
	ContainerName  string `json:"containerName"`            // vessel-registry / vessel-registry-cache
	ContainerID    string `json:"containerId,omitempty"`    // ID corto del contenedor
	ContainerState string `json:"containerState,omitempty"` // running, exited, missing, unknown
	VolumeName     string `json:"volumeName"`               // vessel-registry-data / vessel-registry-cache-data
	Image          string `json:"image"`                    // registry:2.8.3
	RemoteURL      string `json:"remoteUrl,omitempty"`      // https://registry-1.docker.io (solo cache)
	RestartPolicy  string `json:"restartPolicy,omitempty"`  // unless-stopped
	EndpointHost   string `json:"endpointHost,omitempty"`   // ej. "10.250.0.5:5000"
	EndpointURL    string `json:"endpointUrl,omitempty"`    // ej. "http://10.250.0.5:5000"
	APIURL         string `json:"apiUrl,omitempty"`         // ej. "http://10.250.0.5:5000/v2/"
	Healthy        bool   `json:"healthy"`                  // true si /v2/ responde HTTP 200
	StatusText     string `json:"statusText,omitempty"`     // Descripción legible del estado
	Error          string `json:"error,omitempty"`          // Error de conectividad si aplica
	UpdatedAt      string `json:"updatedAt,omitempty"`
}

// NodeRegistryConfigStatus representa el estado de configuración de daemon.json en una VM del laboratorio.
type NodeRegistryConfigStatus struct {
	VMID                  string   `json:"vmId"`
	VMName                string   `json:"vmName"`
	VMState               string   `json:"vmState"`
	PrivateIP             string   `json:"privateIp"`
	MirrorConfigured      bool     `json:"mirrorConfigured"`
	LocalInsecure         bool     `json:"localInsecure"`
	CacheInsecure         bool     `json:"cacheInsecure"`
	ConfiguredMirrors     []string `json:"configuredMirrors,omitempty"`
	InsecureRegistries    []string `json:"insecureRegistries,omitempty"`
	NeedsConfiguration    bool     `json:"needsConfiguration"`
	StatusLabel           string   `json:"statusLabel"`
	Error                 string   `json:"error,omitempty"`
}

// RegistryOverviewResponse agrupa el estado del Registry local, la caché Pull-through y todos los nodos.
type RegistryOverviewResponse struct {
	Local RegistryInstanceStatus     `json:"local"`
	Cache RegistryInstanceStatus     `json:"cache"`
	Nodes []NodeRegistryConfigStatus `json:"nodes"`
}

// RegistryDeployRequest solicita el despliegue idempotente del Registry local o de la Caché.
type RegistryDeployRequest struct {
	Target        string `json:"target"`                  // "local" o "cache"
	VMName        string `json:"vmName"`                  // VM destino
	HostIP        string `json:"hostIp,omitempty"`        // Opcional; si se omite se detecta de vessel-net
	Port          int    `json:"port,omitempty"`          // Por defecto 5000 (local) o 5001 (cache)
	ContainerName string `json:"containerName,omitempty"` // Por defecto vessel-registry o vessel-registry-cache
	VolumeName    string `json:"volumeName,omitempty"`    // Por defecto vessel-registry-data o vessel-registry-cache-data
	RemoteURL     string `json:"remoteUrl,omitempty"`     // Por defecto https://registry-1.docker.io en cache
}

// RegistryControlRequest solicita iniciar, detener, reiniciar o eliminar una instancia de Registry.
type RegistryControlRequest struct {
	Target string `json:"target"` // "local" o "cache"
	Action string `json:"action"` // "start", "stop", "restart", "remove"
}

// RegistryImageItem representa una imagen (repositorio + tag) almacenada en el Registry consultada por HTTP V2.
type RegistryImageItem struct {
	Repository    string   `json:"repository"`
	Tag           string   `json:"tag"`
	FullReference string   `json:"fullReference"` // ej. "10.250.0.5:5000/mysql:8.4"
	Digest        string   `json:"digest,omitempty"`
	MediaType     string   `json:"mediaType,omitempty"`
	SizeBytes     int64    `json:"sizeBytes"`
	Size          string   `json:"size"`
	LayersCount   int      `json:"layersCount,omitempty"`
	SharedTags    []string `json:"sharedTags,omitempty"` // Otros tags del mismo repo con igual digest
}

// RegistryImagesResponse devuelve la lista de imágenes almacenadas en una instancia de Registry.
type RegistryImagesResponse struct {
	Target       string              `json:"target"`
	EndpointHost string              `json:"endpointHost"`
	EndpointURL  string              `json:"endpointUrl"`
	Healthy      bool                `json:"healthy"`
	Repositories []string            `json:"repositories"`
	Images       []RegistryImageItem `json:"images"`
	Error        string              `json:"error,omitempty"`
}

// RegistryPushRequest solicita etiquetar una imagen local de una VM y subirla al Registry local escribible.
type RegistryPushRequest struct {
	VMName      string `json:"vmName"`
	SourceImage string `json:"sourceImage"` // ej. "mysql:8.4" o ID/repo:tag local
	Repository  string `json:"repository"`  // ej. "mysql" o "equipo/servicio"
	Tag         string `json:"tag"`         // ej. "8.4"
}

// RegistryPushResponse entrega el resultado del etiquetado y push al Registry local.
type RegistryPushResponse struct {
	Success         bool   `json:"success"`
	VMName          string `json:"vmName"`
	SourceImage     string `json:"sourceImage"`
	TargetReference string `json:"targetReference"` // ej. "10.250.0.5:5000/mysql:8.4"
	Repository      string `json:"repository"`
	Tag             string `json:"tag"`
	Output          string `json:"output"`
}

// RegistryPullRequest solicita descargar una imagen desde el Registry local hacia una VM destino.
type RegistryPullRequest struct {
	VMName     string `json:"vmName"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	ImageRef   string `json:"imageRef,omitempty"` // Alternativa directa "repo:tag"
	RetagLocal bool   `json:"retagLocal"`         // Si es true, añade tag corto <repo>:<tag> además de <ip>:<port>/<repo>:<tag>
}

// RegistryPullResponse entrega el resultado de descargar una imagen desde el Registry local.
type RegistryPullResponse struct {
	Success         bool   `json:"success"`
	VMName          string `json:"vmName"`
	PulledReference string `json:"pulledReference"`
	LocalTag        string `json:"localTag,omitempty"`
	Source          string `json:"source"`
	SourceLabel     string `json:"sourceLabel"`
	Output          string `json:"output"`
}

// SmartPullRequest solicita la descarga inteligente (Prioridad 1: Registry local -> Prioridad 2: Docker Hub / Caché).
type SmartPullRequest struct {
	VMName        string `json:"vmName"`
	Image         string `json:"image"`         // ej. "mysql:8.4", "nginx:alpine", "mi-app/api:1.0"
	AllowFallback bool   `json:"allowFallback"` // Si el usuario confirmó continuar con Docker Hub cuando el Registry local no responde
}

// SmartPullResponse entrega el resultado verificable de la descarga inteligente.
type SmartPullResponse struct {
	Success                  bool   `json:"success"`
	VMName                   string `json:"vmName"`
	RequestedImage           string `json:"requestedImage"`
	PulledReference          string `json:"pulledReference,omitempty"`
	LocalTag                 string `json:"localTag,omitempty"`
	FoundInLocalRegistry     bool   `json:"foundInLocalRegistry"`
	LocalRegistryChecked     bool   `json:"localRegistryChecked"`
	LocalRegistryUnreachable bool   `json:"localRegistryUnreachable,omitempty"`
	LocalRegistryError       string `json:"localRegistryError,omitempty"`
	CanFallbackToDockerHub   bool   `json:"canFallbackToDockerHub,omitempty"`
	UsedPullThroughCache     bool   `json:"usedPullThroughCache"`
	CacheHit                 bool   `json:"cacheHit"`
	Source                   string `json:"source"`      // "local-registry", "dockerhub-cache-hit", "dockerhub-cache-miss", "dockerhub-direct"
	SourceLabel              string `json:"sourceLabel"` // Descripción clara del origen utilizado
	Output                   string `json:"output,omitempty"`
}

// ConfigureNodesRegistryRequest solicita aplicar la configuración de mirror e insecure-registries en los nodos indicados.
type ConfigureNodesRegistryRequest struct {
	VMNames       []string `json:"vmNames"`                 // VMs a configurar (vacío = todas las VMs en ejecución)
	RestartDocker bool     `json:"restartDocker,omitempty"` // false = usar `systemctl reload docker` (SIGHUP sin detener contenedores), true = `systemctl restart docker`
	Confirmed     bool     `json:"confirmed"`               // Confirmación explícita del usuario
}

// NodeConfigureResult detalla el resultado de configurar daemon.json en un nodo específico.
type NodeConfigureResult struct {
	VMName  string `json:"vmName"`
	Success bool   `json:"success"`
	Method  string `json:"method,omitempty"` // "reload" o "restart"
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// ConfigureNodesRegistryResponse resume la aplicación de configuración en múltiples nodos.
type ConfigureNodesRegistryResponse struct {
	Success bool                  `json:"success"`
	Results []NodeConfigureResult `json:"results"`
	Message string                `json:"message"`
}
