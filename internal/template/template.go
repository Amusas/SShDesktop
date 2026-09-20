package template

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/ssh"
)

// Template define los metadatos de una plantilla de maquina virtual
type Template struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OSType    string `json:"osType"`
	Disk      string `json:"disk"`     // ej. "base.vdi"
	SSHUser   string `json:"sshUser"`  // ej. "bootstrap"
	SSHPort   int    `json:"sshPort"`  // ej. 22
	DiskPath  string `json:"diskPath"` // Ruta completa resuelta en el host remoto
}

// Service gestiona el descubrimiento y validacion de plantillas
type Service struct {
	cfg *config.Config
	log *slog.Logger
}

// NewService crea un nuevo servicio de plantillas
func NewService(cfg *config.Config, log *slog.Logger) *Service {
	return &Service{
		cfg: cfg,
		log: log,
	}
}

var validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// DiscoverTemplates escanea el directorio TemplateRoot en el host remoto vía SSH y retorna las plantillas validas
func (s *Service) DiscoverTemplates(ctx context.Context) ([]Template, error) {
	templateRoot := s.cfg.TemplateRoot
	if templateRoot == "" {
		templateRoot = `C:\VMManager\Templates`
	}

	// Script PowerShell para descubrir y leer los template.json de forma segura
	psScript := fmt.Sprintf(`$root = '%s'
if (!(Test-Path -LiteralPath $root)) {
    Write-Output '[]'
    exit 0
}
$dirs = Get-ChildItem -LiteralPath $root -Directory -ErrorAction SilentlyContinue
$list = @()
foreach ($d in $dirs) {
    $jsonPath = Join-Path $d.FullName 'template.json'
    if (Test-Path -LiteralPath $jsonPath) {
        try {
            $content = Get-Content -LiteralPath $jsonPath -Raw -Encoding UTF8
            $obj = $content | ConvertFrom-Json
            if ($obj.id -and $obj.name) {
                $osType = if ($obj.osType) { [string]$obj.osType } else { 'Debian_64' }
                $diskName = if ($obj.disk) { [string]$obj.disk } else { '' }
                $diskFile = if ($diskName) { Join-Path $d.FullName $diskName } else { '' }
                if (-not ($diskFile -and (Test-Path -LiteralPath $diskFile))) {
                    $vdis = Get-ChildItem -LiteralPath $d.FullName -Filter *.vdi -File -ErrorAction SilentlyContinue
                    if ($vdis -and $vdis.Count -gt 0) {
                        $diskFile = $vdis[0].FullName
                        $diskName = $vdis[0].Name
                    }
                }
                if ($diskFile -and (Test-Path -LiteralPath $diskFile)) {
                    $item = [PSCustomObject]@{
                        id       = [string]$obj.id
                        name     = [string]$obj.name
                        osType   = $osType
                        disk     = $diskName
                        sshUser  = if ($obj.sshUser) { [string]$obj.sshUser } else { 'bootstrap' }
                        sshPort  = if ($obj.sshPort) { [int]$obj.sshPort } else { 22 }
                        diskPath = $diskFile
                    }
                    $list += $item
                }
            }
        } catch {}
    }
}
if ($list.Count -eq 0) {
    Write-Output '[]'
} else {
    $list | ConvertTo-Json -Compress
}
`, strings.ReplaceAll(templateRoot, "'", "''"))

	sshCfg := ssh.Config{
		Host:      s.cfg.SSHHost,
		Port:      s.cfg.SSHPort,
		User:      s.cfg.SSHUser,
		KeyPath:   s.cfg.SSHKeyPath,
		Shell:     s.cfg.SSHShell,
		StartPath: s.cfg.SSHStartPath,
	}

	encodedCmd := encodePowerShellCommand(psScript)
	out, err := ssh.ExecuteCommandContext(ctx, sshCfg, fmt.Sprintf("powershell -NoProfile -EncodedCommand %s", encodedCmd), s.log)
	if err != nil {
		s.log.Warn("no se pudieron descubrir plantillas remotamente", "error", err)
		return nil, fmt.Errorf("descubriendo plantillas en host remoto: %w", err)
	}

	out = strings.TrimSpace(out)
	if out == "" || out == "[]" {
		return []Template{}, nil
	}

	// Manejar tanto array como objeto individual si PowerShell devuelve solo 1 elemento
	if strings.HasPrefix(out, "{") {
		var single Template
		if err := json.Unmarshal([]byte(out), &single); err != nil {
			return nil, fmt.Errorf("parseando plantilla remota: %w", err)
		}
		if err := ValidateTemplate(single); err != nil {
			return []Template{}, nil
		}
		return []Template{single}, nil
	}

	var rawList []Template
	if err := json.Unmarshal([]byte(out), &rawList); err != nil {
		return nil, fmt.Errorf("parseando lista de plantillas: %w (salida: %s)", err, out)
	}

	validList := make([]Template, 0, len(rawList))
	for _, t := range rawList {
		if err := ValidateTemplate(t); err != nil {
			s.log.Warn("plantilla invalida ignorada", "id", t.ID, "error", err)
			continue
		}
		validList = append(validList, t)
	}

	return validList, nil
}

// GetTemplateByID busca una plantilla especifica por su ID, validando que no haya path traversal
func (s *Service) GetTemplateByID(ctx context.Context, id string) (Template, error) {
	id = strings.TrimSpace(id)
	if !validIDRegex.MatchString(id) {
		return Template{}, fmt.Errorf("id de plantilla invalido o con caracteres peligrosos: %s", id)
	}

	templates, err := s.DiscoverTemplates(ctx)
	if err != nil {
		return Template{}, err
	}

	for _, t := range templates {
		if strings.EqualFold(t.ID, id) {
			return t, nil
		}
	}

	return Template{}, fmt.Errorf("plantilla no encontrada: %s", id)
}

// ValidateTemplate verifica que los campos obligatorios esten presentes y sean seguros
func ValidateTemplate(t Template) error {
	if !validIDRegex.MatchString(t.ID) {
		return fmt.Errorf("id de plantilla invalido: %s", t.ID)
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("nombre de plantilla vacio")
	}
	if strings.TrimSpace(t.OSType) == "" {
		return fmt.Errorf("osType de plantilla vacio")
	}
	if strings.TrimSpace(t.Disk) == "" {
		return fmt.Errorf("nombre de archivo de disco vacio")
	}
	// Prevenir path traversal en el campo disk
	cleanDisk := filepath.Clean(t.Disk)
	if strings.Contains(cleanDisk, "..") || strings.Contains(cleanDisk, "/") || strings.Contains(cleanDisk, `\`) {
		return fmt.Errorf("archivo de disco no puede contener rutas o separadores: %s", t.Disk)
	}
	return nil
}

func encodePowerShellCommand(script string) string {
	runes := utf16.Encode([]rune(script))
	bytes := make([]byte, len(runes)*2)
	for i, r := range runes {
		bytes[i*2] = byte(r)
		bytes[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}
