package gp200

import "os"

// WriteFile writes a preset to a .prst file (1224-byte user format).
func WriteFile(path string, p Preset) error {
	data, err := p.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
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
