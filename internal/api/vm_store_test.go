package api

import (
	"path/filepath"
	"testing"
)

func TestVMFileStoreRoundTrip(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "vms.json")
	store := newVMFileStore(storePath)

	vms := []VirtualMachine{{
		ID:        "vm-001",
		Name:      "demo-vm",
		OS:        "Windows Server 2022",
		ISO:       "windows_server_2022.iso",
		Cores:     4,
		MemoryMB:  4096,
		DiskGB:    80,
		CreatedAt: "2026-09-13T00:00:00Z",
	}}

	if err := store.Save(vms); err != nil {
		t.Fatalf("store.Save fallo: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load fallo: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("esperaba 1 VM, obtuve %d", len(loaded))
	}
	if loaded[0].Name != "demo-vm" {
		t.Fatalf("esperaba demo-vm, obtuve %s", loaded[0].Name)
	}
}
