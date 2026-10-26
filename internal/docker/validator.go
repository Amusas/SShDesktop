package docker

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

var (
	// Docker oficial: [a-zA-Z0-9][a-zA-Z0-9_.-]+
	containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)

	// Nombres de imagen: repo/image:tag o registry:port/repo/image:tag
	imageNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_./:-]{1,128}$`)

	// Nombres de repositorio para build: repo o usuario/repo
	imageRepoRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*(/[a-zA-Z0-9][a-zA-Z0-9_.-]*)*$`)

	// Etiquetas de imagen Docker
	imageTagRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)

	// Claves de variables de entorno: [a-zA-Z_][a-zA-Z0-9_]*
	envKeyRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,63}$`)

	// Restricción de caracteres peligrosos para shell
	dangerousCharsRegex = regexp.MustCompile(`[;&|` + "`" + `$\n\r\t<>\\!]`)
)

// ValidateImageRepository valida que el nombre del repositorio de la imagen sea seguro y válido
func ValidateImageRepository(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("el nombre de la imagen no puede estar vacío")
	}
	if !imageRepoRegex.MatchString(name) || dangerousCharsRegex.MatchString(name) {
		return fmt.Errorf("nombre de imagen inválido o inseguro: '%s'", name)
	}
	return nil
}

// ValidateImageTag valida que la etiqueta de la imagen sea segura
func ValidateImageTag(tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	if !imageTagRegex.MatchString(tag) || dangerousCharsRegex.MatchString(tag) {
		return fmt.Errorf("etiqueta de imagen inválida o insegura: '%s'", tag)
	}
	return nil
}

// ValidateDockerfile comprueba que el contenido del Dockerfile sea válido y no esté vacío
func ValidateDockerfile(content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fmt.Errorf("el contenido del Dockerfile no puede estar vacío")
	}
	if len(content) > 2*1024*1024 {
		return fmt.Errorf("el tamaño del Dockerfile excede el límite permitido (2MB)")
	}
	return nil
}


// ValidateContainerName verifica que el nombre de contenedor cumpla el formato seguro
func ValidateContainerName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("el nombre del contenedor no puede estar vacío")
	}
	if !containerNameRegex.MatchString(name) {
		return fmt.Errorf("nombre de contenedor inválido: '%s' (solo letras, números, puntos, guiones y guiones bajos)", name)
	}
	return nil
}

// ValidateVolumeName verifica que el nombre de un volumen cumpla el formato seguro
func ValidateVolumeName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("el nombre del volumen no puede estar vacío")
	}
	if !containerNameRegex.MatchString(name) {
		return fmt.Errorf("nombre de volumen inválido: '%s' (solo letras, números, puntos, guiones y guiones bajos)", name)
	}
	return nil
}

// ValidateImageName verifica que el nombre y etiqueta de la imagen sean válidos y seguros
func ValidateImageName(image string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return fmt.Errorf("la imagen del contenedor no puede estar vacía")
	}
	if !imageNameRegex.MatchString(image) || dangerousCharsRegex.MatchString(image) {
		return fmt.Errorf("nombre de imagen inválido o inseguro: '%s'", image)
	}
	return nil
}

// ValidatePort verifica que el puerto esté dentro del rango TCP/UDP válido
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("puerto fuera de rango (1-65535): %d", port)
	}
	return nil
}

// ValidateEnvKey verifica que la clave de entorno sea un identificador POSIX válido
func ValidateEnvKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("la clave de variable de entorno no puede estar vacía")
	}
	if !envKeyRegex.MatchString(key) {
		return fmt.Errorf("clave de variable de entorno inválida: '%s'", key)
	}
	return nil
}

// ValidateVolumePath verifica que una ruta de montaje de volumen no contenga metacaracteres
func ValidateVolumePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("la ruta del volumen no puede estar vacía")
	}
	if dangerousCharsRegex.MatchString(path) || strings.Contains(path, " ") {
		return fmt.Errorf("ruta de volumen insegura o con espacios: '%s'", path)
	}
	return nil
}

// ValidateRestartPolicy valida la política de reinicio contra la lista permitida
func ValidateRestartPolicy(policy string) error {
	if policy == "" {
		return nil
	}
	policy = strings.ToLower(strings.TrimSpace(policy))
	switch policy {
	case "unless-stopped", "always", "on-failure", "no":
		return nil
	default:
		return fmt.Errorf("política de reinicio no soportada: '%s' (permitidas: unless-stopped, always, on-failure, no)", policy)
	}
}

// ValidateNetworkName verifica que el nombre de una red cumpla el formato seguro
func ValidateNetworkName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("el nombre de la red no puede estar vacío")
	}
	if !containerNameRegex.MatchString(name) {
		return fmt.Errorf("nombre de red inválido: '%s' (solo letras, números, puntos, guiones y guiones bajos)", name)
	}
	return nil
}

// ValidateNetworkDriver verifica que el driver de red sea compatible (bridge, host, none, macvlan, ipvlan o overlay)
func ValidateNetworkDriver(driver string) error {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver == "" || driver == "bridge" || driver == "host" || driver == "none" || driver == "macvlan" || driver == "ipvlan" || driver == "overlay" {
		return nil
	}
	return fmt.Errorf("driver de red no soportado: '%s' (permitidos: bridge, host, none, macvlan, ipvlan, overlay)", driver)
}

