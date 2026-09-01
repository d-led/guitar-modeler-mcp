package waza

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// WazaAirDeviceID is the device identifier stored inside a Waza Air backup.
const WazaAirDeviceID = "WAZA-AIR"

// patchNameSize is the fixed width of the patch name field, space-padded.
const patchNameSize = 16

// defaultPatchTSL is a neutral, known-good single-patch backup ("Init Tone"):
// a CLEAN amp at noon with every effect block off. It is the byte container
// new presets start from, so unspecified effects stay off rather than
// inheriting someone's preset.
//
//go:embed default-patch.tsl
var defaultPatchTSL []byte

// Entry is one slot in a backup: an optional memo plus its parameter set. The
// parameter set maps keys (e.g. "User%Patch") to arrays of two-digit
// upper-case hex bytes.
type Entry struct {
	Memo     string              `json:"memo"`
	ParamSet map[string][]string `json:"paramSet"`
}

// Backup is a BOSS TONE STUDIO backup (.tsl) for the Waza Air: a named set of
// one or more patches. Each patch is a fixed-size binary record stored under
// the "User%Patch" key as an array of hex bytes. This is the format the Waza
// Air app exports and imports; it differs from the Katana/GT "liveSetData →
// patchList" variant.
type Backup struct {
	Name      string    `json:"name"`
	FormatRev string    `json:"formatRev"`
	Device    string    `json:"device"`
	Data      [][]Entry `json:"data"`
}

// Patch is one decoded preset: its name and the raw record bytes.
type Patch struct {
	Name string
	Raw  []byte
}

// ParseTSL decodes a .tsl backup document.
func ParseTSL(data []byte) (*Backup, error) {
	var b Backup
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse .tsl: %w", err)
	}
	if b.Device == "" {
		return nil, fmt.Errorf("parse .tsl: missing device")
	}
	return &b, nil
}

// ReadTSLFile reads and parses a .tsl backup from disk.
func ReadTSLFile(path string) (*Backup, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseTSL(data)
}

// Marshal renders the backup as compact JSON with upper-case hex bytes and a
// trailing newline.
func (b *Backup) Marshal() ([]byte, error) {
	out, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// WriteTSLFile writes the backup to disk.
func WriteTSLFile(path string, b *Backup) error {
	data, err := b.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// NewBackup returns an empty Waza Air backup.
func NewBackup(name string) *Backup {
	return &Backup{Name: name, FormatRev: "0000", Device: WazaAirDeviceID}
}

// Patches returns every patch in the backup, decoded from their hex records.
func (b *Backup) Patches() []Patch {
	var out []Patch
	for _, bank := range b.Data {
		for _, e := range bank {
			hexs, ok := e.ParamSet["User%Patch"]
			if !ok {
				continue
			}
			out = append(out, NewPatch(decodeHex(hexs)))
		}
	}
	return out
}

// SetPatches replaces the backup's patches with the given ones, one entry per
// patch in a single bank.
func (b *Backup) SetPatches(patches []Patch) {
	bank := make([]Entry, 0, len(patches))
	for _, p := range patches {
		bank = append(bank, Entry{ParamSet: map[string][]string{"User%Patch": encodeHex(p.Raw)}})
	}
	b.Data = [][]Entry{bank}
}

// NewPatch builds a patch from a raw record, decoding its name.
func NewPatch(raw []byte) Patch {
	return Patch{Name: decodeName(raw), Raw: append([]byte(nil), raw...)}
}

// WithName returns a copy of the patch with its name field replaced. Names are
// truncated to 16 bytes and space-padded.
func (p Patch) WithName(name string) Patch {
	out := NewPatch(p.Raw)
	pad := encodeName(name)
	copy(out.Raw[:patchNameSize], pad[:])
	out.Name = decodeName(out.Raw)
	return out
}

// TemplatePatch returns the built-in neutral patch (a CLEAN amp at noon with
// every effect off), so a new preset starts from a known-good record and only
// the specified tone is applied.
func TemplatePatch() (Patch, error) {
	b, err := ParseTSL(defaultPatchTSL)
	if err != nil {
		return Patch{}, err
	}
	patches := b.Patches()
	if len(patches) == 0 {
		return Patch{}, fmt.Errorf("default patch template is empty")
	}
	return patches[0], nil
}

func encodeHex(raw []byte) []string {
	out := make([]string, len(raw))
	for i, b := range raw {
		out[i] = fmt.Sprintf("%02X", b)
	}
	return out
}

func decodeHex(hexs []string) []byte {
	raw := make([]byte, len(hexs))
	for i, h := range hexs {
		if len(h) != 2 {
			continue
		}
		b, err := hex.DecodeString(h)
		if err == nil && len(b) == 1 {
			raw[i] = b[0]
		}
	}
	return raw
}

func encodeName(name string) [patchNameSize]byte {
	var out [patchNameSize]byte
	copy(out[:], name)
	for i := range out {
		if out[i] == 0 {
			out[i] = ' '
		}
	}
	return out
}

func decodeName(raw []byte) string {
	if len(raw) < patchNameSize {
		return ""
	}
	return strings.TrimRight(string(raw[:patchNameSize]), " ")
}
