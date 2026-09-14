// Package fileutil holds small filesystem-oriented helpers shared by the
// device backends: portable file-name sanitisation, UUID generation and writing
// into a directory a caller names.
package fileutil

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeName keeps only printable ASCII filesystem-safe characters; anything
// else (accented letters, emoji, control characters) becomes an underscore, so
// the name is portable across machines and file systems.
func SanitizeName(name string) string {
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

// WriteFile writes data to path, creating the directory it sits in when that
// does not exist yet - an output directory a caller names should not have to
// exist first. Files are written 0600, like the rest of the tool's output.
func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// NewUUID returns a random version-4 UUID string.
func NewUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
