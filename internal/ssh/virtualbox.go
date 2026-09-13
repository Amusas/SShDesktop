package ssh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	gossh "golang.org/x/crypto/ssh"
)

type VMInfo struct {
	Name      string `json:"name"`
	OS        string `json:"os"`
	Status    string `json:"state"`
	CPU       int    `json:"cores"`
	MemoryMB  int    `json:"memoryMB"`
	DiskGB    int    `json:"diskGB"`
	Network   string `json:"network"`
	Firmware  string `json:"firmware,omitempty"`
	DiskPath  string `json:"diskPath,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

type VMCreateRequest struct {
	Name     string `json:"name"`
	OS       string `json:"os"`
	ISO      string `json:"iso"`
	Cores    int    `json:"cores"`
	MemoryMB int    `json:"memoryMB"`
	DiskGB   int    `json:"diskGB"`
}

type VMCreateResponse struct {
	VM VMInfo `json:"vm"`
}

func ListVMs(cfg Config, log *slog.Logger) ([]VMInfo, error) {
	out, err := runCommand(cfg, `VBoxManage list vms`, log)
	if err != nil {
		return nil, err
	}
	vms := []VMInfo{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := regexp.MustCompile(`^"([^"]+)"\s*\{`).FindStringSubmatch(line)
		if len(m) == 2 {
			name := m[1]
			vms = append(vms, VMInfo{Name: name})
		}
	}
	return vms, nil
}

func ShowVMInfo(cfg Config, name string, log *slog.Logger) (VMInfo, error) {
	cmd := fmt.Sprintf(`VBoxManage showvminfo "%s" --machinereadable`, escapeForDoubleQuote(name))
	out, err := runCommand(cfg, cmd, log)
	if err != nil {
		return VMInfo{}, err
	}

	info := VMInfo{Name: name}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "VMState=") {
			info.Status = strings.Trim(strings.TrimPrefix(line, "VMState="), `"`)
		}
		if strings.HasPrefix(line, "name=") {
			info.Name = strings.Trim(strings.TrimPrefix(line, "name="), `"`)
		}
		if strings.HasPrefix(line, "ostype=") {
			info.OS = strings.Trim(strings.TrimPrefix(line, "ostype="), `"`)
		}
		if strings.HasPrefix(line, "cpus=") {
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "cpus="))); err == nil {
				info.CPU = n
			}
		}
		if strings.HasPrefix(line, "memory=") {
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "memory="))); err == nil {
				info.MemoryMB = n
			}
		}
		if strings.HasPrefix(line, "nic1=") || strings.HasPrefix(line, "nic2=") || strings.HasPrefix(line, "nic3=") || strings.HasPrefix(line, "nic4=") {
			v := strings.Trim(strings.TrimPrefix(line, strings.Split(line, "=")[0]+"="), `"`)
			if strings.TrimSpace(v) != "" {
				info.Network = v
			}
		}
	}
	return info, nil
}

func DeleteVM(cfg Config, name string, log *slog.Logger) error {
	cmd := fmt.Sprintf(`VBoxManage unregistervm "%s" --delete`, escapeForDoubleQuote(name))
	_, err := runCommand(cfg, cmd, log)
	return err
}

