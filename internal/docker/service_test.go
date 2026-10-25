package docker

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
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
	if strings.Contains(executedCmd, "--cap-add NET_ADMIN") {
		t.Errorf("no se esperaba --cap-add NET_ADMIN cuando NetAdmin es false: %s", executedCmd)
	}

	// Probar creación con capacidad NET_ADMIN habilitada (--cap-add NET_ADMIN)
	reqNetAdmin := CreateContainerRequest{
		VMName:   "test-vm",
		Name:     "router-container",
		Image:    "alpine:latest",
		NetAdmin: true,
	}
	_, err = svc.CreateContainer(context.Background(), reqNetAdmin)
	if err != nil {
		t.Fatalf("error creando contenedor con NET_ADMIN: %v", err)
	}
	if !strings.Contains(executedCmd, "--cap-add NET_ADMIN") {
		t.Errorf("se esperaba '--cap-add NET_ADMIN' en el comando ejecutado, se obtuvo: %s", executedCmd)
	}
	// Asegurar que --cap-add NET_ADMIN aparece antes de la imagen
	capIdx := strings.Index(executedCmd, "--cap-add NET_ADMIN")
	imgIdx := strings.Index(executedCmd, "'alpine:latest'")
	if capIdx == -1 || imgIdx == -1 || capIdx > imgIdx {
		t.Errorf("--cap-add NET_ADMIN debe ubicarse antes de la imagen en docker run: %s", executedCmd)
	}

	// Probar creación con múltiples redes
	var capturedCommands []string
	multiNetRunner := func(ctx context.Context, vmName string, cmd string) (string, error) {
		capturedCommands = append(capturedCommands, cmd)
		return "abc123456789\n", nil
	}
	multiSvc := NewService(multiNetRunner, discardLog)
	multiReq := CreateContainerRequest{
		VMName:   "test-vm",
		Name:     "web-service",
		Image:    "nginx:alpine",
		Networks: []string{"frontend_net", "backend_net"},
	}
	_, err = multiSvc.CreateContainer(context.Background(), multiReq)
	if err != nil {
		t.Fatalf("error creando contenedor con múltiples redes: %v", err)
	}
	if len(capturedCommands) < 2 {
		t.Fatalf("se esperaban al menos 2 comandos (run y network connect), se obtuvieron %d: %v", len(capturedCommands), capturedCommands)
	}
	if !strings.Contains(capturedCommands[0], "--network 'frontend_net'") {
		t.Errorf("primer comando docker run no incluyó frontend_net: %s", capturedCommands[0])
	}
	if !strings.Contains(capturedCommands[1], "docker network connect 'backend_net' 'web-service'") {
		t.Errorf("segundo comando no conectó backend_net: %s", capturedCommands[1])
	}

	// Probar InspectContainer con redes
	inspectJSON := `[{
		"Id": "abc123456789",
		"Name": "/web-service",
		"Created": "2026-10-18T20:00:00Z",
		"State": { "Status": "running", "Running": true },
		"Config": { "Image": "nginx:alpine", "Env": [] },
		"HostConfig": { "RestartPolicy": { "Name": "unless-stopped" } },
		"NetworkSettings": {
			"IPAddress": "",
			"Networks": {
				"frontend_net": { "IPAddress": "172.20.0.2", "Gateway": "172.20.0.1", "MacAddress": "02:42:ac:14:00:02", "NetworkID": "net1" },
				"backend_net": { "IPAddress": "172.21.0.5", "Gateway": "172.21.0.1", "MacAddress": "02:42:ac:15:00:05", "NetworkID": "net2" }
			}
		}
	}]`
	inspectRunner := func(ctx context.Context, vmName string, cmd string) (string, error) {
		return inspectJSON, nil
	}
	inspSvc := NewService(inspectRunner, discardLog)
	insp, err := inspSvc.InspectContainer(context.Background(), "test-vm", "web-service")
	if err != nil {
		t.Fatalf("error en InspectContainer: %v", err)
	}
	if len(insp.Networks) != 2 {
		t.Errorf("se esperaban 2 redes en inspect, se obtuvieron: %d", len(insp.Networks))
	}
	if insp.IPAddress != "172.20.0.2" && insp.IPAddress != "172.21.0.5" {
		t.Errorf("IPAddress principal no asignada desde redes: %s", insp.IPAddress)
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

	// Commit container
	commitReq := CommitContainerRequest{
		VMName:      "test-vm",
		ContainerID: "my-container-id",
		ImageName:   "my-committed-image",
		Tag:         "v1",
		Comment:     "test comment",
	}
	commitRes, err := svc.CommitContainer(context.Background(), commitReq)
	if err != nil {
		t.Fatalf("error al hacer commit del contenedor: %v", err)
	}
	if !commitRes.Success || commitRes.Image != "my-committed-image:v1" {
		t.Errorf("respuesta inesperada en commit: %+v", commitRes)
	}
}

