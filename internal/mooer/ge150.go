package mooer

// ge150 is the classic (non-pro) Mooer GE150. It ships the same 55 amps, 26
// cabs and 151 effects as the GE200, but has no USB preset transfer, so the
// only output is a printable setup card.
func ge150() Model {
	return Model{
		Name:         "ge150",
		Display:      "Mooer GE150",
		FileExchange: false,
		ModuleOrder:  append([]string(nil), ModuleOrder...),
		Amps:         mooerAmps,
		Cabs:         mooerCabs,
		Effects:      mooerEffects,
	}
}
