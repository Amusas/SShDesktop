package keys

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestKeysGenerationAndPersistence(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "keys.json")
	store := NewStore(storePath)

	key, err := store.GenerateKeyPairForVM("Ubuntu-Server-Test")
	if err != nil {
		t.Fatalf("Error al generar llave: %v", err)
	}

	if key.VMName != "Ubuntu-Server-Test" {
		t.Errorf("Esperaba VMName Ubuntu-Server-Test, obtuve %s", key.VMName)
	}
	if !strings.HasPrefix(key.PrivateKey, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Errorf("Clave privada no tiene encabezado PEM RSA esperado: %s", key.PrivateKey)
	}
	if !strings.HasPrefix(key.PublicKey, "ssh-rsa ") {
		t.Errorf("Clave publica no tiene formato OpenSSH esperado: %s", key.PublicKey)
	}
	if !strings.HasPrefix(key.Fingerprint, "SHA256:") {
		t.Errorf("Huella digital no tiene prefijo SHA256: %s", key.Fingerprint)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("Error al listar llaves: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Esperaba 1 llave, obtuve %d", len(list))
	}

	retrieved, err := store.Get(key.ID)
	if err != nil {
		t.Fatalf("Error al obtener llave: %v", err)
	}
	if retrieved.ID != key.ID {
		t.Errorf("ID mismatch: %s vs %s", retrieved.ID, key.ID)
	}

	if err := store.Delete(key.ID); err != nil {
		t.Fatalf("Error al eliminar llave: %v", err)
	}

	listAfter, _ := store.List()
	if len(listAfter) != 0 {
		t.Errorf("Esperaba 0 llaves despues de eliminar, obtuve %d", len(listAfter))
	}
}
