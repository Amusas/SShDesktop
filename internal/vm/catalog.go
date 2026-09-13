package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OSCatalogEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	VirtualBoxOSType string `json:"virtualBoxOsType"`
	ISOFilename      string `json:"isoFilename"`
	ISOURL           string `json:"isoUrl"`
}

type isoFileEntry struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	OS          string `json:"os"`
	VBoxOSType  string `json:"vboxOsType"`
}

type isoFilePayload struct {
	ISOs []isoFileEntry `json:"isos"`
}

func Catalog() []OSCatalogEntry {
	return []OSCatalogEntry{}
}

func FindOS(id string) (OSCatalogEntry, error) {
	return FindOSByISO(id)
}

func FindOSByISO(iso string) (OSCatalogEntry, error) {
	iso = strings.ToLower(strings.TrimSpace(iso))
	payload, err := readISOsJSON()
	if err != nil {
		return OSCatalogEntry{}, err
	}
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
	return OSCatalogEntry{}, fmt.Errorf("ISO no soportada o no encontrada en el catalogo")
}

func readISOsJSON() (isoFilePayload, error) {
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
		var payload isoFilePayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return isoFilePayload{}, fmt.Errorf("parseando isos.json: %w", err)
		}
		return payload, nil
	}
	return isoFilePayload{}, fmt.Errorf("leyendo isos.json: %w", lastErr)
}
