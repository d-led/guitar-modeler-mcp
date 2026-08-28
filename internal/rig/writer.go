package rig

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Name returns the rig's display name from the Patch's Rig node.
func (f *RigFile) Name() string {
	c, err := f.Decode()
	if err != nil {
		return "rig"
	}
	rig, ok := c.Data.Patch.Children["Rig"]
	if !ok {
		return "rig"
	}
	item, ok := rig.Children["PresetName"]
	if !ok || item.Str == nil {
		return "rig"
	}
	return *item.Str
}

// Write persists the rig file into dir using the rig name as the file name.
// It returns the absolute path of the written file.
func (f *RigFile) Write(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := sanitizeFileName(f.Name())
	if name == "" {
		name = "rig"
	}
	path := filepath.Join(dir, name+".rig")
	data, err := f.Marshal()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// sanitizeFileName keeps only filesystem-safe characters.
func sanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == ' ', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return strings.TrimRight(strings.TrimSpace(b.String()), ". ")
}
