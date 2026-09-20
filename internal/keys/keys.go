package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

type SSHKey struct {
	ID          string `json:"id"`
	VMName      string `json:"vmName"`
	Name        string `json:"name"`
	Algorithm   string `json:"algorithm"`
	PublicKey   string `json:"publicKey"`
	PrivateKey  string `json:"privateKey"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"createdAt"`
}

type Store struct {
	mu   sync.RWMutex
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

// GenerateKeyPairForVM crea un par de llaves criptográficas RSA 2048 y las vincula a la VM
func (s *Store) GenerateKeyPairForVM(vmName string) (SSHKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyID := fmt.Sprintf("key-%s-%d", sanitizeName(vmName), time.Now().Unix())
	keyName := fmt.Sprintf("id_rsa_%s", sanitizeName(vmName))

	// 1. Generar clave privada RSA de 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return SSHKey{}, fmt.Errorf("generando clave RSA: %w", err)
	}

	// 2. Codificar clave privada en formato PEM (PKCS#1)
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	}
	privateKeyPEM := pem.EncodeToMemory(privateKeyBlock)

	// 3. Generar clave pública en formato OpenSSH
	sshPublicKey, err := gossh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return SSHKey{}, fmt.Errorf("convirtiendo a clave publica OpenSSH: %w", err)
	}
	publicKeyOpenSSH := string(gossh.MarshalAuthorizedKey(sshPublicKey))
	publicKeyOpenSSH = strings.TrimSpace(publicKeyOpenSSH) + fmt.Sprintf(" %s@ssh-web", sanitizeName(vmName))

	// 4. Calcular huella digital (SHA256 fingerprint)
	sha256Sum := sha256.Sum256(sshPublicKey.Marshal())
	fingerprint := "SHA256:" + strings.TrimRight(base64.StdEncoding.EncodeToString(sha256Sum[:]), "=")

	key := SSHKey{
		ID:          keyID,
		VMName:      strings.TrimSpace(vmName),
		Name:        keyName,
		Algorithm:   "RSA 2048",
		PublicKey:   publicKeyOpenSSH,
		PrivateKey:  string(privateKeyPEM),
		Fingerprint: fingerprint,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	// Guardar en el archivo de llaves
	keys, _ := s.loadUnlocked()
	keys = append(keys, key)
	if err := s.saveUnlocked(keys); err != nil {
		return SSHKey{}, fmt.Errorf("guardando llave en store: %w", err)
	}

	return key, nil
}

// Add guarda una llave en el store asegurando persistencia en disco
func (s *Store) Add(key SSHKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys, _ := s.loadUnlocked()
	// Si ya existia con el mismo ID o VMName, actualizar
	updated := false
	for i, k := range keys {
		if k.ID == key.ID || (k.VMName != "" && k.VMName == key.VMName) {
			keys[i] = key
			updated = true
			break
		}
	}
	if !updated {
		keys = append(keys, key)
	}
	return s.saveUnlocked(keys)
}

func (s *Store) List() ([]SSHKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnlocked()
}

func (s *Store) Get(id string) (SSHKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys, err := s.loadUnlocked()
	if err != nil {
		return SSHKey{}, err
	}
	for _, k := range keys {
		if k.ID == id || k.VMName == id || k.Name == id {
			return k, nil
		}
	}
	return SSHKey{}, fmt.Errorf("llave no encontrada: %s", id)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys, err := s.loadUnlocked()
	if err != nil {
		return err
	}

	filtered := make([]SSHKey, 0, len(keys))
	found := false
	for _, k := range keys {
		if k.ID == id || strings.EqualFold(k.VMName, id) || k.Name == id {
			found = true
			continue
		}
		filtered = append(filtered, k)
	}

	if !found {
		return fmt.Errorf("llave no encontrada para eliminar: %s", id)
	}

	return s.saveUnlocked(filtered)
}

func (s *Store) loadUnlocked() ([]SSHKey, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SSHKey{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []SSHKey{}, nil
	}
	var keys []SSHKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("parseando keys.json: %w", err)
	}
	return keys, nil
}

func (s *Store) saveUnlocked(keys []SSHKey) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func SanitizeName(name string) string {
	return sanitizeName(name)
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	if result.Len() == 0 {
		return "vm"
	}
	return result.String()
}
