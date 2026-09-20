package keys

import (
	"strings"
	"testing"
)

func TestGenerateVMUser(t *testing.T) {
	tests := []struct {
		vmName   string
		prefix   string
		minLen   int
	}{
		{
			vmName: "otraMV",
			prefix: "vm_otramv_",
			minLen: 12,
		},
		{
			vmName: "Servidor-Web-Prod",
			prefix: "vm_servidorwebprod_",
			minLen: 15,
		},
		{
			vmName: "VM_Con_Caracteres!@#$",
			prefix: "vm_vmconcaracteres_",
			minLen: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.vmName, func(t *testing.T) {
			user := GenerateVMUser(tt.vmName)
			if !strings.HasPrefix(user, tt.prefix) {
				t.Errorf("GenerateVMUser(%q) = %q, want prefix %q", tt.vmName, user, tt.prefix)
			}
			if len(user) < tt.minLen {
				t.Errorf("GenerateVMUser(%q) = %q, length too short (%d < %d)", tt.vmName, user, len(user), tt.minLen)
			}
		})
	}
}
