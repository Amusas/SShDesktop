package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeDaemonJSON_PreservesExistingFields(t *testing.T) {
	existing := `{
  "log-driver": "json-file",
  "live-restore": true,
  "registry-mirrors": ["http://old-mirror:5001"],
  "insecure-registries": ["10.250.0.2:5000"]
}`

	merged, changed, err := MergeDaemonJSON(
		existing,
		[]string{"http://10.250.0.10:5001", "http://old-mirror:5001"},
		[]string{"10.250.0.10:5000", "10.250.0.10:5001"},
	)
	if err != nil {
		t.Fatalf("error inesperado en MergeDaemonJSON: %v", err)
	}
	if !changed {
		t.Errorf("se esperaba changed=true al agregar nuevos mirrors/insecure-registries")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
		t.Fatalf("JSON resultante inválido: %v", err)
	}

	if parsed["log-driver"] != "json-file" || parsed["live-restore"] != true {
		t.Errorf("se perdieron propiedades existentes de daemon.json: %+v", parsed)
	}

	mirrorsRaw, ok := parsed["registry-mirrors"].([]interface{})
	if !ok || len(mirrorsRaw) != 2 {
		t.Errorf("se esperaban 2 registry-mirrors sin duplicados, obtenido: %+v", parsed["registry-mirrors"])
	}

	insecureRaw, ok := parsed["insecure-registries"].([]interface{})
	if !ok || len(insecureRaw) != 3 {
		t.Errorf("se esperaban 3 insecure-registries sin duplicados, obtenido: %+v", parsed["insecure-registries"])
	}

	// Segunda llamada con los mismos valores debe reportar changed=false
	_, changedAgain, err := MergeDaemonJSON(
		merged,
		[]string{"http://10.250.0.10:5001"},
		[]string{"10.250.0.10:5000"},
	)
	if err != nil || changedAgain {
		t.Errorf("segunda llamada idempotente debía devolver changed=false, obtenido changed=%v err=%v", changedAgain, err)
	}
}

func TestDeployLocalAndCacheRegistry_IndependentAndIdempotent(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "registry.json")
	store := NewRegistryStore(tmpFile)

	provider := newMockSwarmVMProvider(
		SwarmVMInfo{ID: "vm-1", Name: "nodo-reg", State: "running", PrivateIP: "10.250.0.10", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
		SwarmVMInfo{ID: "vm-2", Name: "nodo-worker", State: "running", PrivateIP: "10.250.0.11", NetworkName: "vessel-net", NetworkCIDR: "10.250.0.0/24"},
	)

	var commands []string
	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		commands = append(commands, vmName+": "+cmd)
		if strings.Contains(cmd, "cat /etc/docker/daemon.json") {
			return `{"log-driver":"json-file"}`, nil
		}
		if strings.Contains(cmd, "DEPLOY_OK") {
			return "DEPLOY_OK|1\n", nil
		}
		if strings.Contains(cmd, "PROBE|") {
			return "PROBE|abcdef1234567890|running|unless-stopped|200\n", nil
		}
		return "", nil
	}

	svc := NewRegistryService(runner, provider, store, nil)

	// 1. Desplegar Registry local escribible en puerto 5000
	localSt, err := svc.DeployInstance(context.Background(), RegistryDeployRequest{
		Target: RegistryTargetLocal,
		VMName: "nodo-reg",
		Port:   5000,
	})
	if err != nil {
		t.Fatalf("error desplegando Registry local: %v", err)
	}
	if !localSt.Configured || !localSt.Healthy || localSt.EndpointHost != "10.250.0.10:5000" || localSt.ContainerName != DefaultLocalRegistryContainer {
		t.Errorf("estado inesperado de Registry local: %+v", localSt)
	}

	// 2. Intentar desplegar Cache en el mismo puerto 5000 de la misma VM debe fallar
	if _, err := svc.DeployInstance(context.Background(), RegistryDeployRequest{
		Target: RegistryTargetCache,
		VMName: "nodo-reg",
		Port:   5000,
	}); err == nil {
		t.Fatalf("debía rechazarse colisión de puerto 5000 entre Registry local y Pull-through Cache en la misma VM")
	}

	// 3. Desplegar Pull-through Cache independiente en puerto 5001
	cacheSt, err := svc.DeployInstance(context.Background(), RegistryDeployRequest{
		Target: RegistryTargetCache,
		VMName: "nodo-reg",
		Port:   5001,
	})
	if err != nil {
		t.Fatalf("error desplegando Pull-through Cache: %v", err)
	}
	if !cacheSt.Configured || !cacheSt.Healthy || cacheSt.EndpointURL != "http://10.250.0.10:5001" || cacheSt.ContainerName != DefaultCacheRegistryContainer || cacheSt.RemoteURL != DefaultCacheRemoteURL {
		t.Errorf("estado inesperado de Pull-through Cache: %+v", cacheSt)
	}
}

