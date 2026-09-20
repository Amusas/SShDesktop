package template

import (
	"testing"
)

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name    string
		tpl     Template
		wantErr bool
	}{
		{
			name: "Plantilla valida",
			tpl: Template{
				ID:      "ubuntu-2404",
				Name:    "Ubuntu 24.04 LTS",
				OSType:  "Ubuntu_64",
				Disk:    "base.vdi",
				SSHUser: "bootstrap",
				SSHPort: 22,
			},
			wantErr: false,
		},
		{
			name: "ID con caracteres invalidos",
			tpl: Template{
				ID:     "ubuntu/2404",
				Name:   "Ubuntu 24.04",
				OSType: "Ubuntu_64",
				Disk:   "base.vdi",
			},
			wantErr: true,
		},
		{
			name: "Path traversal en campo Disk",
			tpl: Template{
				ID:     "debian-13",
				Name:   "Debian 13",
				OSType: "Debian_64",
				Disk:   "../../etc/passwd",
			},
			wantErr: true,
		},
		{
			name: "Ruta absoluta en campo Disk",
			tpl: Template{
				ID:     "debian-13",
				Name:   "Debian 13",
				OSType: "Debian_64",
				Disk:   `C:\otro\disco.vdi`,
			},
			wantErr: true,
		},
		{
			name: "Nombre vacio",
			tpl: Template{
				ID:     "debian-13",
				Name:   "",
				OSType: "Debian_64",
				Disk:   "base.vdi",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.tpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
