package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config almacena los parametros de configuracion de la aplicacion y del cliente SSH
type Config struct {
	ServerAddr string `json:"serverAddr"`
	LogLevel   string `json:"logLevel"`

	SSHHost      string `json:"sshHost"`
	SSHPort      int    `json:"sshPort"`
	SSHUser      string `json:"sshUser"`
	SSHKeyPath   string `json:"sshKeyPath"`
	SSHShell     string `json:"sshShell"`     // "powershell", "cmd", o "default"
	SSHStartPath string `json:"sshStartPath"` // Ruta predeterminada de inicio

	// Configuracion centralizada para el Administrador de VMs y Plantillas
	TemplateRoot     string `json:"templateRoot"`     // ej: C:\VMManager\Templates
	VMRoot           string `json:"vmRoot"`           // ej: C:\VMManager\VMs
	SSHRoot          string `json:"sshRoot"`          // ej: C:\VMManager\SSH
	BootstrapKeyPath string `json:"bootstrapKeyPath"` // ej: C:\VMManager\SSH\bootstrap_id_ed25519
	BasePort         int    `json:"basePort"`         // ej: 22000

	// Configuracion de red compartida NAT Network para comunicacion entre VMs
	NATNetworkName string `json:"natNetworkName"` // ej: vessel-net
	NATNetworkCIDR string `json:"natNetworkCidr"` // ej: 10.250.0.0/24
}

// Load carga la configuracion desde archivos locales. El .env es la fuente inicial,
// sin depender de variables de entorno del proceso.
func Load() (*Config, error) {
	vars := make(map[string]string)

	// 1. Cargar config.properties (valores base/predeterminados)
	if propVars, err := parseKeyValueFile("config.properties"); err == nil {
		for k, v := range propVars {
			vars[k] = v
		}
	}

	// 2. Cargar .env (sobrescribe o complementa valores de properties)
	if envVars, err := parseKeyValueFile(".env"); err == nil {
		for k, v := range envVars {
			vars[k] = v
		}
	}

	// Helper para resolver claves entre los archivos locales y sus alternativas
	getVal := func(keys []string, fallback string) string {
		for _, k := range keys {
			if v, ok := vars[k]; ok && v != "" {
				return v
			}
		}
		return fallback
	}

	portStr := getVal([]string{"SSH_PORT", "ssh.port", "port"}, "22")
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		port = 22
	}

	basePortStr := getVal([]string{"VM_BASE_PORT", "vm.base_port", "base_port"}, "22000")
	basePort, err := strconv.Atoi(basePortStr)
	if err != nil || basePort <= 0 {
		basePort = 22000
	}

	bootstrapKey := normalizePath(getVal([]string{"VM_BOOTSTRAP_KEY_PATH", "vm.bootstrap_key_path", "bootstrap_key_path"}, ""))
	if bootstrapKey == "" {
		if home, err := os.UserHomeDir(); err == nil {
			candidate := filepath.Join(home, ".ssh", "vm-bootstrap")
			if _, err := os.Stat(candidate); err == nil {
				bootstrapKey = candidate
			}
		}
		if bootstrapKey == "" {
			bootstrapKey = `C:\VMManager\SSH\bootstrap_id_ed25519`
		}
	}

	cfg := &Config{
		ServerAddr:       getVal([]string{"SERVER_ADDR", "server.addr"}, ":8080"),
		LogLevel:         getVal([]string{"LOG_LEVEL", "log.level"}, "info"),
		SSHHost:          getVal([]string{"SSH_HOST", "ssh.host", "host"}, ""),
		SSHPort:          port,
		SSHUser:          getVal([]string{"SSH_USER", "ssh.user", "user"}, "HP 245 RYZEN 3"),
		SSHKeyPath:       normalizePath(getVal([]string{"SSH_KEY_PATH", "ssh.key_path", "key_path"}, "")),
		SSHShell:         strings.ToLower(getVal([]string{"SSH_SHELL", "ssh.shell", "shell"}, "powershell")),
		SSHStartPath:     getVal([]string{"SSH_START_PATH", "ssh.start_path", "start_path", "start.path"}, "C:/Users/andres.HP245G7"),
		TemplateRoot:     getVal([]string{"VM_TEMPLATE_ROOT", "vm.template_root", "template_root"}, `C:\VMManager\Templates`),
		VMRoot:           getVal([]string{"VM_VM_ROOT", "vm.vm_root", "vm_root"}, `C:\VMManager\VMs`),
		SSHRoot:          getVal([]string{"VM_SSH_ROOT", "vm.ssh_root", "ssh_root"}, `C:\VMManager\SSH`),
		BootstrapKeyPath: bootstrapKey,
		BasePort:         basePort,
		NATNetworkName:   getVal([]string{"VM_NAT_NETWORK_NAME", "vm.nat_network_name", "nat_network_name"}, "vessel-net"),
		NATNetworkCIDR:   getVal([]string{"VM_NAT_NETWORK_CIDR", "vm.nat_network_cidr", "nat_network_cidr"}, "10.250.0.0/24"),
	}

	return cfg, nil
}