func TestServiceVolumeOperations(t *testing.T) {
	mockVolumeList := `{"name":"postgres_data","driver":"local","mountpoint":"/var/lib/docker/volumes/postgres_data/_data"}
{"name":"redis_data","driver":"local","mountpoint":"/var/lib/docker/volumes/redis_data/_data"}`

	mockInUse := "postgres_data\n"

	var executedCommands []string
	runner := func(ctx context.Context, vmName string, command string) (string, error) {
		executedCommands = append(executedCommands, command)
		if strings.Contains(command, "docker volume ls") {
			return mockVolumeList, nil
		}
		if strings.Contains(command, "docker ps -q") && strings.Contains(command, "volume") {
			return mockInUse, nil
		}
		if strings.Contains(command, "docker volume create") {
			return "new_volume_data", nil
		}
		if strings.Contains(command, "docker volume rm") {
			return "new_volume_data", nil
		}
		return "", nil
	}

	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(runner, discardLog)

	// List volumes
	vols, err := svc.ListVolumes(context.Background(), "test-vm")
	if err != nil {
		t.Fatalf("error listando volúmenes: %v", err)
	}
	if len(vols) != 2 {
		t.Fatalf("se esperaban 2 volúmenes, se obtuvieron: %d", len(vols))
	}
	if vols[0].Name != "postgres_data" || vols[0].VMName != "test-vm" || vols[0].Status != "En uso" {
		t.Errorf("volumen 1 mapeado incorrectamente: %+v", vols[0])
	}
	if vols[1].Status != "Sin utilizar" {
		t.Errorf("volumen 2 debió ser 'Sin utilizar': %+v", vols[1])
	}

	// Create volume
	createReq := CreateVolumeRequest{
		VMName: "test-vm",
		Name:   "new_volume_data",
		Driver: "local",
	}
	vol, err := svc.CreateVolume(context.Background(), createReq)
	if err != nil {
		t.Fatalf("error creando volumen: %v", err)
	}
	if vol.Name != "new_volume_data" || vol.Driver != "local" {
		t.Errorf("volumen creado devuelto es incorrecto: %+v", vol)
	}

	// Delete volume
	err = svc.DeleteVolume(context.Background(), "test-vm", "new_volume_data")
	if err != nil {
		t.Fatalf("error eliminando volumen: %v", err)
	}
}

