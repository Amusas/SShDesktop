package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/internal/docker"
	"github.com/feliren/ssh-web/internal/template"
	"github.com/feliren/ssh-web/internal/vm"
)

func TestImageAPIEndpoints(t *testing.T) {
	cfg := &config.Config{
		SSHHost: "127.0.0.1",
		SSHPort: 22,
		SSHUser: "testuser",
		SSHRoot: t.TempDir(),
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockRunner := func(ctx context.Context, vmName string, command string) (string, error) {
		if strings.Contains(command, "docker image ls") {
			return `{"id":"sha256:1111","repository":"alpine","tag":"latest","size":"5MB","createdAt":"yesterday"}`, nil
		}
		if strings.Contains(command, "docker build") {
			return "Successfully built 1111\nSuccessfully tagged my-app:latest", nil
		}
		if strings.Contains(command, "docker rmi") {
			return "Untagged: alpine:latest", nil
		}
		if strings.Contains(command, "docker commit") {
			return "sha256:2222", nil
		}
		return "", nil
	}

	h := &Handler{
		cfg:       cfg,
		log:       log,
		vmSvc:     vm.NewService(cfg, log),
		tplSvc:    template.NewService(cfg, log),
		dockerSvc: docker.NewService(mockRunner, log),
	}

	// 1. GET /api/images?vm=test-vm
	reqList := httptest.NewRequest(http.MethodGet, "/api/images?vm=test-vm", nil)
	rrList := httptest.NewRecorder()
	h.ServeHTTP(rrList, reqList)

	if rrList.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200, obtenido: %d", rrList.Code)
	}
	var images []docker.Image
	if err := json.NewDecoder(rrList.Body).Decode(&images); err != nil {
		t.Fatalf("error decodificando respuesta: %v", err)
	}
	if len(images) != 1 || images[0].Repository != "alpine" {
		t.Errorf("datos de imagen inesperados: %+v", images)
	}

	// 2. POST /api/images/build
	buildReq := docker.BuildImageRequest{
		VMName:     "test-vm",
		ImageName:  "my-app",
		Tag:        "latest",
		Dockerfile: "FROM alpine:latest\nCOPY script.sh .\nCMD [\"sh\", \"script.sh\"]",
		Files: []docker.BuildFile{
			{
				Name:    "script.sh",
				Content: "ZWNobyAnaGVsbG8nCg==",
			},
		},
	}
	bodyBytes, _ := json.Marshal(buildReq)
	reqBuild := httptest.NewRequest(http.MethodPost, "/api/images/build", bytes.NewReader(bodyBytes))
	reqBuild.Header.Set("Content-Type", "application/json")
	rrBuild := httptest.NewRecorder()
	h.ServeHTTP(rrBuild, reqBuild)

	if rrBuild.Code != http.StatusCreated {
		t.Fatalf("se esperaba status 201 en build, obtenido: %d, body: %s", rrBuild.Code, rrBuild.Body.String())
	}
	var buildRes docker.BuildImageResponse
	if err := json.NewDecoder(rrBuild.Body).Decode(&buildRes); err != nil {
		t.Fatalf("error decodificando respuesta de build: %v", err)
	}
	if !buildRes.Success || buildRes.Image != "my-app:latest" {
		t.Errorf("respuesta de build inesperada: %+v", buildRes)
	}

	// 3. DELETE /api/images?vm=test-vm&image=alpine:latest
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/images?vm=test-vm&image=alpine:latest", nil)
	rrDel := httptest.NewRecorder()
	h.ServeHTTP(rrDel, reqDel)

	if rrDel.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200 en delete, obtenido: %d, body: %s", rrDel.Code, rrDel.Body.String())
	}

	// 4. DELETE /api/images/test-vm/alpine:latest
	reqDelPath := httptest.NewRequest(http.MethodDelete, "/api/images/test-vm/alpine:latest", nil)
	rrDelPath := httptest.NewRecorder()
	h.ServeHTTP(rrDelPath, reqDelPath)

	if rrDelPath.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200 en delete con ruta, obtenido: %d", rrDelPath.Code)
	}

	// 5. POST /api/containers/test-vm/c123/commit
	commitPayload := docker.CommitContainerRequest{
		ImageName: "backup-image",
		Tag:       "v2.0",
		Comment:   "snapshot",
	}
	commitBytes, _ := json.Marshal(commitPayload)
	reqCommit := httptest.NewRequest(http.MethodPost, "/api/containers/test-vm/c123/commit", bytes.NewReader(commitBytes))
	reqCommit.Header.Set("Content-Type", "application/json")
	rrCommit := httptest.NewRecorder()
	h.ServeHTTP(rrCommit, reqCommit)

	if rrCommit.Code != http.StatusCreated {
		t.Fatalf("se esperaba status 201 en commit, obtenido: %d, body: %s", rrCommit.Code, rrCommit.Body.String())
	}
	var commitResp docker.CommitContainerResponse
	if err := json.NewDecoder(rrCommit.Body).Decode(&commitResp); err != nil {
		t.Fatalf("error decodificando respuesta de commit: %v", err)
	}
	if !commitResp.Success || commitResp.Image != "backup-image:v2.0" {
		t.Errorf("respuesta de commit inesperada: %+v", commitResp)
	}
}

