package docker

import (
	"fmt"
	"regexp"
	"strings"
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

// EscapeShellArg envuelve un argumento en comillas simples seguras para POSIX shell
func EscapeShellArg(arg string) string {
	// Reemplaza comillas simples por '\'' para cerrar y reabrir comillas dentro de sh
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
