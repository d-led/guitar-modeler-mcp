package waza

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// WazaAirDeviceID is the device identifier stored inside a Waza Air liveset.
const WazaAirDeviceID = "WAZA-AIR"

// Param is one key/value pair in a patch's parameter list.
type Param struct {
	ID    string `json:"id"`
	Value any    `json:"value"`
}

// Patch is one preset inside a liveset.
type Patch struct {
	Name  string  `json:"name"`
	Param []Param `json:"param"`
}

// Liveset is a named collection of patches.
type Liveset struct {
	Name    string  `json:"name"`
	Patches []Patch `json:"patches"`
}

// TSLFile is a BOSS TONE STUDIO liveset file (.tsl) for the Waza Air.
//
// The Waza Air uses the "liveset → patches → param array" variant of the TSL
// format. This differs from the "liveSetData → patchList → flat params map"
// variant used by the Boss Katana and GT series (see katana-docs and
// lib-katana); the two variants must not be mixed.
type TSLFile struct {
	Version string  `json:"version"`
	Device  string  `json:"device"`
	Liveset Liveset `json:"liveset"`
}

// ParseTSL decodes a .tsl document. The document must carry a version and a
// device identifier; the parameter list is kept verbatim so unknown IDs are
// never lost on a round trip.
func ParseTSL(data []byte) (*TSLFile, error) {
	var f TSLFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse .tsl: %w", err)
	}
	if f.Version == "" || f.Device == "" {
		return nil, fmt.Errorf("parse .tsl: missing version or device")
	}
	return &f, nil
}

// ReadTSLFile reads and parses a .tsl file from disk.
func ReadTSLFile(path string) (*TSLFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseTSL(data)
}

// Marshal renders the liveset with the two-space indentation BOSS TONE STUDIO
// writes, followed by a trailing newline.
func (f *TSLFile) Marshal() ([]byte, error) {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// WriteTSLFile writes the liveset to disk.
func WriteTSLFile(path string, f *TSLFile) error {
	data, err := f.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// String returns the string value of a parameter, or "" when absent.
func (p Patch) String(id string) string {
	for _, prm := range p.Param {
		if prm.ID == id {
			if s, ok := prm.Value.(string); ok {
				return s
			}
			return ""
		}
	}
	return ""
}

// Int returns the integer value of a parameter, or 0 when absent.
func (p Patch) Int(id string) int {
	for _, prm := range p.Param {
		if prm.ID == id {
			switch n := prm.Value.(type) {
			case float64:
				return int(n)
			case int:
				return n
			}
			return 0
		}
	}
	return 0
}

// FirstSpec extracts the tone of the first patch using the raw TSL values. It
// deliberately does not run through the device catalog, so a value outside the
// documented lists (e.g. a reverb type "ROOM") is reported verbatim instead of
// being rejected or silently rewritten.
func (f *TSLFile) FirstSpec() Spec {
	if len(f.Liveset.Patches) == 0 {
		return Spec{Name: f.Liveset.Name}
	}
	p := f.Liveset.Patches[0]
	s := Spec{
		Name:      p.Name,
		Amp:       p.String("AMP_TYPE"),
		Gain:      p.Int("AMP_GAIN"),
		Volume:    p.Int("AMP_VOLUME"),
		DelayTime: p.Int("DELAY_TIME"),
		Reverb:    p.String("REVERB_TYPE"),
	}
	if fx := p.String("FX1_TYPE"); fx != "" {
		if strings.EqualFold(fx, "BOOSTER") {
			s.Booster = p.String("BOOSTER_TYPE")
		} else {
			s.Mod = fx
		}
	}
	return s
}

// NewTSLFile builds a one-patch liveset for a resolved Spec using the Waza
// Air's parameter IDs. Only the IDs observed in real livesets are written:
// the amp type, gain and volume, a single FX1 slot (booster OR mod/fx), and
// the delay and reverb on/off switches plus the delay time and reverb type.
// The spatial settings (cabinet resonance, ambience, position, mode) have no
// observed IDs yet and are left to BOSS TONE STUDIO.
func NewTSLFile(s Spec) *TSLFile {
	params := []Param{}
	add := func(id string, v any) { params = append(params, Param{ID: id, Value: v}) }

	if s.Amp != "" {
		add("AMP_TYPE", s.Amp)
	}
	if s.Gain > 0 {
		add("AMP_GAIN", s.Gain)
	}
	if s.Volume > 0 {
		add("AMP_VOLUME", s.Volume)
	}

	fxType := ""
	switch {
	case s.Booster != "":
		fxType = "BOOSTER"
	case s.Mod != "":
		fxType = s.Mod
	case s.FX != "":
		fxType = s.FX
	}
	if fxType != "" {
		add("FX1_TYPE", fxType)
		add("FX1_SW", "ON")
		if strings.EqualFold(fxType, "BOOSTER") {
			add("BOOSTER_TYPE", s.Booster)
		}
	}

	if s.Delay != "" {
		add("DELAY_SW", "ON")
		if s.DelayTime > 0 {
			add("DELAY_TIME", s.DelayTime)
		}
	}
	if s.Reverb != "" {
		add("REVERB_SW", "ON")
		add("REVERB_TYPE", reverbTSLValue(s.Reverb))
	}

	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = "New Patch"
	}

	return &TSLFile{
		Version: "1.0.0",
		Device:  WazaAirDeviceID,
		Liveset: Liveset{
			Name:    name,
			Patches: []Patch{{Name: name, Param: params}},
		},
	}
}

// reverbTSLValue maps a resolved device reverb name to the REVERB_TYPE value
// BOSS TONE STUDIO expects. Only the pairing observed in real livesets
// ("HALL REVERB" → "HALL") is listed; any other name passes through unchanged
// so the file is never silently rewritten with a guessed value.
func reverbTSLValue(name string) string {
	if name == "HALL REVERB" {
		return "HALL"
	}
	return name
}
