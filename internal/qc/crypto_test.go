package qc

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// TestDeriveKeyIV pins the key derivation against vectors computed
// independently from the same KEY_MATERIAL and the EVP_BytesToKey
// specification (SHA-1, 10 iterations, no salt).
func TestDeriveKeyIV(t *testing.T) {
	cases := []struct {
		serial, keyHex, ivHex string
	}{
		{"QA00XXXXX", "afe20ad5964aa9b63288cea6a8b1de58", "ea01460b64359aec0d5649b9890186fa"},
		{"", "8dd6f4088e4eb377b82b5ec89431c897", "0e2b764293bf1976c3a1dfe9fefd7589"},
	}
	for _, c := range cases {
		key, iv, err := deriveKeyIV(c.serial)
		if err != nil {
			t.Fatalf("deriveKeyIV(%q): %v", c.serial, err)
		}
		if got := hex.EncodeToString(key); got != c.keyHex {
			t.Fatalf("deriveKeyIV(%q) key = %s, want %s", c.serial, got, c.keyHex)
		}
		if got := hex.EncodeToString(iv); got != c.ivHex {
			t.Fatalf("deriveKeyIV(%q) iv = %s, want %s", c.serial, got, c.ivHex)
		}
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plain := []byte("a Quad Cortex preset: BinaryPreset protobuf")
	enc, err := Encrypt("QA00XXXXX", plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(enc, plain) {
		t.Fatal("ciphertext equals plaintext; encryption did nothing")
	}
	dec, err := Decrypt("QA00XXXXX", enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Fatalf("round trip mismatch:\n got %q\nwant %q", dec, plain)
	}
}

func TestDecryptRejectsBadSerial(t *testing.T) {
	for _, serial := range []string{"QA00", "QA00XXXXXXXX"} {
		if _, err := Decrypt(serial, []byte{0x00}); err == nil {
			t.Fatalf("Decrypt(%q) succeeded, want error", serial)
		}
	}
}

func TestEmptySerialRoundTrip(t *testing.T) {
	plain := []byte("cloud file")
	enc, err := Encrypt("", plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	dec, err := Decrypt("", enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Fatalf("empty-serial round trip mismatch: got %q want %q", dec, plain)
	}
}
