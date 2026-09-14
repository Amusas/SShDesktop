package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type OSCatalogEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	VirtualBoxOSType string `json:"virtualBoxOsType"`
	ISOFilename      string `json:"isoFilename"`
	ISOURL           string `json:"isoUrl"`
}

type ISOFileEntry struct {
	Name       string `json:"name"`
	URL        string `json:"url,omitempty"`
	OS         string `json:"os,omitempty"`
	VBoxOSType string `json:"vboxOsType,omitempty"`
}

type ISOFilePayload struct {
	ISOs []ISOFileEntry `json:"isos"`
}

var isoMu sync.Mutex

func Catalog() []OSCatalogEntry {
	payload, err := readISOsJSON()
	if err != nil {
		return []OSCatalogEntry{}
	}
	entries := make([]OSCatalogEntry, 0, len(payload.ISOs))
	for _, item := range payload.ISOs {
		entries = append(entries, OSCatalogEntry{
			ID:               strings.ToLower(strings.ReplaceAll(item.OS, " ", "-")),
			Name:             item.OS,
			VirtualBoxOSType: item.VBoxOSType,
			ISOFilename:      item.Name,
			ISOURL:           item.URL,
		})
	}
	return entries
}

func FindOS(id string) (OSCatalogEntry, error) {
	return FindOSByISO(id)
}

func FindOSByISO(iso string) (OSCatalogEntry, error) {
	iso = strings.ToLower(strings.TrimSpace(iso))
	payload, err := readISOsJSON()
	if err == nil {
		for _, entry := range payload.ISOs {
			if strings.EqualFold(strings.ToLower(entry.Name), iso) {
				return OSCatalogEntry{
					ID:               strings.ToLower(strings.ReplaceAll(entry.OS, " ", "-")),
					Name:             entry.OS,
					VirtualBoxOSType: entry.VBoxOSType,
					ISOFilename:      entry.Name,
					ISOURL:           entry.URL,
				}, nil
			}
		}
	}

	// Auto-detección inteligente para ISOs personalizadas
	vboxType := "Linux_64"
	osName := "Linux"
	if strings.Contains(iso, "win") || strings.Contains(iso, "windows") {
		vboxType = "Windows10_64"
		osName = "Windows"
	} else if strings.Contains(iso, "ubuntu") {
		vboxType = "Ubuntu_64"
		osName = "Ubuntu"
	} else if strings.Contains(iso, "debian") {
		vboxType = "Debian_64"
		osName = "Debian"
	} else if strings.Contains(iso, "centos") || strings.Contains(iso, "rhel") || strings.Contains(iso, "redhat") {
		vboxType = "RedHat_64"
		osName = "RedHat"
	} else if strings.Contains(iso, "arch") {
		vboxType = "ArchLinux_64"
		osName = "Arch Linux"
	}

	return OSCatalogEntry{
		ID:               strings.ToLower(osName),
		Name:             osName,
		VirtualBoxOSType: vboxType,
		ISOFilename:      iso,
	}, nil
}

func SaveISO(entry ISOFileEntry) error {
	isoMu.Lock()
	defer isoMu.Unlock()

	payload, _ := readISOsJSON()
	for _, item := range payload.ISOs {
		if strings.EqualFold(item.Name, entry.Name) {
			return nil // ya existe
		}
	}

	if entry.OS == "" {
		if strings.Contains(strings.ToLower(entry.Name), "win") {
			entry.OS = "Windows"
			entry.VBoxOSType = "Windows10_64"
		} else if strings.Contains(strings.ToLower(entry.Name), "ubuntu") {
			entry.OS = "Ubuntu"
			entry.VBoxOSType = "Ubuntu_64"
		} else if strings.Contains(strings.ToLower(entry.Name), "debian") {
			entry.OS = "Debian"
			entry.VBoxOSType = "Debian_64"
		} else {
			entry.OS = "Linux"
			entry.VBoxOSType = "Linux_64"
		}
	}

	payload.ISOs = append(payload.ISOs, entry)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("isos.json", data, 0644)
}

func readISOsJSON() (ISOFilePayload, error) {
	candidates := []string{
		filepath.Join(".", "isos.json"),
		filepath.Join("..", "isos.json"),
		filepath.Join("..", "..", "isos.json"),
	}
	var lastErr error
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			lastErr = err
			continue
		}
		var payload ISOFilePayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return ISOFilePayload{}, fmt.Errorf("parseando isos.json: %w", err)
		}
		return payload, nil
	}
	return ISOFilePayload{}, fmt.Errorf("leyendo isos.json: %w", lastErr)
}
