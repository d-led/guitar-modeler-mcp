package mooer

import "strings"

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
