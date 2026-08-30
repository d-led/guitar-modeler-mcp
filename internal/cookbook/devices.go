package cookbook

import (
	"fmt"
	"sort"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/params"
	"github.com/d-led/guitar-modeler-mcp/internal/qc"
	"github.com/d-led/guitar-modeler-mcp/internal/thr"
	"github.com/d-led/guitar-modeler-mcp/internal/waza"
)

// Ingredients returns every block of a device, reduced to tagged ingredients.
// The tags are derived statically from each catalog's own fields (category,
// gain character, speaker config) plus the keyword dictionary in cookbook.go.
func Ingredients(device string) ([]Ingredient, error) {
	d := strings.ToLower(strings.TrimSpace(device))
	switch d {
	case "gigboard", "headrush", "headrush-gigboard":
		return fromGigboard(), nil
	case "quad-cortex", "quad cortex", "qc", "neural-dsp-quad-cortex":
		return fromQC(), nil
	case "wazaair", "waza", "waza-air", "boss-waza-air":
		return fromWaza(), nil
	}
	for _, m := range mooer.Models() {
		if strings.EqualFold(m.Name, d) {
			return fromMooer(m), nil
		}
	}
	for _, t := range thr.Models() {
		if strings.EqualFold(t.Name, d) {
			return fromTHR(t), nil
		}
	}
	return nil, fmt.Errorf("unknown device %q", device)
}

func fromGigboard() []Ingredient {
	cat := catalog.New()
	ampParams := gigboardParamNames(cat, "Amp")
	cabParams := gigboardParamNames(cat, "Cab")
	var out []Ingredient
	for _, a := range cat.Amps() {
		kind := KindAmp
		if a.Bass {
			kind = KindBassAmp
		}
		extra := append([]string{}, a.Style...)
		extra = append(extra, a.Brand)
		in := newIngredient("gigboard", kind, a.Model, a.ModeledAfter, kind, a.Gain,
			strings.Join(extra, " ")+" "+a.Description)
		in.Params = ampParams
		out = append(out, in)
	}
	for _, c := range cat.Cabs() {
		in := newIngredient("gigboard", KindCab, c.Model, c.ModeledAfter, KindCab, "",
			c.Speakers+" "+c.SpeakersRef+" "+c.Description)
		in.Params = cabParams
		out = append(out, in)
	}
	for _, f := range cat.FX() {
		in := newIngredient("gigboard", KindFX, f.Name, f.ModeledAfter, primaryFX(f.Category), "",
			f.Description+" "+f.Category)
		in.Params = gigboardParamNames(cat, f.Name)
		out = append(out, in)
	}
	return out
}

// gigboardParamNames returns the sorted parameter names of a module, or nil
// when the catalog has no spec for it.
func gigboardParamNames(cat *catalog.Catalog, name string) []string {
	spec, err := params.Describe(cat, name)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(spec))
	for k := range spec {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func fromQC() []Ingredient {
	d := qc.Default()
	var out []Ingredient
	for _, a := range d.Amps {
		in := newIngredient(d.Name, KindAmp, a.Name, a.InspiredBy, KindAmp, "", a.Name)
		in.Params = qcParamNames(d, a.ID)
		out = append(out, in)
	}
	for _, a := range d.BassAmps {
		in := newIngredient(d.Name, KindBassAmp, a.Name, a.InspiredBy, KindBassAmp, "", a.Name)
		in.Params = qcParamNames(d, a.ID)
		out = append(out, in)
	}
	for _, c := range d.Cabs {
		in := newIngredient(d.Name, KindCab, c.Name, c.InspiredBy, KindCab, "", c.Name)
		in.Params = qcParamNames(d, c.ID)
		out = append(out, in)
	}
	for category, items := range d.Effects {
		for _, f := range items {
			in := newIngredient(d.Name, KindFX, f.Name, f.InspiredBy, primaryFX(category), "", f.Name)
			in.Params = qcParamNames(d, f.ID)
			out = append(out, in)
		}
	}
	return out
}

// qcParamNames returns the parameter names of a Quad Cortex model, or nil.
func qcParamNames(d qc.Device, id int) []string {
	m, ok := d.Catalog.Model(id)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(m.Params))
	for _, p := range m.Params {
		if p.Name != "" {
			out = append(out, p.Name)
		}
	}
	return out
}

func fromWaza() []Ingredient {
	d := waza.Default()
	var out []Ingredient
	for _, a := range d.Amps {
		out = append(out, newIngredient(d.Name, KindAmp, a.Name, a.InspiredBy, KindAmp, "", a.Name))
	}
	for _, b := range d.Boosters {
		out = append(out, newIngredient(d.Name, KindFX, b.Name, b.InspiredBy, "drive", "", b.Name))
	}
	for _, f := range d.ModFX {
		out = append(out, newIngredient(d.Name, KindFX, f.Name, f.InspiredBy, "mod", "", f.Name))
	}
	for _, f := range d.Delays {
		out = append(out, newIngredient(d.Name, KindFX, f.Name, f.InspiredBy, "delay", "", f.Name))
	}
	for _, f := range d.Reverbs {
		out = append(out, newIngredient(d.Name, KindFX, f.Name, f.InspiredBy, "reverb", "", f.Name))
	}
	return out
}

func fromMooer(m mooer.Model) []Ingredient {
	var out []Ingredient
	for _, a := range m.Amps {
		out = append(out, newIngredient(m.Name, KindAmp, a.Name, a.InspiredBy, KindAmp, "", a.Name))
	}
	for _, c := range m.Cabs {
		out = append(out, newIngredient(m.Name, KindCab, c.Name, c.InspiredBy, KindCab, "", c.Name))
	}
	for module, items := range m.Effects {
		for _, f := range items {
			out = append(out, newIngredient(m.Name, KindFX, f.Name, f.InspiredBy, primaryFX(module), "", f.Name))
		}
	}
	return out
}

func fromTHR(t thr.Device) []Ingredient {
	var out []Ingredient
	for _, a := range t.Amps {
		out = append(out, newIngredient(t.Name, KindAmp, a.Name, a.InspiredBy, KindAmp, "", a.Name+" "+a.Description))
	}
	for _, c := range t.Cabs {
		out = append(out, newIngredient(t.Name, KindCab, c.Name, c.InspiredBy, KindCab, "", c.Name))
	}
	for _, f := range t.Modulation {
		out = append(out, newIngredient(t.Name, KindFX, f.Name, f.InspiredBy, "mod", "", f.Name))
	}
	for _, f := range t.Echo {
		out = append(out, newIngredient(t.Name, KindFX, f.Name, f.InspiredBy, "delay", "", f.Name))
	}
	for _, f := range t.Reverb {
		out = append(out, newIngredient(t.Name, KindFX, f.Name, f.InspiredBy, "reverb", "", f.Name))
	}
	return out
}

// primaryFX maps a device's own category/module name to the canonical primary
// feature tag. Empty means "derive from the name via the keyword dictionary".
func primaryFX(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "distortion", "od", "drive", "overdrive":
		return "drive"
	case "boost", "booster":
		return "boost"
	case "compressor", "comp":
		return "comp"
	case "gate", "ns", "noise suppressor":
		return "gate"
	case "equalizer", "eq":
		return "eq"
	case "filter":
		return "filter"
	case "wah":
		return "wah"
	case "pitch":
		return "pitch"
	case "delay", "echo":
		return "delay"
	case "reverb", "reverbs":
		return "reverb"
	case "modulation", "mod":
		return "mod"
	default:
		return ""
	}
}