func TestListAndDeleteRegistryImages_SharedDigestProtection(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "registry.json")
	store := NewRegistryStore(tmpFile)
	_ = store.SetInstance(RegistryInstanceConfig{
		Target:        RegistryTargetLocal,
		VMID:          "vm-1",
		VMName:        "nodo-reg",
		HostIP:        "10.250.0.10",
		Port:          5000,
		ContainerPort: 5000,
		ContainerName: DefaultLocalRegistryContainer,
		VolumeName:    DefaultLocalRegistryVolume,
		Image:         DefaultRegistryImage,
	})

	provider := newMockSwarmVMProvider(
		SwarmVMInfo{ID: "vm-1", Name: "nodo-reg", State: "running", PrivateIP: "10.250.0.10"},
	)

	const sharedDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	const uniqueDigest = "sha256:2222222222222222222222222222222222222222222222222222222222222222"

	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		switch {
		case strings.Contains(cmd, "/v2/_catalog"):
			return `{"repositories":["equipo/api","mysql"]}`, nil
		case strings.Contains(cmd, "/v2/mysql/tags/list"):
			return `{"name":"mysql","tags":["8.4","latest"]}`, nil
		case strings.Contains(cmd, "/v2/equipo/api/tags/list"):
			return `{"name":"equipo/api","tags":["1.0"]}`, nil
		case strings.Contains(cmd, "/v2/mysql/manifests/"):
			return fmt.Sprintf("HTTP/1.1 200 OK\r\nDocker-Content-Digest: %s\r\nContent-Type: application/vnd.oci.image.manifest.v1+json\r\n\r\n{\"schemaVersion\":2,\"config\":{\"size\":1024},\"layers\":[{\"size\":10485760}]}", sharedDigest), nil
		case strings.Contains(cmd, "/v2/equipo/api/manifests/1.0"):
			return fmt.Sprintf("HTTP/1.1 200 OK\r\nDocker-Content-Digest: %s\r\nContent-Type: application/vnd.docker.distribution.manifest.v2+json\r\n\r\n{\"schemaVersion\":2,\"config\":{\"size\":512},\"layers\":[{\"size\":2097152}]}", uniqueDigest), nil
		case strings.Contains(cmd, "DELETE") && strings.Contains(cmd, uniqueDigest):
			return "DELETE_OK\n", nil
		}
		return "", nil
	}

	svc := NewRegistryService(runner, provider, store, nil)

	res, err := svc.ListRegistryImages(context.Background(), RegistryTargetLocal)
	if err != nil {
		t.Fatalf("error listando imágenes del Registry: %v", err)
	}
	if len(res.Images) != 3 {
		t.Fatalf("se esperaban 3 imágenes en el catálogo, obtenido: %d", len(res.Images))
	}

	// Verificar que mysql:8.4 y mysql:latest detecten que comparten el mismo digest
	if err := svc.DeleteRegistryImage(context.Background(), "mysql", "8.4", false); err == nil {
		t.Fatalf("debía bloquearse la eliminación sin force de mysql:8.4 porque comparte digest con mysql:latest")
	}

	// Eliminar equipo/api:1.0 (que tiene digest único) debe funcionar sin force
	if err := svc.DeleteRegistryImage(context.Background(), "equipo/api", "1.0", false); err != nil {
		t.Fatalf("error inesperado eliminando imagen con digest único: %v", err)
	}
}

