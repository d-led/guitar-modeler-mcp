// Package qc is the device backend for the Neural DSP Quad Cortex. The model
// catalog is parsed from the device's own ModelRepo.xml (see modelrepo.go),
// which pairs each model name with its wire hash and the real hardware it is
// "Based on". Presets are encrypted protobufs (BinaryPreset); this package
// reads and writes them without any non-public key — see crypto.go.
package qc

import (
	"fmt"
	"strings"
)

// Item is one selectable model: the on-device name, its wire hash (id) and
// the real hardware it emulates (empty when not documented).
type Item struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	InspiredBy string `json:"inspired_by,omitempty"`
}

// Device describes the Quad Cortex and its model catalog. The QC uses a free
// 4-lane signal grid (each lane a serial chain that can split and merge),
// rather than a fixed module order, so there is no ModuleOrder here.
type Device struct {
	Name         string
	Display      string
	FileExchange bool
	FileExt      string
	// Topology describes the grid layout for the agent guide.
	Topology string
	// Catalog is the full parsed ModelRepo, keyed by wire hash.
	Catalog *Catalog
	// Amps is the guitar amp list.
	Amps []Item
	// BassAmps is the bass amp list.
	BassAmps []Item
	// Cabs is the guitar cabinet (cabsim) list.
	Cabs []Item
	// Effects maps a category to its models: drive, compressor, equalizer,
	// delay, modulation, reverb, wah, pitch, filter and gate.
	Effects map[string][]Item
}

// Default returns the Quad Cortex device model, sourced from the embedded
// ModelRepo catalog. It panics if the embedded catalog fails to parse, which
// is a build-time bug, not a runtime input error.
func Default() Device {
	cat, err := defaultCatalog()
	if err != nil {
		panic(fmt.Sprintf("qc: embedded ModelRepo: %v", err))
	}
	d := Device{
		Name:         "quad-cortex",
		Display:      "Neural DSP Quad Cortex",
		FileExchange: false, // no device-importable file: .pb is our own reference archive
		FileExt:      "",
		Topology:     "free 4-lane grid; each lane is a serial chain that can split and merge",
		Catalog:      cat,
		Effects:      map[string][]Item{},
	}
	d.Amps = catItems(cat, "Guitar Amplifier")
	d.BassAmps = catItems(cat, "Bass Amplifier")
	d.Cabs = catItems(cat, "Cabsim Guitar (M)", "Cabsim Guitar (ST)")
	d.Effects["drive"] = catItems(cat, "Guitar Overdrive")
	d.Effects["compressor"] = catItems(cat, "Compressor")
	d.Effects["equalizer"] = catItems(cat, "Equalizer")
	d.Effects["delay"] = catItems(cat, "Delay")
	d.Effects["modulation"] = catItems(cat, "Modulation")
	d.Effects["reverb"] = catItems(cat, "Reverb")
	d.Effects["wah"] = catItems(cat, "Wah")
	d.Effects["pitch"] = catItems(cat, "Pitch")
	d.Effects["filter"] = catItems(cat, "Filter")
	d.Effects["gate"] = gateItems(cat)
	return d
}

// catItems returns every model of the given categories, in wire-hash order.
func catItems(cat *Catalog, categories ...string) []Item {
	want := make(map[string]bool, len(categories))
	for _, c := range categories {
		want[c] = true
	}
	var out []Item
	for _, m := range cat.sortedByID() {
		if want[m.Category] {
			out = append(out, itemOf(m))
		}
	}
	return out
}

// gateItems returns the noise-gate subset of the Utility category.
func gateItems(cat *Catalog) []Item {
	var out []Item
	for _, m := range cat.sortedByID() {
		if m.Category == "Utility" && strings.Contains(m.Name, "Gate") {
			out = append(out, itemOf(m))
		}
	}
	return out
}

func itemOf(m *ModelSpec) Item {
	return Item{ID: m.ID, Name: m.Name, InspiredBy: m.BasedOn}
}