func TestNetworkValidation(t *testing.T) {
	if err := ValidateNetworkName("my-net_1.0"); err != nil {
		t.Errorf("nombre de red válido rechazado: %v", err)
	}
	if err := ValidateNetworkName("bad net;rm"); err == nil {
		t.Errorf("se esperaba error para nombre de red con metacaracteres")
	}

	if err := ValidateNetworkDriver("bridge"); err != nil {
		t.Errorf("driver bridge rechazado: %v", err)
	}
	if err := ValidateNetworkDriver("host"); err != nil {
		t.Errorf("driver host rechazado: %v", err)
	}
	if err := ValidateNetworkDriver("none"); err != nil {
		t.Errorf("driver none rechazado: %v", err)
	}
	if err := ValidateNetworkDriver("macvlan"); err != nil {
		t.Errorf("driver macvlan rechazado: %v", err)
	}
	if err := ValidateNetworkDriver("ipvlan"); err != nil {
		t.Errorf("driver ipvlan rechazado: %v", err)
	}
	if err := ValidateNetworkDriver("overlay"); err != nil {
		t.Errorf("driver overlay rechazado: %v", err)
	}
	if err := ValidateNetworkDriver("invalid-driver"); err == nil {
		t.Errorf("driver inválido aceptado")
	}

	// CIDR
	if _, err := ValidateCIDR("172.20.0.0/16"); err != nil {
		t.Errorf("CIDR válido rechazado: %v", err)
	}
	if _, err := ValidateCIDR("172.20.0.0"); err == nil {
		t.Errorf("IP sin máscara aceptada como CIDR")
	}

	// Gateway & Subnet
	if err := ValidateGatewaySubnet("172.20.0.1", "172.20.0.0/16"); err != nil {
		t.Errorf("gateway válido en subred rechazado: %v", err)
	}
	if err := ValidateGatewaySubnet("10.0.0.1", "172.20.0.0/16"); err == nil {
		t.Errorf("gateway fuera de subred aceptado")
	}

	// IP range
	if err := ValidateIPRange("172.20.10.0/24", "172.20.0.0/16"); err != nil {
		t.Errorf("rango IP válido en subred rechazado: %v", err)
	}
	if err := ValidateIPRange("192.168.1.0/24", "172.20.0.0/16"); err == nil {
		t.Errorf("rango IP fuera de subred aceptado")
	}
}

