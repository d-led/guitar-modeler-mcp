package mooer

import (
	"fmt"
	"strings"
)

// ModuleOrder is the fixed signal-chain order of the device's nine modules.
// It is also the order the modules appear in a preset record on the wire.
var ModuleOrder = []string{"fx", "od", "amp", "cab", "ns", "eq", "mod", "delay", "reverb"}

// Item is one named model on a Mooer device: the on-device screen name and the
// real hardware it emulates (empty when not documented).
type Item struct {
	Name       string `json:"name"`
	InspiredBy string `json:"inspired_by,omitempty"`
}

// Model describes one Mooer multi-effects device: its fixed module chain, its
// per-module model lists (in effect_type index order), and whether presets can
// be exchanged as files or must be reproduced by hand from a printable card.
type Model struct {
	// Name is the stable identifier, e.g. "ge200".
	Name string `json:"name"`
	// Display is the human-readable name, e.g. "Mooer GE200".
	Display string `json:"display"`
	// FileExchange reports whether presets can be transferred via a file. When
	// false the device has no USB preset transfer and a printable setup card is
	// the output instead.
	FileExchange bool `json:"file_exchange"`
	// FileExt is the preset file extension when FileExchange is true.
	FileExt string `json:"file_ext,omitempty"`
	// ModuleOrder is the fixed signal-chain order of the modules.
	ModuleOrder []string `json:"module_order"`
	// Amps is the amp list in effect_type index order.
	Amps []Item `json:"amps"`
	// Cabs is the cabinet list in effect_type index order.
	Cabs []Item `json:"cabs"`
	// Effects is each module's effect list in effect_type index order.
	Effects map[string][]Item `json:"effects"`
}

// AmpName returns the amp model for an effect_type index, or "" when out of
// range.
func (m Model) AmpName(index uint8) string {
	return itemName(m.Amps, index)
}

// AmpIndex returns the effect_type index for a named amp model.
func (m Model) AmpIndex(name string) (uint8, bool) {
	return itemIndex(m.Amps, name)
}

// CabName returns the cab model for an effect_type index, or "" when out of
// range.
func (m Model) CabName(index uint8) string {
	return itemName(m.Cabs, index)
}

// CabIndex returns the effect_type index for a named cab model.
func (m Model) CabIndex(name string) (uint8, bool) {
	return itemIndex(m.Cabs, name)
}

// EffectName returns the human-readable effect name for a module and
// effect_type index. Modules with a fixed single effect (ns, eq) return their
// own name.
func (m Model) EffectName(module string, index uint8) string {
	switch strings.ToLower(module) {
	case "amp":
		return m.AmpName(index)
	case "cab":
		return m.CabName(index)
	case "ns":
		return "Noise Gate"
	case "eq":
		return "EQ"
	}
	return itemName(m.Effects[strings.ToLower(module)], index)
}

// EffectIndex returns the effect_type index for a named effect in a module.
func (m Model) EffectIndex(module, name string) (uint8, bool) {
	return itemIndex(m.Effects[strings.ToLower(module)], name)
}

// InspiredAmp returns the real amplifier an amp model emulates.
func (m Model) InspiredAmp(name string) (string, bool) {
	return itemInspiredBy(m.Amps, name)
}

// InspiredCab returns the real cabinet a cab model emulates.
func (m Model) InspiredCab(name string) (string, bool) {
	return itemInspiredBy(m.Cabs, name)
}

// InspiredFX returns the real effect an effect (module, name) emulates.
func (m Model) InspiredFX(module, name string) (string, bool) {
	return itemInspiredBy(m.Effects[strings.ToLower(module)], name)
}

func itemName(items []Item, index uint8) string {
	if int(index) >= len(items) {
		return ""
	}
	return items[index].Name
}

func itemIndex(items []Item, name string) (uint8, bool) {
	for i, it := range items {
		if strings.EqualFold(it.Name, name) {
			return uint8(i), true
		}
	}
	return 0, false
}

