package gp200

import (
	"fmt"
	"strings"
)

// FXSpec is one effect to place in a block of the fixed eleven-block chain.
// Params holds raw float32 values keyed by the editor's parameter names (see
// ParamNames), e.g. {"gain": 60, "level": 50}.
type FXSpec struct {
	Slot    string // pre, wah, dst, amp, nr, cab, eq, mod, dly, rvb or vol
	Type    string // effect name within that module catalog
	Enabled bool
	Params  map[string]float32
}

// Spec is a tone to dial in on a GP-200: an amp, an optional cab, and any
// number of effects placed into their blocks.
type Spec struct {
	Name   string
	Tempo  uint16
	Volume uint8
	Amp    string // amp model name, or a real-hardware description to resolve
	Cab    string // cab model name or description (optional)
	FX     []FXSpec
}

// slotIndex maps a module name to its physical block position.
func slotIndex(module string) (int, bool) {
	for i, mod := range slotModules {
		if strings.EqualFold(mod, module) {
			return i, true
		}
	}
	return 0, false
}

// resolveEffect finds the code for a named effect. A slot hint narrows the
// search to that module's catalog first (needed for names like "Tube", which
// is both a DST overdrive and a DLY delay); without a hint the first match in
// catalog order wins.
func resolveEffect(slot, name string) (uint32, error) {
	q := strings.TrimSpace(name)
	if q == "" {
		return 0, fmt.Errorf("an effect name is required")
	}
	if slot != "" {
		for _, e := range ModuleEffects(slot) {
			if strings.EqualFold(e.Name, q) {
				return e.Code, nil
			}
		}
	}
	if code, ok := EffectCode(q); ok {
		return code, nil
	}
	return 0, fmt.Errorf("no GP-200 effect matches %q; list them with gp200_catalog_list_fx", q)
}

// resolveAmp finds the amp code for a query: exact model name first, then a
// case-insensitive substring match against the "inspired by" hardware.
func resolveAmp(query string) (uint32, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, fmt.Errorf("an amp is required")
	}
	for _, a := range Amps() {
		if strings.EqualFold(a.Name, q) {
			return a.Code, nil
		}
	}
	for _, a := range Amps() {
		if d := InspiredBy(a.Name); d != "" && normContains(d, q) {
			return a.Code, nil
		}
	}
	return 0, fmt.Errorf("no GP-200 amp matches %q; list them with gp200_catalog_list_amps", q)
}

// resolveCab finds the cab code for a query. An empty query is allowed (the
// cab block is left at its default).
func resolveCab(query string) (uint32, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, nil
	}
	for _, c := range Cabs() {
		if strings.EqualFold(c.Name, q) {
			return c.Code, nil
		}
	}
	for _, c := range Cabs() {
		if d := InspiredBy(c.Name); d != "" && normContains(d, q) {
			return c.Code, nil
		}
	}
	return 0, fmt.Errorf("no GP-200 cab matches %q; list them with gp200_catalog_list_cabs", q)
}

// normContains reports whether needle appears in haystack, ignoring case,
// trademark symbols, middot separators and extra whitespace, so "Marshall
// JCM800" matches "Marshall® JCM800" and "Mesa Boogie" matches "Mesa/Boogie®".
func normContains(haystack, needle string) bool {
	h := normalize(haystack)
	n := normalize(needle)
	return strings.Contains(h, n)
}

func normalize(s string) string {
	s = stripMarks(strings.ToLower(s))
	return strings.Join(strings.Fields(s), " ")
}

func stripMarks(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '®', '™', '©', '·', '/':
			return ' '
		}
		return r
	}, s)
}

// place loads a named effect into a physical block, starting from the
// effect's defaults and then applying any parameter overrides.
func place(b *Block, slot int, code uint32, enabled bool, overrides map[string]float32) {
	b.Slot = uint8(slot) // #nosec G115 -- slot is a validated 0..10 block index
	b.EffectID = code
	b.Enabled = enabled
	b.Params = DefaultParams(code)
	for name, value := range overrides {
		_ = SetParam(b, name, value)
	}
}

// BuildPreset resolves a Spec into a concrete Preset.
func BuildPreset(s Spec) (Preset, error) {
	p := New()
	if strings.TrimSpace(s.Name) != "" {
		p.PatchName = s.Name
	}
	if s.Tempo != 0 {
		p.Tempo = s.Tempo
	}
	if s.Volume != 0 {
		p.Volume = s.Volume
	}

	ampCode, err := resolveAmp(s.Amp)
	if err != nil {
		return p, err
	}
	place(&p.Blocks[3], 3, ampCode, true, nil)

	if s.Cab != "" {
		cabCode, err := resolveCab(s.Cab)
		if err != nil {
			return p, err
		}
		place(&p.Blocks[5], 5, cabCode, true, nil)
	}

	for _, f := range s.FX {
		slot, ok := slotIndex(f.Slot)
		if !ok {
			return p, fmt.Errorf("unknown GP-200 block %q (want pre, wah, dst, amp, nr, cab, eq, mod, dly, rvb or vol)", f.Slot)
		}
		code, err := resolveEffect(f.Slot, f.Type)
		if err != nil {
			return p, err
		}
		place(&p.Blocks[slot], slot, code, f.Enabled, f.Params)
	}
	return p, nil
}
