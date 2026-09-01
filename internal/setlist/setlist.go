// Package setlist writes device .setlist files: an ordered list of rigs that
// the Gigboard steps through as one bank, so a single song can use several
// incompatible chains (e.g. clean, drive, solo).
package setlist

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Setlist is a device .setlist file. The setlist's own name is not part of the
// on-disk JSON — the device names a setlist by its file name.
type Setlist struct {
	Author    string   `json:"author"`
	CreatedAt int64    `json:"created_at"`
	ID        string   `json:"id"`
	Readonly  bool     `json:"readonly"`
	RigNames  []string `json:"rig_names"`
	Rigs      []string `json:"rigs"`
	Version   string   `json:"version"`

	name string
}

// Entry is one rig referenced by a setlist, in order.
type Entry struct {
	ID   string
	Name string
}

// New builds a setlist referencing the given rigs, in order.
func New(name string, entries []Entry) (*Setlist, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("a setlist name is required")
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("a setlist needs at least one rig")
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	s := &Setlist{
		Author:    "UserName",
		CreatedAt: time.Now().Unix(),
		ID:        id,
		Readonly:  false,
		Version:   "1.0.0",
		name:      strings.TrimSpace(name),
	}
	for _, e := range entries {
		if strings.TrimSpace(e.ID) == "" {
			return nil, fmt.Errorf("rig %q has an empty id", e.Name)
		}
		s.RigNames = append(s.RigNames, e.Name)
		s.Rigs = append(s.Rigs, e.ID)
	}
	return s, nil
}

// Name returns the setlist's file name (without the .setlist extension).
func (s *Setlist) Name() string { return s.name }

// Marshal renders the setlist as compact, single-line JSON, as the device
// writes it.
func (s *Setlist) Marshal() ([]byte, error) { return json.Marshal(s) }

// Write persists the setlist to dir/<name>.setlist and returns the path.
func (s *Setlist) Write(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	name := sanitizeFileName(s.name)
	if name == "" {
		name = "setlist"
	}
	path := filepath.Join(dir, name+".setlist")
	data, err := s.Marshal()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// sanitizeFileName keeps only printable ASCII filesystem-safe characters.
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
