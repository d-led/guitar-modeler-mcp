package gp200

import (
	"os"

	"github.com/d-led/guitar-modeler-mcp/internal/fileutil"
)

// WriteFile writes a preset to a .prst file (1224-byte user format), creating
// the directory when it does not exist yet.
func WriteFile(path string, p Preset) error {
	data, err := p.Marshal()
	if err != nil {
		return err
	}
	return fileutil.WriteFile(path, data)
}

// ReadFile reads a preset from a .prst file (1224-byte user or 1176-byte
// factory format).
func ReadFile(path string) (Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, err
	}
	return Unmarshal(data)
}