func CreateVM(cfg Config, req VMCreateRequest, log *slog.Logger) (VMInfo, error) {
	name := sanitizeVMName(req.Name)
	if name == "" || !regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`).MatchString(name) {
		return VMInfo{}, fmt.Errorf("nombre de VM invalido")
	}
	if req.Cores < 1 || req.Cores > 8 {
		return VMInfo{}, fmt.Errorf("CPU debe estar entre 1 y 8")
	}
	if req.MemoryMB < 256 || req.MemoryMB > 65536 {
		return VMInfo{}, fmt.Errorf("RAM debe estar entre 256 MB y 65536 MB")
	}
	if req.DiskGB < 10 || req.DiskGB > 2048 {
		return VMInfo{}, fmt.Errorf("disco debe estar entre 10 GB y 2048 GB")
	}
	if strings.TrimSpace(req.ISO) == "" {
		return VMInfo{}, fmt.Errorf("la ISO es obligatoria")
	}

	osInfo, err := loadOSDetailsFromISO(req.ISO)
	if err != nil {
		return VMInfo{}, err
	}

	// Determinar directorio base de almacenamiento según el tipo de shell del host remoto
	storageDir := `/vms`
	if strings.EqualFold(strings.ToLower(cfg.Shell), "powershell") || strings.EqualFold(strings.ToLower(cfg.Shell), "cmd") {
		storageDir = `C:\VMs`
	}

	// Validar que la VM no exista y crear recursos en la ruta remota
	if out, err := runCommand(cfg, `VBoxManage list vms`, log); err == nil {
		if strings.Contains(out, `"`+name+`"`) {
			return VMInfo{}, fmt.Errorf("la maquina virtual ya existe")
		}
	}

	// Crear directorios remotos de ISO y máquinas.
	baseDir := strings.ReplaceAll(storageDir, `\`, `/`)
	isoDir := baseDir + `/isos`
	machineDir := baseDir + `/machines/` + name
	if strings.Contains(storageDir, `C:\`) {
		isoDir = `C:\VMs\isos`
		machineDir = `C:\VMs\machines\` + name
	}

	cmds := []string{}
	if strings.Contains(storageDir, `C:\`) {
		cmds = append(cmds, fmt.Sprintf(`powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '%s','%s' | Out-Null"`, isoDir, machineDir))
	} else {
		cmds = append(cmds, fmt.Sprintf(`mkdir -p "%s" "%s"`, isoDir, machineDir))
	}

	// Si no existe la ISO, usar una descarga remota simple usando curl/wget o PowerShell.
	isoPath := isoDir + `/` + osInfo.ISO
	if strings.Contains(storageDir, `C:\`) {
		isoPath = strings.ReplaceAll(isoPath, `/`, `\`)
	}
	cmds = append(cmds, buildISOEnsureCommand(cfg, osInfo, isoPath))

	cmds = append(cmds,
		fmt.Sprintf(`VBoxManage createvm --name "%s" --ostype "%s" --register`, escapeForDoubleQuote(name), osInfo.VBoxOSType),
		fmt.Sprintf(`VBoxManage modifyvm "%s" --memory %d --cpus %d --nic1 nat`, escapeForDoubleQuote(name), req.MemoryMB, req.Cores),
		fmt.Sprintf(`VBoxManage createmedium disk --filename "%s/%s.vdi" --size %d --format VDI --variant Standard`, escapeForDoubleQuote(machineDir), escapeForDoubleQuote(name), req.DiskGB*1024),
		fmt.Sprintf(`VBoxManage storagectl "%s" --name "SATA" --add sata --controller IntelAhci`, escapeForDoubleQuote(name)),
		fmt.Sprintf(`VBoxManage storageattach "%s" --storagectl "SATA" --port 0 --device 0 --type hdd --medium "%s/%s.vdi"`, escapeForDoubleQuote(name), escapeForDoubleQuote(machineDir), escapeForDoubleQuote(name)),
		fmt.Sprintf(`VBoxManage storagectl "%s" --name "IDE" --add ide`, escapeForDoubleQuote(name)),
		fmt.Sprintf(`VBoxManage storageattach "%s" --storagectl "IDE" --port 0 --device 0 --type dvddrive --medium "%s"`, escapeForDoubleQuote(name), escapeForDoubleQuote(isoPath)),
	)

	for _, command := range cmds {
		if _, err := runCommand(cfg, command, log); err != nil {
			return VMInfo{}, err
		}
	}

	return ShowVMInfo(cfg, name, log)
}

type OSDetails struct {
	VBoxOSType string
	ISO        string
	URL        string
}

func loadOSDetailsFromISO(iso string) (OSDetails, error) {
	iso = strings.ToLower(strings.TrimSpace(iso))
	candidates := []string{
		filepath.Join(".", "isos.json"),
		filepath.Join("..", "isos.json"),
		filepath.Join("..", "..", "isos.json"),
	}
	var lastErr error
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			lastErr = err
			continue
		}
		var payload struct {
			ISOs []struct {
				Name       string `json:"name"`
				URL        string `json:"url"`
				OS         string `json:"os"`
				VBoxOSType string `json:"vboxOsType"`
			} `json:"isos"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return OSDetails{}, fmt.Errorf("parseando isos.json: %w", err)
		}
		for _, item := range payload.ISOs {
			if strings.EqualFold(strings.ToLower(item.Name), iso) {
				return OSDetails{VBoxOSType: item.VBoxOSType, ISO: item.Name, URL: item.URL}, nil
			}
		}
	}
	return OSDetails{}, fmt.Errorf("ISO no soportada o no encontrada en el catalogo: %w", lastErr)
}

func buildISOEnsureCommand(cfg Config, osInfo OSDetails, isoPath string) string {
	if strings.Contains(strings.ToLower(cfg.Shell), "powershell") || strings.Contains(strings.ToLower(cfg.Shell), "cmd") {
		return fmt.Sprintf("powershell -NoProfile -Command \"if (!(Test-Path '%s')) { curl.exe -L '%s' -o '%s' }\"", isoPath, osInfo.URL, isoPath)
	}
	return fmt.Sprintf("if [ ! -f %s ]; then curl -L -o %s %s || wget -O %s %s; fi", quoteFile(isoPath), quoteFile(isoPath), quoteURL(osInfo.URL), quoteFile(isoPath), quoteURL(osInfo.URL))
}

func runCommand(cfg Config, command string, log *slog.Logger) (string, error) {
	authMethods, err := buildAuthMethods(cfg)
	if err != nil {
		return "", fmt.Errorf("construyendo metodos de autenticacion: %w", err)
	}
	clientConfig := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := gossh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return "", fmt.Errorf("conectando al servidor SSH: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("creando sesion SSH: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(command); err != nil {
		if stderr.Len() > 0 {
			return stdout.String(), fmt.Errorf("ejecutando %s: %w: %s", command, err, strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), fmt.Errorf("ejecutando %s: %w", command, err)
	}
	return stdout.String(), nil
}

func sanitizeVMName(name string) string {
	r := regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)
	return strings.TrimSpace(r.ReplaceAllString(name, "-"))
}

func escapeForDoubleQuote(value string) string {
	return strings.ReplaceAll(value, `"`, `"`)
}

func quoteFile(path string) string {
	return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
}

func quoteURL(url string) string {
	return `"` + strings.ReplaceAll(url, `"`, `\"`) + `"`
}

func parseVMCreateRequest(raw interface{}) VMCreateRequest {
	// JSON parse happens in handler. This helper keeps compatibility with existing JSON payloads.
	buf, _ := json.Marshal(raw)
	var req VMCreateRequest
	_ = json.Unmarshal(buf, &req)
	return req
}

func init() {
	_ = os.Stdout
}
