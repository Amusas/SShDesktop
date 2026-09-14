package vm

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFindOSByISO(t *testing.T) {
	got, err := FindOSByISO("ubuntu-24.04.5-live-server-amd64.iso")
	if err != nil {
		t.Fatalf("esperaba ISO conocida: %v", err)
	}
	if !strings.HasPrefix(got.Name, "Ubuntu 24.04") {
		t.Fatalf("OS soportada derivada incorrecta: %q", got.Name)
	}
}

func TestVMStore(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "vms.json")
	store := NewStore(storePath)

	rec := VMRecord{
		Name:     "Test-VM-1",
		OS:       "Ubuntu 24.04 LTS",
		ISO:      "ubuntu-24.04.5-live-server-amd64.iso",
		Cores:    4,
		MemoryMB: 4096,
		DiskGB:   60,
		State:    "Creada",
	}

	added, err := store.Add(rec)
	if err != nil {
		t.Fatalf("error agregando VM: %v", err)
	}
	if added.Name != "Test-VM-1" || added.Cores != 4 || added.MemoryMB != 4096 || added.DiskGB != 60 {
		t.Fatalf("valores de VM incorrectos: %+v", added)
	}

	found, exists, err := store.Find("Test-VM-1")
	if err != nil || !exists {
		t.Fatalf("error buscando VM: %v", err)
	}
	if found.DiskGB != 60 {
		t.Fatalf("esperaba 60 GB de disco, obtuve %d", found.DiskGB)
	}

	list, err := store.Load()
	if err != nil || len(list) != 1 {
		t.Fatalf("error listando VMs: %v", err)
	}

	if err := store.Delete("Test-VM-1"); err != nil {
		t.Fatalf("error eliminando VM: %v", err)
	}

	listAfter, _ := store.Load()
	if len(listAfter) != 0 {
		t.Fatalf("esperaba 0 VMs, obtuve %d", len(listAfter))
	}
}
