package docker

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	// Container names
	if err := ValidateContainerName("my-postgres_1.0"); err != nil {
		t.Errorf("nombre válido rechazado: %v", err)
	}
	invalidNames := []string{"", "bad name", "injected;rm -rf /", "name$(whoami)", "name`id`", "name|pipe", "-leadingdash"}
	for _, bad := range invalidNames {
		if err := ValidateContainerName(bad); err == nil {
			t.Errorf("se esperaba error para nombre inválido '%s'", bad)
		}
	}

	// Images
	if err := ValidateImageName("postgres:16-alpine"); err != nil {
		t.Errorf("imagen válida rechazada: %v", err)
	}
	if err := ValidateImageName("registry.local:5000/my-org/my-image:v1.2"); err != nil {
		t.Errorf("imagen válida con registro rechazada: %v", err)
	}
	invalidImages := []string{"", "image;echo bad", "image&echo bad", "image$var"}
	for _, bad := range invalidImages {
		if err := ValidateImageName(bad); err == nil {
			t.Errorf("se esperaba error para imagen inválida '%s'", bad)
		}
	}

	// Ports
	if err := ValidatePort(80); err != nil {
		t.Errorf("puerto 80 válido rechazado: %v", err)
	}
	if err := ValidatePort(0); err == nil {
		t.Errorf("puerto 0 no debe ser válido")
	}
	if err := ValidatePort(65536); err == nil {
		t.Errorf("puerto 65536 no debe ser válido")
	}

	// Env keys
	if err := ValidateEnvKey("POSTGRES_PASSWORD"); err != nil {
		t.Errorf("clave de env válida rechazada: %v", err)
	}
	if err := ValidateEnvKey("123BAD"); err == nil {
		t.Errorf("clave numérica no debe ser válida")
	}
	if err := ValidateEnvKey("KEY;DROP"); err == nil {
		t.Errorf("clave con metacaracteres no debe ser válida")
	}

	// Volume paths
	if err := ValidateVolumePath("/var/lib/postgresql/data"); err != nil {
		t.Errorf("ruta válida rechazada: %v", err)
	}
	if err := ValidateVolumePath("/var/lib/data;rm -rf"); err == nil {
		t.Errorf("ruta con metacaracteres no debe ser válida")
	}

	// Shell arg escaping
	escaped := EscapeShellArg("hello'world")
	if escaped != "'hello'\\''world'" {
		t.Errorf("escape inesperado: %s", escaped)
	}
}

func TestCatalog(t *testing.T) {
	cat := GetDefaultCatalog()
	if len(cat) == 0 {
		t.Fatalf("catálogo vacío")
	}
	hasDatabases := false
	hasWeb := false
	for _, c := range cat {
		if c.ID == "databases" {
			hasDatabases = true
		}
		if c.ID == "web" {
			hasWeb = true
		}
	}
	if !hasDatabases || !hasWeb {
		t.Errorf("faltan categorías esenciales en el catálogo")
	}
}

func TestServiceListAndCreate(t *testing.T) {
	mockOutput := `{"id":"c123","name":"my-redis","image":"redis:alpine","command":"redis-server","createdAt":"2026-09-20","state":"running","status":"Up 1 hour","ports":"0.0.0.0:6379->6379/tcp"}`
	var executedCmd string

	runner := func(ctx context.Context, vmName string, command string) (string, error) {
		executedCmd = command
		if strings.Contains(command, "docker ps") {
			return mockOutput, nil
		}
		if strings.Contains(command, "docker run") {
			return "c123456789abc", nil
		}
		return "", nil
	}

	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(runner, discardLog)

	list, err := svc.ListContainers(context.Background(), "test-vm")
	if err != nil {
		t.Fatalf("error listando contenedores: %v", err)
	}
	if len(list) != 1 || list[0].Name != "my-redis" || list[0].VMName != "test-vm" {
		t.Fatalf("datos de contenedor mapeados incorrectamente: %+v", list)
	}

	req := CreateContainerRequest{
		VMName: "test-vm",
		Name:   "app-nginx",
		Image:  "nginx:alpine",
		Ports: []PortMapping{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		},
		Env: []EnvVar{
			{Key: "ENV_NAME", Value: "prod"},
		},
		RestartPolicy: "unless-stopped",
	}

	created, err := svc.CreateContainer(context.Background(), req)
	if err != nil {
		t.Fatalf("error creando contenedor: %v", err)
	}
	if created.Name != "app-nginx" || created.ID != "c123456789ab" {
		t.Fatalf("contenedor creado inesperado: %+v", created)
	}

	if !strings.Contains(executedCmd, "-p '8080:80/tcp'") {
		t.Errorf("puerto no incluido correctamente en comando: %s", executedCmd)
	}
	if !strings.Contains(executedCmd, "-e 'ENV_NAME=prod'") {
		t.Errorf("variable de entorno no incluida correctamente en comando: %s", executedCmd)
	}
}

