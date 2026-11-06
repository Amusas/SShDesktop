package config

import (
	"path/filepath"
	"testing"
)

func TestConfigLoadAndSave(t *testing.T) {
	tempDir := t.TempDir()
	propPath := filepath.Join(tempDir, "config.properties")

	cfg := &Config{
		SSHHost:      "192.168.1.50",
		SSHPort:      2222,
		SSHUser:      "testuser",
		SSHKeyPath:   `C:\Users\testuser\.ssh\id_rsa`,
		SSHShell:     "cmd",
		SSHStartPath: `C:\Users\testuser`,
		ServerAddr:   ":9090",
		LogLevel:     "debug",
	}

	if err := SaveProperties(propPath, cfg); err != nil {
		t.Fatalf("SaveProperties fallo: %v", err)
	}

	vars, err := parseKeyValueFile(propPath)
	if err != nil {
		t.Fatalf("parseKeyValueFile fallo: %v", err)
	}

	if vars["ssh.host"] != "192.168.1.50" {
		t.Errorf("esperaba host 192.168.1.50, obtuve %s", vars["ssh.host"])
	}
	if vars["ssh.port"] != "2222" {
		t.Errorf("esperaba port 2222, obtuve %s", vars["ssh.port"])
	}
	if vars["ssh.shell"] != "cmd" {
		t.Errorf("esperaba shell cmd, obtuve %s", vars["ssh.shell"])
	}
	if vars["ssh.start_path"] != `C:\Users\testuser` {
		t.Errorf("esperaba start_path C:\\Users\\testuser, obtuve %s", vars["ssh.start_path"])
	}
	if vars["vm.nat_network_name"] != "vessel-net" {
		t.Errorf("esperaba vm.nat_network_name vessel-net, obtuve %s", vars["vm.nat_network_name"])
	}
	if vars["vm.nat_network_cidr"] != "10.250.0.0/24" {
		t.Errorf("esperaba vm.nat_network_cidr 10.250.0.0/24, obtuve %s", vars["vm.nat_network_cidr"])
	}
}
