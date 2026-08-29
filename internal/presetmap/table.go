// Package presetmap cross-links presets between devices. It does not mix the
// device backends: each device package knows its own models and the real
// hardware they emulate ("inspired by"). This package joins those two tables
// through a shared canonical key, so a model on one device can be mapped to
// the model emulating the same hardware on another.
package presetmap

import (
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
)

// Device names, used as the device axis of the lookup tables.
const (
	DeviceGigboard = "gigboard"
	DeviceMooer    = "mooer"
)

// Devices lists the supported device names in a stable order.
var Devices = []string{DeviceGigboard, DeviceMooer}

// normalize lowercases and collapses whitespace, the first step towards a
// canonical "inspired by" key.
func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
}

// keyAliases reconciles the two devices' spellings of the same real hardware
// into one canonical key. Each side's raw "inspired by" string passes through
// canonicalKey; equal keys mean the two models emulate the same hardware.
var keyAliases = map[string]string{
	// amps
	"fender twin reverb (shimmer)":       "fender twin reverb",
	"fender tweed bassman":               "fender bassman",
	"fender princeton reverb":            "fender princeton",
	"marshall jcm800 2203":               "marshall jcm800",
	"marshall jcm800 2204":               "marshall jcm800",
	"marshall jcm800 (bright)":           "marshall jcm800",
	"marshall jcm800 (bass mod)":         "marshall jcm800",
	"marshall jcm800 (ts mod)":           "marshall jcm800",
	"marshall super lead plexi (variac)": "marshall super lead",
	"marshall super lead (el84 mod)":     "marshall super lead",
	"marshall super lead 1987":           "marshall super lead",
	"marshall super lead 1959":           "marshall super lead",
	"bogner ecstasy 101b":                "bogner ecstasy",
	"peavey 5150 ii":                     "peavey 5150",
	"orange ad30 twin channel":           "orange ad30",
	"matchless dc-30":                    "matchless dc30",
	"ampeg svt (scooped)":                "ampeg svt",
	// cabs
	"celestion greenback 20w":   "celestion greenback 4x12",
	"celestion greenback 25w":   "celestion greenback 4x12",
	"celestion greenback 4x12":  "celestion greenback 4x12",
	"celestion alnico blue":     "vox 2x12 alnico blue",
	"vox ac30 2x12 alnico blue": "vox 2x12 alnico blue",
	"vox ac30 2x12":             "vox 2x12 alnico blue",
	"blackface 2x12":            "fender twin 2x12",
	"fender twin 2x12":          "fender twin 2x12",
	"blackface 1x12":            "fender deluxe reverb 1x12",
	"fender deluxe reverb 1x12": "fender deluxe reverb 1x12",
	"tweed 4x10":                "fender bassman 4x10",
	"fender bassman 4x10":       "fender bassman 4x10",
	"celestion 65w":             "marshall 4x12",
	"marshall 1960a 4x12":       "marshall 4x12",
	"marshall 1960b 4x12":       "marshall 4x12",
	// effects
	"ibanez ts808":         "ibanez tube screamer",
	"ibanez ts9":           "ibanez tube screamer",
	"ibanez tube screamer": "ibanez tube screamer",
	"boss ds-1":            "boss ds-1",
	"ehx big muff":         "ehx big muff",
	"studio compressor":    "studio compressor",
	"ross compressor":      "ross compressor",
	"envelope filter":      "envelope filter",
	"ring modulator":       "ring modulator",
	"uni-vibe":             "uni-vibe",
	"volume pedal":         "volume pedal",
	"digital delay":        "digital delay",
	"analog delay":         "analog delay",
	"tape echo":            "tape echo",
	"reverse delay":        "reverse delay",
	"room reverb":          "room reverb",
	"hall reverb":          "hall reverb",
	"spring reverb":        "spring reverb",
	"modulated reverb":     "modulated reverb",
	"shimmer reverb":       "shimmer reverb",
	"ambient reverb":       "ambient reverb",
}

func canonicalKey(s string) string {
	n := normalize(s)
	if k, ok := keyAliases[n]; ok {
		return k
	}
	return n
}

// gigboardCabInspiredBy maps a Gigboard cab model to the real cabinet it
// emulates. It is the explicit counterpart of the Mooer cabInspiredBy table.
var gigboardCabInspiredBy = map[string]string{
	"1x12 Black Panel Lux": "Fender Deluxe Reverb 1x12",
	"2x12 AC Blue":         "Vox AC30 2x12 Alnico Blue",
	"2x12 Black Panel Duo": "Fender Twin 2x12",
	"4x10 Tweed Bass":      "Fender Bassman 4x10",
	"4x12 65W":             "Marshall 1960A 4x12",
	"4x12 Green 20W":       "Celestion Greenback 20W",
	"4x12 Green 25W":       "Celestion Greenback 25W",
}