func TestCreateContainerWithNetAdminAPI(t *testing.T) {
	cfg := &config.Config{
		SSHHost: "127.0.0.1",
		SSHPort: 22,
		SSHUser: "testuser",
		SSHRoot: t.TempDir(),
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	var lastCommand string
	mockRunner := func(ctx context.Context, vmName string, command string) (string, error) {
		lastCommand = command
		if strings.Contains(command, "docker run") {
			return "abcdef1234567890", nil
		}
		return "", nil
	}

	h := &Handler{
		cfg:       cfg,
		log:       log,
		vmSvc:     vm.NewService(cfg, log),
		tplSvc:    template.NewService(cfg, log),
		dockerSvc: docker.NewService(mockRunner, log),
	}

	// Caso 1: Crear contenedor con netAdmin: true
	createReq := docker.CreateContainerRequest{
		VMName:   "test-vm",
		Name:     "net-admin-box",
		Image:    "alpine:latest",
		NetAdmin: true,
	}
	bodyBytes, _ := json.Marshal(createReq)
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/containers", bytes.NewReader(bodyBytes))
	reqCreate.Header.Set("Content-Type", "application/json")
	rrCreate := httptest.NewRecorder()
	h.ServeHTTP(rrCreate, reqCreate)

	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("se esperaba status 201 al crear contenedor, obtenido: %d, body: %s", rrCreate.Code, rrCreate.Body.String())
	}
	if !strings.Contains(lastCommand, "--cap-add NET_ADMIN") {
		t.Errorf("el comando docker run generado por la API no contiene '--cap-add NET_ADMIN': %s", lastCommand)
	}

	// Caso 2: Crear contenedor con netAdmin: false (por defecto)
	createReqStd := docker.CreateContainerRequest{
		VMName:   "test-vm",
		Name:     "std-box",
		Image:    "alpine:latest",
		NetAdmin: false,
	}
	bodyBytesStd, _ := json.Marshal(createReqStd)
	reqCreateStd := httptest.NewRequest(http.MethodPost, "/api/containers", bytes.NewReader(bodyBytesStd))
	reqCreateStd.Header.Set("Content-Type", "application/json")
	rrCreateStd := httptest.NewRecorder()
	h.ServeHTTP(rrCreateStd, reqCreateStd)

	if rrCreateStd.Code != http.StatusCreated {
		t.Fatalf("se esperaba status 201 al crear contenedor estándar, obtenido: %d", rrCreateStd.Code)
	}
	if strings.Contains(lastCommand, "--cap-add NET_ADMIN") {
		t.Errorf("el comando docker run no debería incluir '--cap-add NET_ADMIN' cuando netAdmin es false: %s", lastCommand)
	}
}

func TestImageExportAPIEndpoints(t *testing.T) {
	cfg := &config.Config{
		SSHHost: "127.0.0.1",
		SSHPort: 22,
		SSHUser: "testuser",
		SSHRoot: t.TempDir(),
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockRunner := func(ctx context.Context, vmName string, command string) (string, error) {
		if strings.Contains(command, "docker save -o") {
			return "TAR_OK|/home/testuser/docker-image-exports/nginx_alpine_20261024_232000.tar|25165824|1792884000\n", nil
		}
		if strings.Contains(command, "find \"$EXPORT_DIR\"") {
			return "/home/testuser/docker-image-exports/nginx_alpine_20261024_232000.tar|25165824|1792884000\n", nil
		}
		return "", nil
	}

	h := &Handler{
		cfg:       cfg,
		log:       log,
		vmSvc:     vm.NewService(cfg, log),
		tplSvc:    template.NewService(cfg, log),
		dockerSvc: docker.NewService(mockRunner, log),
	}

	// 1. POST /api/images/export
	exportPayload := docker.ExportImageRequest{
		VMName:   "test-vm",
		ImageRef: "nginx:alpine",
	}
	exportBytes, _ := json.Marshal(exportPayload)
	reqExport := httptest.NewRequest(http.MethodPost, "/api/images/export", bytes.NewReader(exportBytes))
	reqExport.Header.Set("Content-Type", "application/json")
	rrExport := httptest.NewRecorder()
	h.ServeHTTP(rrExport, reqExport)

	if rrExport.Code != http.StatusCreated {
		t.Fatalf("se esperaba status 201 en export, obtenido: %d, body: %s", rrExport.Code, rrExport.Body.String())
	}
	var exportResp docker.ExportImageResponse
	if err := json.NewDecoder(rrExport.Body).Decode(&exportResp); err != nil {
		t.Fatalf("error decodificando respuesta de export: %v", err)
	}
	if !exportResp.Success || exportResp.FileName != "nginx_alpine_20261024_232000.tar" || exportResp.SizeBytes != 25165824 {
		t.Errorf("respuesta de export inesperada: %+v", exportResp)
	}

	// 2. GET /api/images/exports?vm=test-vm
	reqListTars := httptest.NewRequest(http.MethodGet, "/api/images/exports?vm=test-vm", nil)
	rrListTars := httptest.NewRecorder()
	h.ServeHTTP(rrListTars, reqListTars)

	if rrListTars.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200 en exports list, obtenido: %d, body: %s", rrListTars.Code, rrListTars.Body.String())
	}
	var tars []docker.ExportedTarFile
	if err := json.NewDecoder(rrListTars.Body).Decode(&tars); err != nil {
		t.Fatalf("error decodificando lista de tars: %v", err)
	}
	if len(tars) != 1 || tars[0].FileName != "nginx_alpine_20261024_232000.tar" || tars[0].Size != "24.0 MB" {
		t.Errorf("lista de tars inesperada: %+v", tars)
	}

	// 3. DELETE /api/images/exports?vm=test-vm&file=nginx_alpine_20261024_232000.tar
	reqDelTar := httptest.NewRequest(http.MethodDelete, "/api/images/exports?vm=test-vm&file=nginx_alpine_20261024_232000.tar", nil)
	rrDelTar := httptest.NewRecorder()
	h.ServeHTTP(rrDelTar, reqDelTar)

	if rrDelTar.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200 al eliminar TAR, obtenido: %d, body: %s", rrDelTar.Code, rrDelTar.Body.String())
	}
}

