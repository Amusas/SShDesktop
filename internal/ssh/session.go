package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// CreateVirtualMachine lanza una creación de VM remota por SSH usando un comando de ejemplo seguro.
func CreateVirtualMachine(ctx context.Context, cfg Config, vm interface {
	GetName() string
	GetOS() string
	GetISO() string
	GetCores() int
	GetMemoryMB() int
	GetDiskGB() int
}, log *slog.Logger) error {
	_ = ctx
	name := sanitizeToken(vm.GetName())
	osName := sanitizeToken(vm.GetOS())
	iso := sanitizeToken(vm.GetISO())
	cores := vm.GetCores()
	memory := vm.GetMemoryMB()
	disk := vm.GetDiskGB()
	cmd := fmt.Sprintf(`mkdir -p "/tmp/ssh-web-vms" && printf 'name=%s;os=%s;iso=%s;cores=%d;memory=%d;disk=%d' > "/tmp/ssh-web-vms/%s.json"`,
		name, osName, iso, cores, memory, disk, name)
	if _, err := ExecuteCommand(cfg, cmd, log); err != nil {
		return fmt.Errorf("creando maquina virtual remota por SSH: %w", err)
	}
	log.Info("maquina virtual registrada remotamente por SSH", "name", vm.GetName(), "iso", vm.GetISO())
	return nil
}

func ExecuteCommand(cfg Config, command string, log *slog.Logger) (string, error) {
	return ExecuteCommandContext(context.Background(), cfg, command, log)
}

// ExecuteCommandContext ejecuta un comando en el servidor SSH remoto (o VM via ProxyHost) con soporte de contexto y cancelacion
func ExecuteCommandContext(ctx context.Context, cfg Config, command string, log *slog.Logger) (string, error) {
	authMethods, err := buildAuthMethods(cfg)
	if err != nil {
		return "", fmt.Errorf("construyendo metodos de autenticacion: %w", err)
	}

	clientConfig := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}

	var client *gossh.Client
	var proxyClient *gossh.Client

	if cfg.ProxyHost != nil {
		pClient, err := DialHost(*cfg.ProxyHost)
		if err != nil {
			return "", fmt.Errorf("conectando al host intermedio: %w", err)
		}
		proxyClient = pClient
		defer proxyClient.Close()

		targetAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
		netConn, err := proxyClient.Dial("tcp", targetAddr)
		if err != nil {
			return "", fmt.Errorf("conectando a %s via tunel: %w", targetAddr, err)
		}
		sshConn, chans, reqs, err := gossh.NewClientConn(netConn, targetAddr, clientConfig)
		if err != nil {
			netConn.Close()
			return "", fmt.Errorf("autenticando SSH con destino %s: %w", targetAddr, err)
		}
		client = gossh.NewClient(sshConn, chans, reqs)
	} else {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		c, err := gossh.Dial("tcp", addr, clientConfig)
		if err != nil {
			return "", fmt.Errorf("conectando al servidor SSH: %w", err)
		}
		client = c
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("creando sesion SSH para comando: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Soporte de cancelacion por contexto
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(gossh.SIGKILL)
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			if stderr.Len() > 0 {
				return stdout.String(), fmt.Errorf("ejecutando comando remoto: %w: %s", err, strings.TrimSpace(stderr.String()))
			}
			return stdout.String(), fmt.Errorf("ejecutando comando remoto: %w", err)
		}
	}

	out := strings.TrimSpace(stdout.String())
	return out, nil
}

// StreamCommandContext ejecuta un comando remoto por SSH y transmite su salida estándar directamente al io.Writer indicado
func StreamCommandContext(ctx context.Context, cfg Config, command string, stdout io.Writer, log *slog.Logger) error {
	authMethods, err := buildAuthMethods(cfg)
	if err != nil {
		return fmt.Errorf("construyendo metodos de autenticacion: %w", err)
	}

	clientConfig := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}

	var client *gossh.Client
	var proxyClient *gossh.Client

	if cfg.ProxyHost != nil {
		pClient, err := DialHost(*cfg.ProxyHost)
		if err != nil {
			return fmt.Errorf("conectando al host intermedio: %w", err)
		}
		proxyClient = pClient
		defer proxyClient.Close()

		targetAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
		netConn, err := proxyClient.Dial("tcp", targetAddr)
		if err != nil {
			return fmt.Errorf("conectando a %s via tunel: %w", targetAddr, err)
		}
		sshConn, chans, reqs, err := gossh.NewClientConn(netConn, targetAddr, clientConfig)
		if err != nil {
			netConn.Close()
			return fmt.Errorf("autenticando SSH con destino %s: %w", targetAddr, err)
		}
		client = gossh.NewClient(sshConn, chans, reqs)
	} else {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		c, err := gossh.Dial("tcp", addr, clientConfig)
		if err != nil {
			return fmt.Errorf("conectando al servidor SSH: %w", err)
		}
		client = c
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("creando sesion SSH para streaming: %w", err)
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stdout = stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(gossh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("transmitiendo comando remoto: %w: %s", err, strings.TrimSpace(stderr.String()))
			}
			return fmt.Errorf("transmitiendo comando remoto: %w", err)
		}
	}

	return nil
}

func sanitizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	return re.ReplaceAllString(value, "-")
}

const readBufferSize = 4096

// Session encapsula la conexion SSH, el canal PTY y las tuberias de entrada/salida
type Session struct {
	client      *gossh.Client
	proxyClient *gossh.Client
	session     *gossh.Session
	stdin       io.WriteCloser
	log         *slog.Logger
	closed      atomic.Bool
	wg          sync.WaitGroup
	onOutput    func([]byte)
	onClose     func()
}

// Config contiene las credenciales y parametros para establecer la sesion SSH
type Config struct {
	Host        string
	Port        int
	User        string
	KeyPath     string
	Shell       string // "powershell", "cmd", o "default"
	StartPath   string // Directorio inicial predeterminado
	ProxyHost   *Config
	ExecCommand string // Si no esta vacio, se ejecuta este comando interactivo en la PTY en lugar de una shell
}

// DialHost establece una conexion SSH con el servidor configurado
func DialHost(cfg Config) (*gossh.Client, error) {
	authMethods, err := buildAuthMethods(cfg)
	if err != nil {
		return nil, fmt.Errorf("construyendo metodos de autenticacion: %w", err)
	}

	clientConfig := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return gossh.Dial("tcp", addr, clientConfig)
}

// NewSession establece la conexion TCP, autentica al usuario y solicita una PTY interactiva
func NewSession(ctx context.Context, cfg Config, log *slog.Logger) (*Session, error) {
	authMethods, err := buildAuthMethods(cfg)
	if err != nil {
		return nil, fmt.Errorf("construyendo metodos de autenticacion: %w", err)
	}

	clientConfig := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	var client *gossh.Client
	var proxyClient *gossh.Client

	if cfg.ProxyHost != nil {
		pClient, err := DialHost(*cfg.ProxyHost)
		if err != nil {
			return nil, fmt.Errorf("conectando al host intermedio: %w", err)
		}
		proxyClient = pClient

		targetAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
		log.Info("conectando a la VM mediante tunel SSH", "intermediario", cfg.ProxyHost.Host, "destino", targetAddr, "usuario", cfg.User)

		netConn, err := proxyClient.Dial("tcp", targetAddr)
		if err != nil {
			proxyClient.Close()
			return nil, fmt.Errorf("conectando a %s via tunel: %w", targetAddr, err)
		}

		sshConn, chans, reqs, err := gossh.NewClientConn(netConn, targetAddr, clientConfig)
		if err != nil {
			netConn.Close()
			proxyClient.Close()
			return nil, fmt.Errorf("autenticando SSH con la VM en %s: %w", targetAddr, err)
		}
		client = gossh.NewClient(sshConn, chans, reqs)
	} else {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		log.Info("conectando al servidor SSH", "direccion", addr, "usuario", cfg.User, "shell", cfg.Shell)

		c, err := gossh.Dial("tcp", addr, clientConfig)
		if err != nil {
			return nil, fmt.Errorf("conectando al servidor SSH: %w", err)
		}
		client = c
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("creando sesion SSH: %w", err)
	}

	modes := gossh.TerminalModes{
		gossh.ECHO:          0,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", 1000, 120, modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("solicitando PTY: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("abriendo canal stdin: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("abriendo canal stdout: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("abriendo canal stderr: %w", err)
	}

	shellType := strings.ToLower(strings.TrimSpace(cfg.Shell))
	var startErr error

	if strings.TrimSpace(cfg.ExecCommand) != "" {
		log.Info("iniciando comando interactivo directo en PTY", "comando", cfg.ExecCommand)
		startErr = session.Start(cfg.ExecCommand)
	} else {
		switch shellType {
		case "cmd", "cmd.exe":
			cmdStr := "cmd.exe"
			if cfg.StartPath != "" {
				cmdStr = fmt.Sprintf(`cmd.exe /k "cd /d %s"`, cfg.StartPath)
			}
			log.Info("iniciando shell CMD", "comando", cmdStr)
			startErr = session.Start(cmdStr)

		case "powershell", "powershell.exe", "pwsh":
			cmdStr := "powershell.exe -NoExit"
			if cfg.StartPath != "" {
				cmdStr = fmt.Sprintf(`powershell.exe -NoExit -Command "Set-Location -LiteralPath '%s'"`, cfg.StartPath)
			}
			log.Info("iniciando shell PowerShell", "comando", cmdStr)
			startErr = session.Start(cmdStr)

		default:
			if cfg.StartPath != "" {
				cmdStr := fmt.Sprintf(`cmd.exe /k "cd /d %s"`, cfg.StartPath)
				log.Info("iniciando shell por defecto con ruta", "comando", cmdStr)
				startErr = session.Start(cmdStr)
				if startErr != nil {
					startErr = session.Shell()
				}
			} else {
				log.Info("iniciando shell interactiva por defecto")
				startErr = session.Shell()
			}
		}
	}

	if startErr != nil {
		stdin.Close()
		session.Close()
		client.Close()
		if proxyClient != nil {
			proxyClient.Close()
		}
		return nil, fmt.Errorf("iniciando shell (%s): %w", cfg.Shell, startErr)
	}

	s := &Session{
		client:      client,
		proxyClient: proxyClient,
		session:     session,
		stdin:       stdin,
		log:         log,
	}

	addrDisplay := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	if cfg.ProxyHost != nil {
		addrDisplay = fmt.Sprintf("127.0.0.1:%d (via %s)", cfg.Port, cfg.ProxyHost.Host)
	}
	log.Info("sesion SSH establecida", "direccion", addrDisplay, "shell", cfg.Shell)

	s.wg.Add(2)
	go s.pumpOutput(ctx, stdout, "stdout")
	go s.pumpOutput(ctx, stderr, "stderr")

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	return s, nil
}

func (s *Session) SetOutputHandler(fn func([]byte)) {
	s.onOutput = fn
}

func (s *Session) SetCloseHandler(fn func()) {
	s.onClose = fn
}

func (s *Session) Write(data []byte) error {
	if s.closed.Load() {
		return fmt.Errorf("la sesion esta cerrada")
	}
	_, err := s.stdin.Write(data)
	return err
}

func (s *Session) Resize(cols, rows int) error {
	if s.closed.Load() {
		return fmt.Errorf("la sesion esta cerrada")
	}
	if cols < 80 {
		cols = 80
	}
	if rows < 1000 {
		rows = 1000
	}
	s.log.Debug("redimensionando PTY", "columnas", cols, "filas", rows)
	return s.session.WindowChange(rows, cols)
}

// pumpOutput lee fragmentos del flujo remoto y los entrega inmediatamente al manejador de salida
func (s *Session) pumpOutput(ctx context.Context, reader io.Reader, stream string) {
	defer s.wg.Done()

	buf := make([]byte, readBufferSize)
	s.log.Debug("bomba de salida iniciada", "flujo", stream, "tamano_buffer", readBufferSize)

	for {
		select {
		case <-ctx.Done():
			s.log.Debug("bomba de salida detenida por contexto", "flujo", stream)
			return
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 && s.onOutput != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			s.onOutput(chunk)
		}

		if err != nil {
			if err != io.EOF {
				s.log.Warn("error de lectura en bomba de salida", "flujo", stream, "error", err)
			}
			s.log.Debug("bomba de salida finalizada", "flujo", stream)
			if stream == "stdout" && !s.closed.Load() && s.onClose != nil {
				s.onClose()
			}
			return
		}
	}
}

// Close realiza la liberacion ordenada de descriptores y espera que las goroutines finalicen
func (s *Session) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}

	s.log.Info("cerrando sesion SSH")

	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.session != nil {
		s.session.Close()
	}
	if s.client != nil {
		s.client.Close()
	}
	if s.proxyClient != nil {
		s.proxyClient.Close()
	}

	s.wg.Wait()
	s.log.Debug("goroutines de la sesion SSH liberadas")

	return nil
}

func buildAuthMethods(cfg Config) ([]gossh.AuthMethod, error) {
	var methods []gossh.AuthMethod

	if cfg.KeyPath != "" {
		keyData, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("leyendo archivo de llave: %w", err)
		}

		signer, err := gossh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("analizando llave privada: %w", err)
		}

		methods = append(methods, gossh.PublicKeys(signer))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no hay llave privada SSH configurada")
	}

	return methods, nil
}