// SaveProperties persiste la configuracion en un archivo con formato properties
func SaveProperties(path string, cfg *Config) error {
	var b strings.Builder
	b.WriteString("# ==============================================================================\n")
	b.WriteString("# Archivo de Propiedades de Configuración - SSH Web Terminal\n")
	b.WriteString("# ==============================================================================\n\n")

	fmt.Fprintf(&b, "ssh.host=%s\n", cfg.SSHHost)
	fmt.Fprintf(&b, "ssh.port=%d\n", cfg.SSHPort)
	fmt.Fprintf(&b, "ssh.user=%s\n", cfg.SSHUser)
	fmt.Fprintf(&b, "ssh.shell=%s\n", cfg.SSHShell)
	fmt.Fprintf(&b, "ssh.start_path=%s\n", cfg.SSHStartPath)
	fmt.Fprintf(&b, "ssh.key_path=%s\n", cfg.SSHKeyPath)
	fmt.Fprintf(&b, "server.addr=%s\n", cfg.ServerAddr)
	fmt.Fprintf(&b, "log.level=%s\n\n", cfg.LogLevel)

	fmt.Fprintf(&b, "# Rutas centralizadas de virtualizacion\n")
	fmt.Fprintf(&b, "vm.template_root=%s\n", cfg.TemplateRoot)
	fmt.Fprintf(&b, "vm.vm_root=%s\n", cfg.VMRoot)
	fmt.Fprintf(&b, "vm.ssh_root=%s\n", cfg.SSHRoot)
	fmt.Fprintf(&b, "vm.bootstrap_key_path=%s\n", cfg.BootstrapKeyPath)
	fmt.Fprintf(&b, "vm.base_port=%d\n", cfg.BasePort)
	natNetName := cfg.NATNetworkName
	if natNetName == "" {
		natNetName = "vessel-net"
	}
	natNetCIDR := cfg.NATNetworkCIDR
	if natNetCIDR == "" {
		natNetCIDR = "10.250.0.0/24"
	}
	fmt.Fprintf(&b, "vm.nat_network_name=%s\n", natNetName)
	fmt.Fprintf(&b, "vm.nat_network_cidr=%s\n", natNetCIDR)

	return os.WriteFile(path, []byte(b.String()), 0644)
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}

	path = strings.Trim(strings.TrimSpace(path), `"'`)
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/\\"))
		}
	}

	return filepath.Clean(filepath.FromSlash(strings.ReplaceAll(path, `\`, `/`)))
}

func parseKeyValueFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			key, value, ok = strings.Cut(line, ":")
			if !ok {
				continue
			}
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		vars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error al procesar archivo %s: %w", path, err)
	}

	return vars, nil
}