// gigboardFXInspiredBy maps a Gigboard effect module to the real effect it
// emulates, for the effects that have a Mooer counterpart.
var gigboardFXInspiredBy = map[string]string{
	"Green JRC-OD":  "Ibanez TS808",
	"DC Distort":    "BOSS DS-1",
	"Round Fuzz":    "EHX Big Muff",
	"Chorus":        "Chorus",
	"Flanger":       "Flanger",
	"Orange Phaser": "Phaser",
	"Vibrato":       "Vibrato",
	"Rotary":        "Rotary",
	"Tremolo":       "Tremolo",
	"Ring Mod":      "Ring Modulator",
	"Vibe Phaser":   "Uni-Vibe",
	"Env Filter":    "Envelope Filter",
	"Detune":        "Detune",
	"Smart Harm":    "Harmonizer",
	"BBD Delay":     "Analog Delay",
	"AIR Delay":     "Digital Delay",
	"Tape Echo":     "Tape Echo",
	"Reverse Delay": "Reverse Delay",
	"Eleven Reverb": "Room Reverb",
	"AIR Reverb":    "Hall Reverb",
	"Spring Reverb": "Spring Reverb",
	"Party Verb":    "Modulated Reverb",
	"Shimmer":       "Shimmer Reverb",
	"Ambi Verb":     "Ambient Reverb",
	"DynIII Comp":   "Studio Compressor",
	"Gray Comp":     "Ross Compressor",
	"Graphic EQ":    "Graphic EQ",
	"Black Wah":     "Wah",
	"Volume":        "Volume Pedal",
	"Octaves":       "Octave",
}

// linkIndex is one cross-device lookup table (amps, cabs, effects, mics). It
// stores, for each device, which canonical key a model resolves to, and for
// each key which representative model each device uses.
type linkIndex struct {
	byModel map[string]map[string]string // device -> lower(model) -> key
	byKey   map[string]map[string]string // key -> device -> model
}

func newLinkIndex() *linkIndex {
	return &linkIndex{
		byModel: make(map[string]map[string]string),
		byKey:   make(map[string]map[string]string),
	}
}

// add registers a device model under a canonical key. The first model added
// for a (key, device) pair becomes that device's representative model.
func (li *linkIndex) add(device, model, key string) {
	if key == "" {
		return
	}
	if li.byModel[device] == nil {
		li.byModel[device] = make(map[string]string)
	}
	li.byModel[device][normalize(model)] = key

	if li.byKey[key] == nil {
		li.byKey[key] = make(map[string]string)
	}
	if _, exists := li.byKey[key][device]; !exists {
		li.byKey[key][device] = model
	}
}

// mapTo resolves model on fromDevice to the model emulating the same hardware
// on toDevice.
func (li *linkIndex) mapTo(fromDevice, model, toDevice string) (string, bool) {
	key, ok := li.byModel[fromDevice][normalize(model)]
	if !ok {
		return "", false
	}
	target, ok := li.byKey[key][toDevice]
	return target, ok
}

// fxTable is the effect lookup table. Effects are not unambiguous by name
// alone — the Mooer device can place the same effect in more than one module
// (e.g. "Tremolo" in both FX and MOD) — so the Mooer side is keyed by
// (module, name), and the Gigboard side carries its effect category, which
// decides the target Mooer module.
type fxTable struct {
	gigNameKey map[string]string            // lower(gig name) -> canonical key
	gigCat     map[string]string            // lower(gig name) -> category
	gigByKey   map[string]string            // key -> gigboard name
	mooNameKey map[string]map[string]string // module -> lower(name) -> key
	mooByKey   map[string]map[string]string // key -> module -> name
}

func newFXTable() *fxTable {
	return &fxTable{
		gigNameKey: make(map[string]string),
		gigCat:     make(map[string]string),
		gigByKey:   make(map[string]string),
		mooNameKey: make(map[string]map[string]string),
		mooByKey:   make(map[string]map[string]string),
	}
}

func (ft *fxTable) addGigboard(name, category, key string) {
	if key == "" {
		return
	}
	lower := normalize(name)
	ft.gigNameKey[lower] = key
	ft.gigCat[lower] = category
	if _, ok := ft.gigByKey[key]; !ok {
		ft.gigByKey[key] = name
	}
}

