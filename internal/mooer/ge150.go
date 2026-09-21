package mooer

// ge150 is the classic (non-pro) Mooer GE150. It ships the same 55 amps, 26
// cabs and 151 effects as the GE200, and exchanges presets as JSON .mo files
// with the "GE150 Preset" schema (see ge150json.go) — the format the GE150
// Edit app reads and writes.
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
		Effects:      mooerEffects,
	}
}
