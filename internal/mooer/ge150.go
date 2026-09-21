package mooer

// ge150 is the classic (non-pro) Mooer GE150. It ships the same 55 amps and 26
// cabs as the GE200, and exchanges presets as JSON .mo files with the "GE150
// Preset" schema (see ge150json.go) — the format the GE150 Edit app reads and
// writes. Its effect list differs from the GE200's in two places (see
// ge150Effects): 19 mod types instead of 20, and 9 delays with a MOD delay.
func ge150() Model {
	return Model{
		Name:         "ge150",
		Display:      "Mooer GE150",
		FileExchange: true,
		FileExt:      ".mo",
		ModuleOrder:  append([]string(nil), ModuleOrder...),
		codec:        ge150JSONCodec{},
		Amps:         mooerAmps,
		Cabs:         mooerCabs,
		Effects:      ge150Effects(),
	}
}

// ge150Effects is the classic GE150's per-module effect list, derived from the
// GE150 Edit app's bundled preset.json. It shares the GE200's tables except
// for the two the GE150 numbers differently:
//
//   - mod: 19 types — no MONO PITCH (the GE200's 20th, append-only there).
//   - delay: 9 types — a MOD delay sits at index 4, shifting REVERSE and the
//     later delays up by one.
//
// The type index is what the preset stores, so a list with the wrong length or
// order would load the wrong model — the knobs would still parse, but the
// effect would not be the one the user picked.
func ge150Effects() map[string][]Item {
	out := make(map[string][]Item, len(mooerEffects))
	for k, v := range mooerEffects {
		out[k] = v
	}
	out["mod"] = mooerEffects["mod"][:19]
	delay := make([]Item, 0, len(mooerEffects["delay"])+1)
	delay = append(delay, mooerEffects["delay"][:4]...)
	delay = append(delay, Item{Name: "MOD", InspiredBy: "Modulated Delay"})
	delay = append(delay, mooerEffects["delay"][4:]...)
	out["delay"] = delay
	return out
}
