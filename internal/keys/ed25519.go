package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// VMSSHIdentity almacena los metadatos de identidad SSH generados para una maquina virtual
type VMSSHIdentity struct {
	VMName         string `json:"vmName"`
	Username       string `json:"username"`
	PublicKey      string `json:"publicKey"`
	PrivateKeyPEM  string `json:"privateKeyPEM,omitempty"`
	PrivateKeyPath string `json:"privateKeyPath"`
	PublicKeyPath  string `json:"publicKeyPath"`
	Fingerprint    string `json:"fingerprint"`
}

// GenerateVMUser deriva un nombre de usuario Linux unico y valido para la VM (ej. vm_otramv_a82f)
func GenerateVMUser(vmName string) string {
	cleanName := strings.ToLower(strings.TrimSpace(vmName))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	cleanName = re.ReplaceAllString(cleanName, "")
	if cleanName == "" {
		cleanName = "vm"
	}
	if len(cleanName) > 16 {
		cleanName = cleanName[:16]
	}

	// Agregar sufijo aleatorio corto de 4 caracteres para garantizar unicidad
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", vmName, time.Now().UnixNano())))
	shortID := fmt.Sprintf("%x", hash[:2])

	return fmt.Sprintf("vm_%s_%s", cleanName, shortID)
}

// GenerateEd25519KeyForVM genera un par de claves Ed25519 de forma nativa en Go y las guarda en sshRoot/vms/<vmName>/
func GenerateEd25519KeyForVM(sshRoot, vmName string) (VMSSHIdentity, error) {
	if sshRoot == "" {
		sshRoot = `C:\VMManager\SSH`
	}

	vmDir := filepath.Join(sshRoot, "vms", sanitizeFileName(vmName))
	if err := os.MkdirAll(vmDir, 0700); err != nil {
		return VMSSHIdentity{}, fmt.Errorf("creando directorio de llaves para VM '%s': %w", vmName, err)
	}

	// 1. Generar par de claves Ed25519 nativo
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return VMSSHIdentity{}, fmt.Errorf("generando clave Ed25519: %w", err)
	}

	// 2. Codificar clave privada en formato OpenSSH / PKCS8 PEM
	privBlock, err := gossh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return VMSSHIdentity{}, fmt.Errorf("codificando clave privada Ed25519 a OpenSSH: %w", err)
	}
	privPEM := pem.EncodeToMemory(privBlock)

	// 3. Generar clave publica en formato OpenSSH
	sshPubKey, err := gossh.NewPublicKey(pubKey)
	if err != nil {
		return VMSSHIdentity{}, fmt.Errorf("convirtiendo a clave publica OpenSSH: %w", err)
	}

	username := GenerateVMUser(vmName)
	pubKeyStr := strings.TrimSpace(string(gossh.MarshalAuthorizedKey(sshPubKey))) + fmt.Sprintf(" %s@sshdesktop", username)

	// 4. Calcular fingerprint SHA256
	shaSum := sha256.Sum256(sshPubKey.Marshal())
	fingerprint := "SHA256:" + strings.TrimRight(base64.StdEncoding.EncodeToString(shaSum[:]), "=")

	// 5. Guardar archivos en el host local/backend
	privPath := filepath.Join(vmDir, "id_ed25519")
	pubPath := filepath.Join(vmDir, "id_ed25519.pub")

	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return VMSSHIdentity{}, fmt.Errorf("guardando clave privada: %w", err)
	}
	if err := os.WriteFile(pubPath, []byte(pubKeyStr+"\n"), 0644); err != nil {
		return VMSSHIdentity{}, fmt.Errorf("guardando clave publica: %w", err)
	}

	return VMSSHIdentity{
		VMName:         vmName,
		Username:       username,
		PublicKey:      pubKeyStr,
		PrivateKeyPEM:  string(privPEM),
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		Fingerprint:    fingerprint,
	}, nil
}

// DeleteVMKeys elimina el directorio de llaves de una VM especifica
func DeleteVMKeys(sshRoot, vmName string) error {
	if sshRoot == "" {
		sshRoot = `C:\VMManager\SSH`
	}
	vmDir := filepath.Join(sshRoot, "vms", sanitizeFileName(vmName))
	return os.RemoveAll(vmDir)
}

func sanitizeFileName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	clean := re.ReplaceAllString(strings.TrimSpace(name), "-")
	if clean == "" {
		clean = "vm"
	}
	return clean
}