func TestSmartPullImage_LocalPriorityAndCacheFallback(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "registry.json")
	store := NewRegistryStore(tmpFile)
	_ = store.SetInstance(RegistryInstanceConfig{
		Target:        RegistryTargetLocal,
		VMID:          "vm-reg",
		VMName:        "nodo-reg",
		HostIP:        "10.250.0.10",
		Port:          5000,
		ContainerPort: 5000,
		ContainerName: DefaultLocalRegistryContainer,
		VolumeName:    DefaultLocalRegistryVolume,
		Image:         DefaultRegistryImage,
	})
	_ = store.SetInstance(RegistryInstanceConfig{
		Target:        RegistryTargetCache,
		VMID:          "vm-reg",
		VMName:        "nodo-reg",
		HostIP:        "10.250.0.10",
		Port:          5001,
		ContainerPort: 5000,
		ContainerName: DefaultCacheRegistryContainer,
		VolumeName:    DefaultCacheRegistryVolume,
		Image:         DefaultRegistryImage,
		RemoteURL:     DefaultCacheRemoteURL,
	})

	provider := newMockSwarmVMProvider(
		SwarmVMInfo{ID: "vm-reg", Name: "nodo-reg", State: "running", PrivateIP: "10.250.0.10"},
		SwarmVMInfo{ID: "vm-wrk", Name: "nodo-1", State: "running", PrivateIP: "10.250.0.11"},
	)

	localHasMysql := true
	localHealthy := true
	cacheHasRedis := false
	var pulledRefs []string

	runner := func(_ context.Context, vmName, cmd string) (string, error) {
		if strings.Contains(cmd, "cat /etc/docker/daemon.json") {
			return `{"registry-mirrors":["http://10.250.0.10:5001"],"insecure-registries":["10.250.0.10:5000","10.250.0.10:5001"]}`, nil
		}
		if strings.Contains(cmd, "http://127.0.0.1:5000/v2/\"") {
			if !localHealthy {
				return "000", fmt.Errorf("connection refused")
			}
			return "200", nil
		}
		if strings.Contains(cmd, "http://127.0.0.1:5001/v2/\"") {
			return "200", nil
		}
		if strings.Contains(cmd, "http://127.0.0.1:5000/v2/mysql/manifests/8.4") {
			if localHasMysql {
				return "200", nil
			}
			return "404", nil
		}
		if strings.Contains(cmd, "http://127.0.0.1:5000/v2/") && strings.Contains(cmd, "/manifests/") {
			return "404", nil
		}
		if strings.Contains(cmd, "http://127.0.0.1:5001/v2/") && strings.Contains(cmd, "redis/manifests/7.2") {
			if cacheHasRedis {
				return "200", nil
			}
			return "404", nil
		}
		if strings.Contains(cmd, "sudo docker pull ") {
			pulledRefs = append(pulledRefs, cmd)
			if strings.Contains(cmd, "redis:7.2") {
				cacheHasRedis = true
			}
			return "Status: Downloaded newer image\n", nil
		}
		return "", nil
	}

	svc := NewRegistryService(runner, provider, store, nil)

	// Escenario 1 (Prioridad 1): mysql:8.4 está en el Registry local -> debe descargarse desde 10.250.0.10:5000/mysql:8.4
	res1, err := svc.SmartPullImage(context.Background(), SmartPullRequest{
		VMName: "nodo-1",
		Image:  "mysql:8.4",
	})
	if err != nil {
		t.Fatalf("error inesperado en SmartPull Prioridad 1: %v", err)
	}
	if !res1.FoundInLocalRegistry || res1.Source != "local-registry" || res1.PulledReference != "10.250.0.10:5000/mysql:8.4" {
		t.Errorf("se esperaba descarga desde Registry local (10.250.0.10:5000/mysql:8.4), obtenido: %+v", res1)
	}

	// Escenario 2 (Prioridad 2 - Cache Miss inicial): redis:7.2 NO está en el Registry local -> descarga vía Docker Hub / Cache y queda en caché
	res2, err := svc.SmartPullImage(context.Background(), SmartPullRequest{
		VMName: "nodo-1",
		Image:  "redis:7.2",
	})
	if err != nil {
		t.Fatalf("error inesperado en SmartPull Prioridad 2 (Cache Miss): %v", err)
	}
	if res2.FoundInLocalRegistry || !res2.UsedPullThroughCache || res2.CacheHit || res2.Source != "dockerhub-cache-miss" {
		t.Errorf("se esperaba dockerhub-cache-miss en primera descarga de redis:7.2, obtenido: %+v", res2)
	}

	// Escenario 3 (Prioridad 2 - Cache Hit en segunda descarga): redis:7.2 ya está en la caché Pull-through
	res3, err := svc.SmartPullImage(context.Background(), SmartPullRequest{
		VMName: "nodo-1",
		Image:  "redis:7.2",
	})
	if err != nil {
		t.Fatalf("error inesperado en SmartPull Prioridad 2 (Cache Hit): %v", err)
	}
	if !res3.UsedPullThroughCache || !res3.CacheHit || res3.Source != "dockerhub-cache-hit" {
		t.Errorf("se esperaba dockerhub-cache-hit en segunda descarga de redis:7.2, obtenido: %+v", res3)
	}

	// Escenario 4: Si el Registry local está caído y AllowFallback=false, debe informar el fallo y ofrecer alternativa
	localHealthy = false
	res4, err := svc.SmartPullImage(context.Background(), SmartPullRequest{
		VMName:        "nodo-1",
		Image:         "redis:7.2",
		AllowFallback: false,
	})
	if err == nil || res4 == nil || !res4.LocalRegistryUnreachable || !res4.CanFallbackToDockerHub {
		t.Errorf("se esperaba LocalRegistryUnreachable=true y CanFallbackToDockerHub=true cuando el Registry local no responde, obtenido: res=%+v err=%v", res4, err)
	}

	// Y si el usuario confirma AllowFallback=true, continúa la descarga desde Docker Hub / Caché
	res5, err := svc.SmartPullImage(context.Background(), SmartPullRequest{
		VMName:        "nodo-1",
		Image:         "redis:7.2",
		AllowFallback: true,
	})
	if err != nil || !res5.Success {
		t.Errorf("se esperaba éxito al usar AllowFallback=true hacia Docker Hub: res=%+v err=%v", res5, err)
	}
}
