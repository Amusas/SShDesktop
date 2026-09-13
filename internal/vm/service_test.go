package vm

import "testing"

func TestFindOSByISO(t *testing.T) {
	got, err := FindOSByISO("ubuntu-24.04.5-live-server-amd64.iso")
	if err != nil {
		t.Fatalf("esperaba ISO conocida: %v", err)
	}
	if got.Name != "Ubuntu 24.04" {
		t.Fatalf("OS soportada derivada incorrecta: %q", got.Name)
	}
}
