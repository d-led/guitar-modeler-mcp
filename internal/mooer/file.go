package mooer

import (
	"fmt"
	"os"
)

// MOFileSize is the size of a .mo single-preset file: a zeroed 0x200-byte
// header, the 0x200-byte preset record, then zero padding to 0x800.
const (
	MOFileSize     = 0x800 // 2048
	MOPresetOffset = 0x200 // 512
)

// MarshalMO wraps a preset in the .mo container.
func MarshalMO(p Preset) []byte {
	buf := make([]byte, MOFileSize)
	copy(buf[MOPresetOffset:MOPresetOffset+PresetSize], p.Marshal())
	return buf
}

// UnmarshalMO parses a .mo container into a preset.
func UnmarshalMO(data []byte) (Preset, error) {
	if len(data) < MOPresetOffset+PresetSize {
		return Preset{}, fmt.Errorf(".mo file is %d bytes, need %d", len(data), MOPresetOffset+PresetSize)
	}
	return Unmarshal(data[MOPresetOffset : MOPresetOffset+PresetSize])
}

// WriteMOFile writes a preset to a .mo file.
func WriteMOFile(path string, p Preset) error {
	return os.WriteFile(path, MarshalMO(p), 0o644)
}

// ReadMOFile reads a preset from a .mo file.
func ReadMOFile(path string) (Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, err
	}
	return UnmarshalMO(data)
}