func itemInspiredBy(items []Item, name string) (string, bool) {
	for _, it := range items {
		if strings.EqualFold(it.Name, name) {
			return it.InspiredBy, it.InspiredBy != ""
		}
	}
	return "", false
}

// FXSpec is one effect to place in a module of the fixed chain.
type FXSpec struct {
	Module  string // fx, od, mod, delay, reverb, ns or eq
	Type    string // effect name within that module
	Enabled bool
}

// Spec is a tone to dial in on a Mooer device.
type Spec struct {
	Name string
	Amp  string // amp model name, or a real-hardware description to resolve
	Cab  string // cab model name or description (optional)
	FX   []FXSpec
}

// ResolveAmp finds the amp effect_type for a query: exact model name first,
// then a case-insensitive substring match against the "inspired by" hardware.
func (m Model) ResolveAmp(query string) (uint8, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, fmt.Errorf("an amp is required")
	}
	if index, ok := m.AmpIndex(q); ok {
		return index, nil
	}
	for i, a := range m.Amps {
		if a.InspiredBy != "" && strings.Contains(strings.ToLower(a.InspiredBy), strings.ToLower(q)) {
			return uint8(i), nil
		}
	}
	return 0, fmt.Errorf("no %s amp matches %q; list them with mooer_catalog_list_amps", m.Name, q)
}

// ResolveCab finds the cab effect_type for a query. An empty query returns 0
// (cab is optional).
func (m Model) ResolveCab(query string) (uint8, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, nil
	}
	if index, ok := m.CabIndex(q); ok {
		return index, nil
	}
	for i, c := range m.Cabs {
		if c.InspiredBy != "" && strings.Contains(strings.ToLower(c.InspiredBy), strings.ToLower(q)) {
			return uint8(i), nil
		}
	}
	return 0, fmt.Errorf("no %s cab matches %q; list them with mooer_catalog_list_cabs", m.Name, q)
}

// ResolveFX finds the effect_type for a named effect in a module.
func (m Model) ResolveFX(module, name string) (uint8, error) {
	if index, ok := m.EffectIndex(module, name); ok {
		return index, nil
	}
	return 0, fmt.Errorf("no %s %s effect %q", m.Name, module, name)
}

// BuildPreset resolves a Spec into a concrete Preset for this model.
func (m Model) BuildPreset(s Spec) (Preset, error) {
	p := New()
	p.Name = s.Name

	ampIndex, err := m.ResolveAmp(s.Amp)
	if err != nil {
		return p, err
	}
	p.Amp = Amp{Enabled: true, Type: ampIndex}

	if s.Cab != "" {
		cabIndex, err := m.ResolveCab(s.Cab)
		if err != nil {
			return p, err
		}
		p.Cab = Cab{Enabled: true, Type: cabIndex}
	}

	for _, f := range s.FX {
		module := strings.ToLower(strings.TrimSpace(f.Module))
		if module == "ds" {
			module = "od"
		}
		index, err := m.ResolveFX(module, f.Type)
		if err != nil {
			return p, err
		}
		setModule(&p, module, index, f.Enabled)
	}
	return p, nil
}

// setModule places an effect_type into one module of the fixed chain.
func setModule(p *Preset, module string, index uint8, enabled bool) {
	switch module {
	case "fx":
		p.FX = FX{Enabled: enabled, Type: index}
	case "od":
		p.Drive = Drive{Enabled: enabled, Type: index}
	case "ns":
		p.NoiseGate = NoiseGate{Enabled: enabled, Type: index}
	case "eq":
		p.EQ = EQ{Enabled: enabled, Type: index}
	case "mod":
		p.Mod = Mod{Enabled: enabled, Type: index}
	case "delay":
		p.Delay = Delay{Enabled: enabled, Type: index}
	case "reverb":
		p.Reverb = Reverb{Enabled: enabled, Type: index}
	}
}
