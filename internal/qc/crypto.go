package qc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1" // #nosec G505 -- SHA-1 reproduces the Quad Cortex EVP_BytesToKey KDF
	"fmt"
)

// keyMaterial is the symmetric key material baked into the Quad Cortex
// firmware (/usr/lib/libzc.so, SetupKeys). It is public: the OpenCortex
// project extracted it and published it in File-decryption/qc_decrypt.c.
// Together with the unit's serial number (from /etc/qc_sn) it feeds the key
// derivation below; an empty serial is used for cloud/update files.
var keyMaterial = []byte{
	0x13, 0x27, 0x3f, 0x42,
	0xa5, 0xb6, 0x79, 0xe8,
	0x20, 0x31, 0xc4, 0xf5,
	0x16, 0x17, 0x88, 0x2f,
	0x43, 0xa4, 0x55, 0x69,
	0x77, 0xb8, 0xe2, 0x83,
	0x04, 0x05, 0x60, 0x70,
	0x80, 0x02, 0x03, 0x04,
	0x50, 0x6a, 0x7c, 0x8a,
	0x02, 0x30, 0x40, 0x51,
	0x6a, 0x7d, 0x8d, 0x22,
	0x33, 0x44, 0x59, 0x66,
	0x71, 0x08, 0x02, 0x03,
	0x43, 0x05, 0x67, 0x7a,
	0x8f,
}

const (
	aesKeySize    = 16 // AES-128
	aesBlockSize  = 16 // AES block size, the CTR IV length
	deriveIters   = 10 // EVP_BytesToKey iteration count
	serialKeySize = 9  // a device serial is 9 characters; empty means cloud/update
)

// checkSerial mirrors the reference decryptor: the serial must be empty (for
// update/cloud files) or exactly nine characters (a device serial).
func checkSerial(serial string) error {
	if n := len(serial); n != 0 && n != serialKeySize {
		return fmt.Errorf("qc serial must be empty or %d characters, got %d", serialKeySize, n)
	}
	return nil
}

// deriveKeyIV reproduces OpenSSL's EVP_BytesToKey with SHA-1, ten iterations
// and no salt, as used by the Quad Cortex. It returns the AES-128 key and the
// CTR IV.
func deriveKeyIV(serial string) (key, iv []byte, err error) {
	if err := checkSerial(serial); err != nil {
		return nil, nil, err
	}

	data := make([]byte, 0, len(keyMaterial)+len(serial))
	data = append(data, keyMaterial...)
	data = append(data, serial...)

	out := make([]byte, 0, aesKeySize+aesBlockSize)
	var prev []byte
	for len(out) < aesKeySize+aesBlockSize {
		h := sha1.New() // #nosec G401 -- SHA-1 reproduces the Quad Cortex KDF
		if prev != nil {
			h.Write(prev)
		}
		h.Write(data)
		digest := h.Sum(nil)
		for i := 1; i < deriveIters; i++ {
			h = sha1.New() // #nosec G401 -- SHA-1 reproduces the Quad Cortex KDF
			h.Write(digest)
			digest = h.Sum(nil)
		}
		prev = digest
		out = append(out, digest...)
	}

	return out[:aesKeySize], out[aesKeySize : aesKeySize+aesBlockSize], nil
}

// newCTR returns a streaming AES-128-CTR cipher for the given serial. CTR
// mode is its own inverse, so the same stream encrypts and decrypts.
func newCTR(serial string) (cipher.Stream, error) {
	key, iv, err := deriveKeyIV(serial)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("qc: build AES cipher: %w", err)
	}
	return cipher.NewCTR(block, iv), nil
}

// Decrypt reverses the Quad Cortex's file encryption for the given serial.
func Decrypt(serial string, data []byte) ([]byte, error) {
	if err := checkSerial(serial); err != nil {
		return nil, err
	}
	stream, err := newCTR(serial)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	stream.XORKeyStream(out, data)
	return out, nil
}

// Encrypt applies the Quad Cortex's file encryption for the given serial, so
// the result is byte-compatible with the device's own on-disk preset files
// (the encrypted BinaryPreset the unit stores as <name>.pb) and can be
// decrypted with the OpenCortex tooling. It is NOT an upload path: the unit's
// firmware ignores a File CREATE's preset_payload, so this file is a reference
// archive, not something that can be pushed back onto the device.
func Encrypt(serial string, data []byte) ([]byte, error) {
	// CTR is symmetric: encrypting is the same stream operation.
	return Decrypt(serial, data)
}
