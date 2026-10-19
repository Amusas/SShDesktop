package docker

// Container representa la información resumida de un contenedor Docker en una VM
type Container struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Command   string `json:"command,omitempty"`
	CreatedAt string `json:"createdAt"`
	State     string `json:"state"`  // running, exited, created, paused, restarting
	Status    string `json:"status"` // ej. "Up 2 hours", "Exited (0) 10 minutes ago"
	Ports     string `json:"ports"`
	VMName    string `json:"vmName"` // Nombre de la VM anfitriona
}

// PortMapping define un mapeo de puertos entre la VM y el contenedor
type PortMapping struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"` // "tcp" o "udp"
}

// VolumeMount define un montaje de volumen o ruta
type VolumeMount struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
}

// EnvVar representa un par clave-valor para variables de entorno
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CreateContainerRequest contiene los parámetros para crear y ejecutar un contenedor
type CreateContainerRequest struct {
	VMName        string        `json:"vmName"`
	Name          string        `json:"name"`
	Image         string        `json:"image"`
	Ports         []PortMapping `json:"ports,omitempty"`
	Volumes       []VolumeMount `json:"volumes,omitempty"`
	Env           []EnvVar      `json:"env,omitempty"`
	RestartPolicy string        `json:"restartPolicy,omitempty"` // unless-stopped, always, on-failure, no
	Network       string        `json:"network,omitempty"`  // Red Docker a la que conectar el contenedor
	Networks      []string      `json:"networks,omitempty"` // Múltiples redes Docker adicionales o lista de redes
	Command       string        `json:"command,omitempty"`
}

// ContainerNetworkInfo describe la conexión de un contenedor a una red específica
type ContainerNetworkInfo struct {
	Name       string `json:"name"`
	IPAddress  string `json:"ipAddress,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	MacAddress string `json:"macAddress,omitempty"`
	NetworkID  string `json:"networkID,omitempty"`
}

// ContainerInspect contiene detalles completos de inspección de un contenedor
type ContainerInspect struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	VMName        string                 `json:"vmName"`
	Created       string                 `json:"created"`
	Image         string                 `json:"image"`
	State         InspectState           `json:"state"`
	RestartPolicy string                 `json:"restartPolicy"`
	IPAddress     string                 `json:"ipAddress,omitempty"`
	Networks      []ContainerNetworkInfo `json:"networks,omitempty"`
	Ports         []PortMapping          `json:"ports"`
	Mounts        []VolumeMount          `json:"mounts"`
	Env           []string               `json:"env"`
	Labels        map[string]string      `json:"labels,omitempty"`
}

// InspectState encapsula el estado detallado de ejecución
type InspectState struct {
	Status     string `json:"status"`
	Running    bool   `json:"running"`
	Paused     bool   `json:"paused"`
	Restarting bool   `json:"restarting"`
	ExitCode   int    `json:"exitCode"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	Error      string `json:"error,omitempty"`
}

// DockerStatus reporta la disponibilidad y estadísticas del motor Docker en la VM
type DockerStatus struct {
	Available         bool   `json:"available"`
	Version           string `json:"version,omitempty"`
	ContainersRunning int    `json:"containersRunning"`
	ContainersTotal   int    `json:"containersTotal"`
	ImagesTotal       int    `json:"imagesTotal"`
	Error             string `json:"error,omitempty"`
}

// CatalogItem representa una plantilla predefinida de contenedor
type CatalogItem struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Category           string        `json:"category"`
	Description        string        `json:"description"`
	DefaultImage       string        `json:"defaultImage"`
	RecommendedPorts   []PortMapping `json:"recommendedPorts"`
	RecommendedEnv     []EnvVar      `json:"recommendedEnv"`
	RecommendedVolumes []VolumeMount `json:"recommendedVolumes"`
	DefaultRestart     string        `json:"defaultRestart"`
}

// CategoryCatalog agrupa aplicaciones por categoría para el asistente visual
type CategoryCatalog struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Items       []CatalogItem `json:"items"`
}

// Image representa una imagen Docker en una VM
type Image struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	CreatedAt  string `json:"createdAt"`
	VMName     string `json:"vmName"`
}

// BuildFile representa un archivo adjunto que formará parte del contexto de construcción del Dockerfile
type BuildFile struct {
	Name    string `json:"name"`
	Content string `json:"content"` // Base64 encoded
}

// BuildImageRequest contiene los datos para construir una imagen desde un Dockerfile
type BuildImageRequest struct {
	VMName     string      `json:"vmName"`
	ImageName  string      `json:"imageName"`
	Tag        string      `json:"tag,omitempty"`
	Dockerfile string      `json:"dockerfile"`
	Files      []BuildFile `json:"files,omitempty"`
}

// BuildImageResponse entrega el resultado de la construcción de la imagen
type BuildImageResponse struct {
	Success bool   `json:"success"`
	Image   string `json:"image"`
	Output  string `json:"output"`
}

// CommitContainerRequest contiene los datos para crear una imagen Docker a partir de un contenedor existente
type CommitContainerRequest struct {
	VMName      string `json:"vmName"`
	ContainerID string `json:"containerId"`
	ImageName   string `json:"imageName"`
	Tag         string `json:"tag,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Author      string `json:"author,omitempty"`
}

// CommitContainerResponse entrega el resultado de la creación de la imagen desde un contenedor
type CommitContainerResponse struct {
	Success bool   `json:"success"`
	Image   string `json:"image"`
	ImageID string `json:"imageId,omitempty"`
	Output  string `json:"output,omitempty"`
}

// Volume representa un volumen Docker en una VM
type Volume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	CreatedAt  string `json:"createdAt,omitempty"`
	Status     string `json:"status"` // "En uso", "Sin utilizar", etc.
	VMName     string `json:"vmName"`
}

// CreateVolumeRequest contiene los parámetros para crear un volumen
type CreateVolumeRequest struct {
	VMName string `json:"vmName"`
	Name   string `json:"name"`
	Driver string `json:"driver,omitempty"` // local por defecto
}

// Network representa una red Docker en una VM
type Network struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Driver         string            `json:"driver"`
	Scope          string            `json:"scope"`
	Subnet         string            `json:"subnet,omitempty"`
	Gateway        string            `json:"gateway,omitempty"`
	IPRange        string            `json:"ipRange,omitempty"`
	Internal       bool              `json:"internal"`
	Attachable     bool              `json:"attachable"`
	Containers     int               `json:"containers"`
	ContainerList  []string          `json:"containerList,omitempty"` // Nombres de contenedores conectados
	VMName         string            `json:"vmName"`
	Options        map[string]string `json:"options,omitempty"`
}

// CreateNetworkRequest contiene los parámetros para crear una red Docker
type CreateNetworkRequest struct {
	VMName     string `json:"vmName"`
	Name       string `json:"name"`
	Driver     string `json:"driver,omitempty"` // bridge por defecto
	Subnet     string `json:"subnet,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	IPRange    string `json:"ipRange,omitempty"`
	Internal   bool   `json:"internal,omitempty"`
	Attachable bool   `json:"attachable,omitempty"`
}
