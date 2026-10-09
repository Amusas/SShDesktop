package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
		Dockerfile: "FROM alpine:latest\nCMD [\"echo\", \"ok\"]",
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