func TestDockerHubAPIEndpoints(t *testing.T) {
	cfg := &config.Config{
		SSHHost: "127.0.0.1",
		SSHPort: 22,
		SSHUser: "testuser",
		SSHRoot: t.TempDir(),
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockRunner := func(ctx context.Context, vmName string, command string) (string, error) {
		if strings.Contains(command, "sudo docker tag") {
			return "", nil
		}
		if strings.Contains(command, "sudo docker push") {
			return "v1: digest: sha256:abcdef size: 1024", nil
		}
		return "", nil
	}

	dockerSvc := docker.NewService(mockRunner, log)
	dockerSvc.SetCredentialsPath(filepath.Join(t.TempDir(), "dockerhub_credentials.json"))
	dockerSvc.SetStdinRunner(func(ctx context.Context, vmName string, command string, stdin string) (string, error) {
		return "Login Succeeded\n", nil
	})

	h := &Handler{
		cfg:       cfg,
		log:       log,
		vmSvc:     vm.NewService(cfg, log),
		tplSvc:    template.NewService(cfg, log),
		dockerSvc: dockerSvc,
	}

	// 1. GET /api/dockerhub/credentials (no configuradas aún)
	reqGet1 := httptest.NewRequest(http.MethodGet, "/api/dockerhub/credentials", nil)
	rrGet1 := httptest.NewRecorder()
	h.ServeHTTP(rrGet1, reqGet1)
	if rrGet1.Code != http.StatusOK {
		t.Fatalf("status inesperado en GET /api/dockerhub/credentials: %d", rrGet1.Code)
	}
	var st1 docker.DockerHubCredentialsStatus
	_ = json.NewDecoder(rrGet1.Body).Decode(&st1)
	if st1.Configured {
		t.Errorf("no se esperaba Configured=true inicialmente")
	}

	// 2. POST /api/dockerhub/credentials (guardar PAT y comprobar que nunca se devuelve el token en la respuesta)
	secretToken := "dckr_pat_SECRET_NEVER_EXPOSE"
	saveBody, _ := json.Marshal(map[string]string{
		"username":     "mi_usuario",
		"access_token": secretToken,
	})
	reqSave := httptest.NewRequest(http.MethodPost, "/api/dockerhub/credentials", bytes.NewReader(saveBody))
	reqSave.Header.Set("Content-Type", "application/json")
	rrSave := httptest.NewRecorder()
	h.ServeHTTP(rrSave, reqSave)
	if rrSave.Code != http.StatusOK {
		t.Fatalf("status inesperado guardando credenciales: %d, body: %s", rrSave.Code, rrSave.Body.String())
	}
	if strings.Contains(rrSave.Body.String(), secretToken) {
		t.Fatalf("la respuesta de la API NUNCA debe exponer el access_token: %s", rrSave.Body.String())
	}

	// 3. POST /api/images/push
	pushBody, _ := json.Marshal(docker.PushImageRequest{
		VMName:      "test-vm",
		SourceImage: "mi-api:1.0",
		Repository:  "mi-api",
		Tag:         "v1",
	})
	reqPush := httptest.NewRequest(http.MethodPost, "/api/images/push", bytes.NewReader(pushBody))
	reqPush.Header.Set("Content-Type", "application/json")
	rrPush := httptest.NewRecorder()
	h.ServeHTTP(rrPush, reqPush)
	if rrPush.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200 en push, obtenido: %d, body: %s", rrPush.Code, rrPush.Body.String())
	}
	var pushResp docker.PushImageResponse
	if err := json.NewDecoder(rrPush.Body).Decode(&pushResp); err != nil {
		t.Fatalf("error decodificando respuesta de push: %v", err)
	}
	if !pushResp.Success || pushResp.TargetImage != "mi_usuario/mi-api:v1" {
		t.Errorf("respuesta de push inesperada: %+v", pushResp)
	}
}

