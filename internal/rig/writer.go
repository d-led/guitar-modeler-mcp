package rig

import (
	"os"
	"path/filepath"
	"strings"
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

// NameLimit is the maximum length of a Gigboard preset name that displays in
// full. The device ellipsizes longer names: only about "SATCH - ALWAYS WITH M"
// (21 characters) fits before the ellipsis, and the longest factory preset is
// 22 characters ("148 I'LL BE WATCHING 2"). We cap at 21 so a generated name
// always displays in full on the hardware.
const NameLimit = 21

// StoredName returns the name as the device displays it: uppercased (the
// Gigboard renders preset names in all caps), truncated to NameLimit
// characters, with trailing separators (space, hyphen, underscore, dot)
// removed so a mid-word cut does not dangle. The second result reports
// truncation.
func StoredName(name string) (string, bool) {
	name = strings.ToUpper(strings.TrimSpace(name))
	if len(name) <= NameLimit {
		return name, false
	}
	return strings.TrimRight(name[:NameLimit], " -_."), true
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

// sanitizeFileName keeps only printable ASCII filesystem-safe characters;
// anything else (accented letters, emoji, control characters) becomes an
// underscore, so the file name is portable across machines and file systems.
func sanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'),
			r == ' ', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return strings.TrimRight(strings.TrimSpace(b.String()), ". ")
}
