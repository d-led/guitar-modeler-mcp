package mooer

import (
	"fmt"
	"os"
)

// MOFileSize is the size of a .mo single-preset file: a zeroed 0x200-byte
// header, the 0x200-byte preset record, then zero padding to 0x800. This is
// the GE150 Pro Li layout; the GE200 layout is handled separately (see
// ge200codec.go) but also totals 0x800 bytes.
const (
	MOFileSize     = 0x800 // 2048
	MOPresetOffset = 0x200 // 512
)

// MarshalMO wraps a preset in the GE150 Pro Li .mo container.
func MarshalMO(p Preset) []byte {
	buf := make([]byte, MOFileSize)
	copy(buf[MOPresetOffset:MOPresetOffset+PresetSize], p.Marshal())
	return buf
}

// UnmarshalMO parses a GE150 Pro Li .mo container into a preset.
func UnmarshalMO(data []byte) (Preset, error) {
	if len(data) < MOPresetOffset+PresetSize {
		return Preset{}, fmt.Errorf(".mo file is %d bytes, need %d", len(data), MOPresetOffset+PresetSize)
	}
	return Unmarshal(data[MOPresetOffset : MOPresetOffset+PresetSize])
}

// MarshalMOFor renders a preset in the target model's .mo layout.
func MarshalMOFor(m Model, p Preset) []byte {
	if isGE200(m) {
		return marshalGE200(p)
	}
	return MarshalMO(p)
}

// UnmarshalMOFor parses a .mo file in the target model's .mo layout.
func UnmarshalMOFor(m Model, data []byte) (Preset, error) {
	if isGE200(m) {
		return unmarshalGE200(data)
	}
	return UnmarshalMO(data)
}

// looksGE200 reports whether a .mo file is in the GE200 layout. GE200 exports
// carry the 0x08/0x01 magic bytes at the head; the GE150 Pro Li layout zeroes
// its whole 0x200-byte header.
func looksGE200(data []byte) bool {
	return len(data) == ge200FileSize && data[1] == 8 && data[8] == 1
}

// UnmarshalMOAny parses a .mo file, auto-detecting the layout.
func UnmarshalMOAny(data []byte) (Preset, error) {
	if looksGE200(data) {
		return unmarshalGE200(data)
	}
	return UnmarshalMO(data)
}

// WriteMOFile writes a preset to a .mo file in the target model's layout.
func WriteMOFile(m Model, path string, p Preset) error {
	return os.WriteFile(path, MarshalMOFor(m, p), 0o600)
}

// ReadMOFile reads a preset from a .mo file in the target model's layout.
func ReadMOFile(m Model, path string) (Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, err
	}
	return UnmarshalMOFor(m, data)
}

// ReadMOFileAny reads a preset from a .mo file, auto-detecting the layout.
func ReadMOFileAny(path string) (Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, err
	}
	return UnmarshalMOAny(data)
}
