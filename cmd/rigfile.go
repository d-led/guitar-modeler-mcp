package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

// readRigFile loads and parses a .rig file from disk.
func readRigFile(path string) (*rig.RigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file rig.RigFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse rig file %q: %w", path, err)
	}
	return &file, nil
}
