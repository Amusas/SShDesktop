package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/feliren/ssh-web/internal/config"
)

type hostProfile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SSHHost      string `json:"sshHost"`
	SSHPort      int    `json:"sshPort"`
	SSHUser      string `json:"sshUser"`
	SSHShell     string `json:"sshShell"`
	SSHStartPath string `json:"sshStartPath"`
	SSHKeyPath   string `json:"sshKeyPath"`
}

type hostsFile struct {
	ActiveHostID string        `json:"activeHostId"`
	Hosts        []hostProfile `json:"hosts"`
}

func newHostProfile(cfg *config.Config) hostProfile {
	return hostProfile{
		ID:           "env-default",
		Name:         hostDisplayName(cfg),
		SSHHost:      cfg.SSHHost,
		SSHPort:      cfg.SSHPort,
		SSHUser:      cfg.SSHUser,
		SSHShell:     cfg.SSHShell,
		SSHStartPath: cfg.SSHStartPath,
		SSHKeyPath:   cfg.SSHKeyPath,
	}
}

func hostDisplayName(cfg *config.Config) string {
	if cfg.SSHHost != "" && cfg.SSHUser != "" {
		return fmt.Sprintf("%s@%s", cfg.SSHUser, cfg.SSHHost)
	}
	if cfg.SSHHost != "" {
		return cfg.SSHHost
	}
	return "Host del .env"
}

func loadHosts(path string, cfg *config.Config) (hostsFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return hostsFile{ActiveHostID: "env-default", Hosts: []hostProfile{newHostProfile(cfg)}}, nil
	}
	if err != nil {
		return hostsFile{}, err
	}

	var file hostsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return hostsFile{}, fmt.Errorf("leyendo hosts: %w", err)
	}
	if len(file.Hosts) == 0 {
		file = hostsFile{ActiveHostID: "env-default", Hosts: []hostProfile{newHostProfile(cfg)}}
	}
	if file.ActiveHostID == "" || findHost(file.Hosts, file.ActiveHostID) == nil {
		file.ActiveHostID = file.Hosts[0].ID
	}
	return file, nil
}

func saveHosts(path string, file hostsFile) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func findHost(hosts []hostProfile, id string) *hostProfile {
	for index := range hosts {
		if hosts[index].ID == id {
			return &hosts[index]
		}
	}
	return nil
}

func normalizeHostProfile(profile hostProfile) (hostProfile, error) {
	profile.ID = strings.TrimSpace(profile.ID)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.SSHHost = strings.TrimSpace(profile.SSHHost)
	profile.SSHUser = strings.TrimSpace(profile.SSHUser)
	profile.SSHShell = strings.ToLower(strings.TrimSpace(profile.SSHShell))
	profile.SSHStartPath = strings.TrimSpace(profile.SSHStartPath)
	profile.SSHKeyPath = strings.TrimSpace(profile.SSHKeyPath)
	if profile.Name == "" {
		profile.Name = hostDisplayName(&config.Config{SSHHost: profile.SSHHost, SSHUser: profile.SSHUser})
	}
	if profile.SSHHost == "" {
		return profile, fmt.Errorf("el host o IP es obligatorio")
	}
	if profile.SSHUser == "" {
		return profile, fmt.Errorf("el usuario SSH es obligatorio")
	}
	if profile.SSHPort <= 0 {
		profile.SSHPort = 22
	}
	if profile.SSHShell != "cmd" && profile.SSHShell != "powershell" {
		profile.SSHShell = "powershell"
	}
	return profile, nil
}

func newHostID() string {
	return fmt.Sprintf("host-%d", time.Now().UnixNano())
}

func applyHostToConfig(profile hostProfile, cfg *config.Config) {
	cfg.SSHHost = profile.SSHHost
	cfg.SSHPort = profile.SSHPort
	cfg.SSHUser = profile.SSHUser
	cfg.SSHShell = profile.SSHShell
	cfg.SSHStartPath = profile.SSHStartPath
	cfg.SSHKeyPath = profile.SSHKeyPath
}