// ValidateCIDR verifica que la cadena sea una subred válida en formato CIDR (ej. 172.20.0.0/16)
func ValidateCIDR(cidr string) (*net.IPNet, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return nil, nil
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("formato CIDR de subred inválido: '%s' (ej. 172.20.0.0/16)", cidr)
	}
	return ipNet, nil
}

// ValidateIP verifica que la cadena sea una dirección IP válida
func ValidateIP(ipStr string) (net.IP, error) {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return nil, nil
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("dirección IP inválida: '%s'", ipStr)
	}
	return ip, nil
}

// ValidateGatewaySubnet verifica que el gateway pertenezca a la subred indicada
func ValidateGatewaySubnet(gatewayStr string, subnetStr string) error {
	gatewayStr = strings.TrimSpace(gatewayStr)
	subnetStr = strings.TrimSpace(subnetStr)
	if gatewayStr == "" {
		return nil
	}
	if subnetStr == "" {
		return fmt.Errorf("no se puede configurar un gateway sin especificar una subred")
	}
	gw, err := ValidateIP(gatewayStr)
	if err != nil {
		return err
	}
	subNet, err := ValidateCIDR(subnetStr)
	if err != nil {
		return err
	}
	if !subNet.Contains(gw) {
		return fmt.Errorf("la IP del gateway '%s' no pertenece a la subred '%s'", gatewayStr, subnetStr)
	}
	return nil
}

// ValidateIPRange verifica que el rango IP esté dentro de la subred
func ValidateIPRange(ipRangeStr string, subnetStr string) error {
	ipRangeStr = strings.TrimSpace(ipRangeStr)
	subnetStr = strings.TrimSpace(subnetStr)
	if ipRangeStr == "" {
		return nil
	}
	if subnetStr == "" {
		return fmt.Errorf("no se puede configurar un rango IP sin especificar una subred")
	}
	rangeNet, err := ValidateCIDR(ipRangeStr)
	if err != nil {
		return fmt.Errorf("formato CIDR de rango IP inválido: '%s'", ipRangeStr)
	}
	subNet, err := ValidateCIDR(subnetStr)
	if err != nil {
		return err
	}
	// El inicio del rango IP debe estar dentro de la subred
	if !subNet.Contains(rangeNet.IP) {
		return fmt.Errorf("el rango IP '%s' no está contenido en la subred '%s'", ipRangeStr, subnetStr)
	}
	return nil
}

// EscapeShellArg envuelve un argumento en comillas simples seguras para POSIX shell
func EscapeShellArg(arg string) string {
	// Reemplaza comillas simples por '\'' para cerrar y reabrir comillas dentro de sh
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

var unsafeTarFileCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
var multiUnderscoreRegex = regexp.MustCompile(`_+`)

// SanitizeTarFileName genera un nombre de archivo .tar seguro en Linux a partir de una referencia de imagen y marca de tiempo
func SanitizeTarFileName(imageRef string, ts time.Time) string {
	cleaned := strings.TrimSpace(imageRef)
	if strings.HasPrefix(strings.ToLower(cleaned), "sha256:") {
		cleaned = cleaned[7:]
		if len(cleaned) > 12 {
			cleaned = cleaned[:12]
		}
	}

	safeBase := unsafeTarFileCharsRegex.ReplaceAllString(cleaned, "_")
	safeBase = multiUnderscoreRegex.ReplaceAllString(safeBase, "_")
	safeBase = strings.Trim(safeBase, "_.")

	if safeBase == "" {
		safeBase = "docker_image"
	}
	if len(safeBase) > 80 {
		safeBase = strings.TrimRight(safeBase[:80], "_.")
	}

	return fmt.Sprintf("%s_%s.tar", safeBase, ts.Format("20060102_150405"))
}

var safeTarFileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,160}\.tar$`)

// ValidateTarFileName verifica que el nombre de archivo TAR sea un nombre base seguro sin rutas relativas
func ValidateTarFileName(fileName string) error {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return fmt.Errorf("el nombre del archivo TAR no puede estar vacío")
	}
	if strings.Contains(fileName, "..") || strings.ContainsAny(fileName, `/\`) || !safeTarFileNameRegex.MatchString(fileName) {
		return fmt.Errorf("nombre de archivo TAR inválido o inseguro: '%s'", fileName)
	}
	return nil
}

var dockerHubUserRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{1,63}$`)
var dockerHubRepoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// ValidateDockerHubUsername valida que el usuario o Docker ID de Docker Hub sea seguro
func ValidateDockerHubUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("el usuario de Docker Hub no puede estar vacío")
	}
	if !dockerHubUserRegex.MatchString(username) || dangerousCharsRegex.MatchString(username) {
		return fmt.Errorf("usuario de Docker Hub inválido: '%s' (solo letras, números, puntos, guiones y guiones bajos)", username)
	}
	return nil
}

// ValidateDockerHubRepoName valida que el nombre del repositorio de destino en Docker Hub sea seguro
func ValidateDockerHubRepoName(repo string) error {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("el nombre del repositorio de destino no puede estar vacío")
	}
	if !dockerHubRepoNameRegex.MatchString(repo) || dangerousCharsRegex.MatchString(repo) {
		return fmt.Errorf("nombre de repositorio de destino inválido: '%s' (solo letras, números, puntos, guiones y guiones bajos)", repo)
	}
	return nil
}



