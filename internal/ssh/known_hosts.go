package ssh

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// KnownHostsManager administra el archivo known_hosts del backend de forma concurrente y segura
type KnownHostsManager struct {
	mu   sync.RWMutex
	path string
}

// NewKnownHostsManager inicializa el gestor de known_hosts en la ruta especificada
func NewKnownHostsManager(path string) *KnownHostsManager {
	if path == "" {
		path = filepath.Join("C:", "VMManager", "SSH", "known_hosts")
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte(""), 0600)
	}
	return &KnownHostsManager{path: path}
}

// HostKeyCallback retorna un callback de verificacion estricta contra el archivo known_hosts
func (km *KnownHostsManager) HostKeyCallback() (gossh.HostKeyCallback, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	callback, err := knownhosts.New(km.path)
	if err != nil {
		return nil, fmt.Errorf("cargando known_hosts desde %s: %w", km.path, err)
	}
	return callback, nil
}

// AddHostKey registra o actualiza la Host Key de un host:puerto en el archivo known_hosts
func (km *KnownHostsManager) AddHostKey(host string, port int, key gossh.PublicKey) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", host, port)
	if port == 22 {
		addr = host
	}

	line := knownhosts.Line([]string{addr}, key)

	// Leer entradas existentes para no duplicar
	var lines []string
	if data, err := os.ReadFile(km.path); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			l := strings.TrimSpace(scanner.Text())
			if l != "" && !strings.HasPrefix(l, addr) && !strings.HasPrefix(l, fmt.Sprintf("[%s]:%d", host, port)) {
				lines = append(lines, l)
			}
		}
	}

	lines = append(lines, line)
	output := strings.Join(lines, "\n") + "\n"

	return os.WriteFile(km.path, []byte(output), 0600)
}

// RemoveHostKey elimina las entradas de un host:puerto del archivo known_hosts
func (km *KnownHostsManager) RemoveHostKey(host string, port int) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", host, port)
	prefix1 := addr
	prefix2 := fmt.Sprintf("[%s]:%d", host, port)

	data, err := os.ReadFile(km.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var newLines []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if l != "" && !strings.HasPrefix(l, prefix1) && !strings.HasPrefix(l, prefix2) {
			newLines = append(newLines, l)
		}
	}

	output := ""
	if len(newLines) > 0 {
		output = strings.Join(newLines, "\n") + "\n"
	}

	return os.WriteFile(km.path, []byte(output), 0600)
}

// FetchAndStoreHostKey se conecta temporalmente a la direccion TCP para obtener su Host Key publica y registrarla
func (km *KnownHostsManager) FetchAndStoreHostKey(host string, port int) (gossh.PublicKey, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	var fetchedKey gossh.PublicKey
	config := &gossh.ClientConfig{
		User: "probe",
		HostKeyCallback: func(hostname string, remote net.Addr, key gossh.PublicKey) error {
			fetchedKey = key
			return nil
		},
		Timeout: 5 * 1000 * 1000 * 1000, // 5s
	}

	// Conectamos: ignoramos el error de autenticacion de usuario porque solo queremos la host key
	conn, err := gossh.Dial("tcp", addr, config)
	if conn != nil {
		conn.Close()
	}

	if fetchedKey == nil {
		return nil, fmt.Errorf("no se pudo capturar la host key de %s: %v", addr, err)
	}

	if err := km.AddHostKey(host, port, fetchedKey); err != nil {
		return nil, fmt.Errorf("guardando host key en known_hosts: %w", err)
	}

	return fetchedKey, nil
}