func TestImageValidation(t *testing.T) {
	// Repositories
	if err := ValidateImageRepository("my-app"); err != nil {
		t.Errorf("repositorio válido rechazado: %v", err)
	}
	if err := ValidateImageRepository("user/repo"); err != nil {
		t.Errorf("repositorio con namespace rechazado: %v", err)
	}
	invalidRepos := []string{"", "bad;repo", "repo&bad", "repo|bad", "repo name"}
	for _, bad := range invalidRepos {
		if err := ValidateImageRepository(bad); err == nil {
			t.Errorf("se esperaba error para repositorio inválido '%s'", bad)
		}
	}

	// Tags
	if err := ValidateImageTag("latest"); err != nil {
		t.Errorf("etiqueta válida rechazada: %v", err)
	}
	if err := ValidateImageTag("v1.2.3-alpine"); err != nil {
		t.Errorf("etiqueta válida con guiones y puntos rechazada: %v", err)
	}
	if err := ValidateImageTag(""); err != nil {
		t.Errorf("etiqueta vacía no debe dar error (se usa default): %v", err)
	}
	if err := ValidateImageTag("tag;rm"); err == nil {
		t.Errorf("se esperaba error para etiqueta con punto y coma")
	}

	// Dockerfile
	if err := ValidateDockerfile("FROM alpine\nCMD [\"echo\", \"hi\"]"); err != nil {
		t.Errorf("dockerfile válido rechazado: %v", err)
	}
	if err := ValidateDockerfile("   \n\t  "); err == nil {
		t.Errorf("dockerfile en blanco no debe ser válido")
	}
}

func TestServiceImageOperations(t *testing.T) {
	mockImageList := `{"id":"sha256:1234567890ab","repository":"nginx","tag":"alpine","size":"23.5MB","createdAt":"2 days ago"}
{"id":"sha256:abcdef123456","repository":"redis","tag":"latest","size":"110MB","createdAt":"5 days ago"}`

	var executedCommands []string
	runner := func(ctx context.Context, vmName string, command string) (string, error) {
		executedCommands = append(executedCommands, command)
		if strings.Contains(command, "docker image ls") {
			return mockImageList, nil
		}
		if strings.Contains(command, "docker build") {
			return "Successfully built 1234567890ab\nSuccessfully tagged my-node:latest", nil
		}
		if strings.Contains(command, "docker rmi") {
			return "Untagged: redis:latest\nDeleted: sha256:abcdef", nil
		}
		return "", nil
	}

	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(runner, discardLog)

	// List images
	imgs, err := svc.ListImages(context.Background(), "test-vm")
	if err != nil {
		t.Fatalf("error listando imágenes: %v", err)
	}
	if len(imgs) != 2 {
		t.Fatalf("se esperaban 2 imágenes, se obtuvieron: %d", len(imgs))
	}
	if imgs[0].Repository != "nginx" || imgs[0].Tag != "alpine" || imgs[0].VMName != "test-vm" {
		t.Errorf("imagen mapeada incorrectamente: %+v", imgs[0])
	}

	// Build image
	buildReq := BuildImageRequest{
		VMName:     "test-vm",
		ImageName:  "my-node",
		Tag:        "1.0",
		Dockerfile: "FROM node:18-alpine\nWORKDIR /app\n",
	}
	res, err := svc.BuildImage(context.Background(), buildReq)
	if err != nil {
		t.Fatalf("error construyendo imagen: %v", err)
	}
	if !res.Success || res.Image != "my-node:1.0" {
		t.Errorf("respuesta de build inesperada: %+v", res)
	}

	// Verify cleanup was executed
	foundCleanup := false
	foundBase64Write := false
	for _, cmd := range executedCommands {
		if strings.Contains(cmd, "rm -rf") && strings.Contains(cmd, "docker-build-") {
			foundCleanup = true
		}
		if strings.Contains(cmd, "base64 -d >") {
			foundBase64Write = true
		}
	}
	if !foundCleanup {
		t.Errorf("no se ejecutó comando de limpieza para build context")
	}
	if !foundBase64Write {
		t.Errorf("no se escribió el dockerfile con base64")
	}

	// Delete image
	err = svc.DeleteImage(context.Background(), "test-vm", "redis:latest", false)
	if err != nil {
		t.Fatalf("error eliminando imagen: %v", err)
	}
}