func (ft *fxTable) addMooer(module, name, key string) {
	if key == "" {
		return
	}
	if ft.mooNameKey[module] == nil {
		ft.mooNameKey[module] = make(map[string]string)
	}
	ft.mooNameKey[module][normalize(name)] = key

	if ft.mooByKey[key] == nil {
		ft.mooByKey[key] = make(map[string]string)
	}
	if _, ok := ft.mooByKey[key][module]; !ok {
		ft.mooByKey[key][module] = name
	}
}

// categoryToMooerModule routes a Gigboard effect category to the Mooer module
// that carries that kind of effect.
var categoryToMooerModule = map[string]string{
	"distortion": "od",
	"dynamics":   "fx",
	"expression": "fx",
	"modulation": "mod",
	"delay":      "delay",
	"reverb":     "reverb",
}

// gigboardToMooer maps a Gigboard effect name to the Mooer (module, name) that
// emulates the same effect.
func (ft *fxTable) gigboardToMooer(gigName string) (module, name string, ok bool) {
	lower := normalize(gigName)
	key, ok := ft.gigNameKey[lower]
	if !ok {
		return "", "", false
	}
	module = categoryToMooerModule[ft.gigCat[lower]]
	if module == "" {
		return "", "", false
	}
	name, ok = ft.mooByKey[key][module]
	return module, name, ok
}

// mooerToGigboard maps a Mooer (module, name) to the Gigboard effect name.
func (ft *fxTable) mooerToGigboard(module, name string) (string, bool) {
	key, ok := ft.mooNameKey[module][normalize(name)]
	if !ok {
		return "", false
	}
	gig, ok := ft.gigByKey[key]
	return gig, ok
}

// Table is the set of cross-device lookup tables, built once from the device
// catalogs.
type Table struct {
	amps  *linkIndex
	cabs  *linkIndex
	fx    *fxTable
	mics  *linkIndex
	mooer mooer.Model
}

// NewTable builds the cross-device lookup tables from the Gigboard catalog and
// a Mooer model (the device the Mooer side of the lookup is built from).
func NewTable(cat *catalog.Catalog, m mooer.Model) *Table {
	t := &Table{amps: newLinkIndex(), cabs: newLinkIndex(), fx: newFXTable(), mics: newLinkIndex(), mooer: m}

	for _, a := range cat.Amps() {
		t.amps.add(DeviceGigboard, a.Model, canonicalKey(a.Brand+" "+a.RealModel))
	}
	for _, a := range m.Amps {
		if raw := a.InspiredBy; raw != "" {
			t.amps.add(DeviceMooer, a.Name, canonicalKey(raw))
		}
	}

	for _, c := range cat.Cabs() {
		if raw, ok := gigboardCabInspiredBy[c.Model]; ok {
			t.cabs.add(DeviceGigboard, c.Model, canonicalKey(raw))
		}
	}
	for _, c := range m.Cabs {
		if raw := c.InspiredBy; raw != "" {
			t.cabs.add(DeviceMooer, c.Name, canonicalKey(raw))
		}
	}

	for _, f := range cat.FX() {
		if raw, ok := gigboardFXInspiredBy[f.Name]; ok {
			t.fx.addGigboard(f.Name, f.Category, canonicalKey(raw))
		}
	}
	for module, list := range m.Effects {
		for _, f := range list {
			if raw := f.InspiredBy; raw != "" {
				t.fx.addMooer(module, f.Name, canonicalKey(raw))
			}
		}
	}

	for _, m := range cat.Mics() {
		t.mics.add(DeviceGigboard, m.Model, canonicalKey(m.RealModel))
	}
	return t
}

// MapAmp resolves an amp model from one device to the amp emulating the same
// hardware on the other.
func (t *Table) MapAmp(fromDevice, model, toDevice string) (string, bool) {
	return t.amps.mapTo(fromDevice, model, toDevice)
}

// MapCab resolves a cab model from one device to the other.
func (t *Table) MapCab(fromDevice, model, toDevice string) (string, bool) {
	return t.cabs.mapTo(fromDevice, model, toDevice)
}

// MapFXGigboardToMooer maps a Gigboard effect name to the Mooer module and
// effect name emulating the same effect.
func (t *Table) MapFXGigboardToMooer(gigName string) (module, name string, ok bool) {
	return t.fx.gigboardToMooer(gigName)
}

// MapFXMooerToGigboard maps a Mooer effect (module, name) to the Gigboard
// effect name.
func (t *Table) MapFXMooerToGigboard(module, name string) (string, bool) {
	return t.fx.mooerToGigboard(module, name)
}

// MapMic resolves a microphone model from one device to the other.
func (t *Table) MapMic(fromDevice, model, toDevice string) (string, bool) {
	return t.mics.mapTo(fromDevice, model, toDevice)
}
