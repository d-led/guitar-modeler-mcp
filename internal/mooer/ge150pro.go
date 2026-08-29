package mooer

// ge150pro is the Mooer GE150 Pro Li / GE150 Max. This is the device the .mo
// preset format was reverse-engineered from, so it supports file exchange.
// Its model tables live in the package-level Amps/Cabs/Effects and the
// ampInspiredBy/cabInspiredBy/fxInspiredBy tables.
func ge150pro() Model {
	return Model{
		Name:         "ge150pro",
		Display:      "Mooer GE150 Pro Li / GE150 Max",
		FileExchange: true,
		FileExt:      ".mo",
		ModuleOrder:  append([]string(nil), ModuleOrder...),
		Amps:         buildItems(Amps, ampInspiredBy),
		Cabs:         buildItems(Cabs, cabInspiredBy),
		Effects:      buildEffects(Effects, fxInspiredBy),
	}
}

func buildItems(names []string, inspired map[string]string) []Item {
	out := make([]Item, len(names))
	for i, name := range names {
		out[i] = Item{Name: name, InspiredBy: inspired[name]}
	}
	return out
}

func buildEffects(effects map[string][]string, inspired map[string]map[string]string) map[string][]Item {
	out := make(map[string][]Item, len(effects))
	for module, names := range effects {
		items := make([]Item, len(names))
		for i, name := range names {
			items[i] = Item{Name: name, InspiredBy: inspired[module][name]}
		}
		out[module] = items
	}
	return out
}
