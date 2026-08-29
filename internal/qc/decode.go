package qc

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// DecodePreset decrypts a Quad Cortex preset file and decodes the result as a
// BinaryPreset protobuf. The serial is the unit's nine-character serial, or
// empty for cloud/update files.
func DecodePreset(serial string, data []byte) (*BinaryPreset, error) {
	plain, err := Decrypt(serial, data)
	if err != nil {
		return nil, err
	}
	var preset BinaryPreset
	if err := proto.Unmarshal(plain, &preset); err != nil {
		return nil, fmt.Errorf("qc: decode BinaryPreset: %w", err)
	}
	return &preset, nil
}

// EncodePreset marshals a BinaryPreset and encrypts it for the given serial,
// producing a file the device can load.
func EncodePreset(serial string, preset *BinaryPreset) ([]byte, error) {
	plain, err := proto.Marshal(preset)
	if err != nil {
		return nil, fmt.Errorf("qc: encode BinaryPreset: %w", err)
	}
	return Encrypt(serial, plain)
}