func TestServiceNetworkOperations(t *testing.T) {
	mockNetList := `{"id":"1234567890ab","name":"bridge","driver":"bridge","scope":"local"}
{"id":"abcdef123456","name":"custom_net","driver":"bridge","scope":"local"}`

	mockInspectMulti := `[
		{
			"Name": "bridge",
			"Internal": false,
			"Attachable": false,
			"IPAM": { "Config": [{ "Subnet": "172.17.0.0/16", "Gateway": "172.17.0.1" }] },
			"Containers": { "c1": { "Name": "c1_app" } }
		},
		{
			"Name": "custom_net",
			"Internal": true,
			"Attachable": true,
			"IPAM": { "Config": [{ "Subnet": "172.20.0.0/16", "Gateway": "172.20.0.1" }] },
			"Containers": {}
		}
	]`

	var executedCommands []string
	runner := func(ctx context.Context, vmName string, command string) (string, error) {
		executedCommands = append(executedCommands, command)
		if strings.Contains(command, "docker network ls") {
			return mockNetList, nil
		}
		if strings.Contains(command, "docker network inspect") {
			return mockInspectMulti, nil
		}
		if strings.Contains(command, "docker network create") {
			return "9876543210fe", nil
		}
		if strings.Contains(command, "docker network rm") {
			return "custom_net", nil
		}
		return "", nil
	}

	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(runner, discardLog)

	// List networks
	nets, err := svc.ListNetworks(context.Background(), "test-vm")
	if err != nil {
		t.Fatalf("error listando redes: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("se esperaban 2 redes, se obtuvieron: %d", len(nets))
	}
	if nets[0].Name != "bridge" || nets[0].Containers != 1 || nets[0].Subnet != "172.17.0.0/16" {
		t.Errorf("red 0 mapeada incorrectamente: %+v", nets[0])
	}
	if nets[1].Name != "custom_net" || nets[1].Containers != 0 || !nets[1].Internal || !nets[1].Attachable {
		t.Errorf("red 1 mapeada incorrectamente: %+v", nets[1])
	}

	// Create network
	createReq := CreateNetworkRequest{
		VMName:     "test-vm",
		Name:       "new_net",
		Driver:     "bridge",
		Subnet:     "172.25.0.0/16",
		Gateway:    "172.25.0.1",
		Internal:   true,
		Attachable: true,
	}
	newNet, err := svc.CreateNetwork(context.Background(), createReq)
	if err != nil {
		t.Fatalf("error creando red: %v", err)
	}
	if newNet.Name != "new_net" || newNet.ID != "9876543210fe" || !newNet.Internal {
		t.Errorf("red creada inesperada: %+v", newNet)
	}

	// Check command parameters
	lastCmd := executedCommands[len(executedCommands)-1]
	if !strings.Contains(lastCmd, "--subnet '172.25.0.0/16'") || !strings.Contains(lastCmd, "--internal") {
		t.Errorf("comando create no contiene argumentos esperados: %s", lastCmd)
	}

	// Delete default network protection
	if err := svc.DeleteNetwork(context.Background(), "test-vm", "bridge"); err == nil {
		t.Fatalf("se esperaba error al intentar eliminar red predeterminada 'bridge'")
	}

	// Delete custom network
	if err := svc.DeleteNetwork(context.Background(), "test-vm", "custom_net"); err != nil {
		t.Fatalf("error eliminando red personalizada: %v", err)
	}
}

func TestBuildImageWithFiles(t *testing.T) {
	var executedCommands []string
	runner := func(ctx context.Context, vmName string, command string) (string, error) {
		executedCommands = append(executedCommands, command)
		if strings.Contains(command, "docker build") {
			return "Successfully built 1234567890ab\nSuccessfully tagged test-app:1.0", nil
		}
		return "", nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(runner, logger)

	fileB64 := base64.StdEncoding.EncodeToString([]byte("console.log('hello world');"))
	req := BuildImageRequest{
		VMName:     "my-vm",
		ImageName:  "test-app",
		Tag:        "1.0",
		Dockerfile: "FROM node:18\nCOPY app.js .\nCMD [\"node\", \"app.js\"]",
		Files: []BuildFile{
			{
				Name:    "app.js",
				Content: fileB64,
			},
		},
	}

	res, err := svc.BuildImage(context.Background(), req)
	if err != nil {
		t.Fatalf("error inesperado en BuildImage: %v", err)
	}
	if !res.Success || res.Image != "test-app:1.0" {
		t.Errorf("resultado inesperado: %+v", res)
	}

	foundAppJs := false
	for _, cmd := range executedCommands {
		if strings.Contains(cmd, "app.js") && strings.Contains(cmd, fileB64) {
			foundAppJs = true
			break
		}
	}
	if !foundAppJs {
		t.Errorf("no se encontró comando para escribir app.js en el contexto: %v", executedCommands)
	}

	// Probar validación de traversal
	badReq := req
	badReq.Files = []BuildFile{{Name: "../etc/passwd", Content: "dGVzdA=="}}
	if _, err := svc.BuildImage(context.Background(), badReq); err == nil {
		t.Errorf("se esperaba error por path traversal pero fue aceptado")
	}

	// Probar error de COPY faltante
	failRunner := func(ctx context.Context, vmName string, command string) (string, error) {
		if strings.Contains(command, "docker build") {
			return "Step 1/2 : FROM node:18\nStep 2/2 : COPY missing.js .\nCOPY failed: file not found in build context or excluded by .dockerignore: stat missing.js: file does not exist", fmt.Errorf("exit status 1")
		}
		return "", nil
	}
	svcFail := NewService(failRunner, logger)
	_, errFail := svcFail.BuildImage(context.Background(), req)
	if errFail == nil {
		t.Fatalf("se esperaba error pero la compilación tuvo éxito")
	}
	if !strings.Contains(errFail.Error(), "falta un archivo en el contexto") {
		t.Errorf("el mensaje de error no identificó la falta de archivo: %v", errFail)
	}
}

func TestAnalyzeBuildError(t *testing.T) {
	tag := "my-image:latest"

	// Caso 1: COPY clásico
	outCopy := "Step 2/2 : COPY script.sh .\nCOPY failed: file not found in build context or excluded by .dockerignore: stat script.sh: file does not exist"
	msgCopy := AnalyzeBuildError(outCopy, tag)
	if !strings.Contains(msgCopy, "falta un archivo en el contexto") {
		t.Errorf("esperaba detección de falta de archivo para COPY, obtenido: %s", msgCopy)
	}

	// Caso 2: BuildKit not found
	outBuildKit := "ERROR: failed to solve: failed to compute cache key: failed to walk /var/lib/docker/tmp: not found"
	msgBuildKit := AnalyzeBuildError(outBuildKit, tag)
	if !strings.Contains(msgBuildKit, "falta un archivo en el contexto") {
		t.Errorf("esperaba detección de falta de archivo para BuildKit, obtenido: %s", msgBuildKit)
	}

	// Caso 3: Error de sintaxis o ejecución general
	outSyntax := "Step 2/3 : RUN npm install\nnpm ERR! code ENOENT"
	msgSyntax := AnalyzeBuildError(outSyntax, tag)
	if !strings.Contains(msgSyntax, "npm ERR! code ENOENT") {
		t.Errorf("esperaba extraer última línea de error, obtenido: %s", msgSyntax)
	}
}

func TestExportImageAndListTars(t *testing.T) {
	fixedTime := time.Date(2026, 10, 24, 23, 15, 30, 0, time.UTC)

	// 1. Verificar sanitización de nombres de archivo TAR
	fnSimple := SanitizeTarFileName("mi-api:1.0", fixedTime)
	if fnSimple != "mi-api_1.0_20261024_231530.tar" {
		t.Errorf("nombre TAR inesperado para mi-api:1.0: %s", fnSimple)
	}

	fnNamespace := SanitizeTarFileName("registry.local:5000/mi-org/servicio-web:v2.1-alpine", fixedTime)
	if fnNamespace != "registry.local_5000_mi-org_servicio-web_v2.1-alpine_20261024_231530.tar" {
		t.Errorf("nombre TAR inesperado para imagen con namespace: %s", fnNamespace)
	}

	fnSha := SanitizeTarFileName("sha256:1234567890abcdef1234567890abcdef", fixedTime)
	if fnSha != "1234567890ab_20261024_231530.tar" {
		t.Errorf("nombre TAR inesperado para sha256: %s", fnSha)
	}

	// 2. Probar ExportImage exitoso
	var lastCmd string
	runnerSuccess := func(ctx context.Context, vmName string, command string) (string, error) {
		lastCmd = command
		return "TAR_OK|/home/debian/docker-image-exports/mi-api_1.0_20261024_231530.tar|15728640|1792883730\n", nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(runnerSuccess, logger)

	exportRes, err := svc.ExportImage(context.Background(), ExportImageRequest{
		VMName:   "debian-vm",
		ImageRef: "mi-api:1.0",
	})
	if err != nil {
		t.Fatalf("error inesperado en ExportImage: %v", err)
	}
	if !exportRes.Success || exportRes.SizeBytes != 15728640 || exportRes.Size != "15.0 MB" {
		t.Errorf("respuesta de exportación inesperada: %+v", exportRes)
	}
	if !strings.Contains(lastCmd, "sudo docker save -o") || !strings.Contains(lastCmd, "'mi-api:1.0'") {
		t.Errorf("el comando ejecutado no contiene docker save con la referencia original: %s", lastCmd)
	}
	if !strings.Contains(lastCmd, "docker-image-exports") {
		t.Errorf("el comando no utiliza el directorio docker-image-exports: %s", lastCmd)
	}

	// 3. Probar rechazo si el archivo generado tiene tamaño 0
	runnerZero := func(ctx context.Context, vmName string, command string) (string, error) {
		return "TAR_OK|/home/debian/docker-image-exports/vacio.tar|0|1792883730\n", nil
	}
	svcZero := NewService(runnerZero, logger)
	if _, err := svcZero.ExportImage(context.Background(), ExportImageRequest{
		VMName:   "debian-vm",
		ImageRef: "mi-api:1.0",
	}); err == nil {
		t.Errorf("se esperaba error cuando el archivo TAR tiene tamaño 0 bytes")
	}

	// 4. Probar error cuando la imagen no existe en Docker
	runnerNotFound := func(ctx context.Context, vmName string, command string) (string, error) {
		return "Error response from daemon: reference does not exist", fmt.Errorf("exit status 1")
	}
	svcNotFound := NewService(runnerNotFound, logger)
	_, errNotFound := svcNotFound.ExportImage(context.Background(), ExportImageRequest{
		VMName:   "debian-vm",
		ImageRef: "inexistente:latest",
	})
	if errNotFound == nil || !strings.Contains(errNotFound.Error(), "no existe") {
		t.Errorf("se esperaba mensaje de imagen inexistente, obtenido: %v", errNotFound)
	}

	// 5. Probar rechazo de inyección en referencia de imagen
	if _, err := svc.ExportImage(context.Background(), ExportImageRequest{
		VMName:   "debian-vm",
		ImageRef: "alpine;rm -rf /",
	}); err == nil {
		t.Errorf("se esperaba rechazo por referencia de imagen insegura")
	}

	// 6. Probar ListExportedTars ordenados por fecha de modificación
	runnerList := func(ctx context.Context, vmName string, command string) (string, error) {
		return "/home/debian/docker-image-exports/old_app_1.0.tar|2048|1792800000\n" +
			"/home/debian/docker-image-exports/new_app_2.0.tar|1048576|1792890000\n", nil
	}
	svcList := NewService(runnerList, logger)
	tars, err := svcList.ListExportedTars(context.Background(), "debian-vm")
	if err != nil {
		t.Fatalf("error inesperado listando archivos TAR: %v", err)
	}
	if len(tars) != 2 {
		t.Fatalf("se esperaban 2 archivos TAR, obtenidos: %d", len(tars))
	}
	if tars[0].FileName != "new_app_2.0.tar" || tars[0].Size != "1.0 MB" || tars[0].VMName != "debian-vm" {
		t.Errorf("primer archivo TAR inesperado (debe ser el más reciente): %+v", tars[0])
	}
	if tars[1].FileName != "old_app_1.0.tar" || tars[1].Size != "2.0 KB" {
		t.Errorf("segundo archivo TAR inesperado: %+v", tars[1])
	}

	// 7. Probar ValidateTarFileName y DeleteExportedTar
	if err := ValidateTarFileName("mi_api_1.0_20261024.tar"); err != nil {
		t.Errorf("nombre de archivo TAR válido rechazado: %v", err)
	}
	for _, invalidFile := range []string{"", "../secret.tar", "sub/file.tar", "archivo.txt", "bad;rm.tar"} {
		if err := ValidateTarFileName(invalidFile); err == nil {
			t.Errorf("se esperaba rechazo para nombre TAR inválido '%s'", invalidFile)
		}
	}

	var deletedCmd string
	runnerDel := func(ctx context.Context, vmName string, command string) (string, error) {
		deletedCmd = command
		return "", nil
	}
	svcDel := NewService(runnerDel, logger)
	if err := svcDel.DeleteExportedTar(context.Background(), "debian-vm", "new_app_2.0.tar"); err != nil {
		t.Fatalf("error inesperado eliminando archivo TAR: %v", err)
	}
	if !strings.Contains(deletedCmd, "'new_app_2.0.tar'") || !strings.Contains(deletedCmd, "docker-image-exports") {
		t.Errorf("comando de eliminación de TAR inesperado: %s", deletedCmd)
	}
}


