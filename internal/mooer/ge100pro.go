package mooer

// ge100pro is the Mooer GE100 Pro. Its ten-slot chain and .mo preset format are
// its own: a preset is a frame dump rather than a record (see ge100procodec.go),
// and its model lists come from the tables Mooer Studio For GE100 Pro ships
// (see ge100pro_catalog_data.go). It supports preset transfer via USB.
func ge100pro() Model {
	return Model{
		Name:         "ge100pro",
		Display:      "Mooer GE100 Pro",
		FileExchange: true,
		FileExt:      ".mo",
		ModuleOrder:  append([]string(nil), ModuleOrder...),
		codec:        ge100ProCodec{},
		Amps:         ge100ProAmpItems(),
		Cabs:         ge100ProCabItems(),
		Effects:      ge100ProEffectItems(),
	}
}

// ge100ProAmpItems pairs the device's amp models with the hardware we can name.
// The editor records modelled hardware for its effects but not for its amps, so
// the amp hints are our own reading of Mooer's model names.
func ge100ProAmpItems() []Item {
	items := make([]Item, len(ge100ProAmpNames))
	for i, name := range ge100ProAmpNames {
		items[i] = Item{Name: name, InspiredBy: ge100ProAmpHints[name]}
	}
	return items
}

// ge100ProCabItems names the cabinet models. The editor does not record the
// hardware behind a cab either.
func ge100ProCabItems() []Item {
	items := make([]Item, len(ge100ProCabNames))
	for i, name := range ge100ProCabNames {
		items[i] = Item{Name: name, InspiredBy: ge100ProCabHints[name]}
	}
	return items
}

// ge100ProEffectItems flattens the generated per-module model lists into the
// catalog items the shared device API uses. The modelled hardware comes from the
// editor itself, which labels each effect with its reference.
func ge100ProEffectItems() map[string][]Item {
	effects := make(map[string][]Item, len(ge100ProEffects))
	for module, models := range ge100ProEffects {
		items := make([]Item, len(models))
		for i, model := range models {
			items[i] = Item{Name: model.Name, InspiredBy: model.InspiredBy}
		}
		effects[module] = items
	}
	return effects
}

// ge100ProAmpHints names the real amplifier behind the amp models we can
// identify from Mooer's naming. A model we cannot place stays unnamed rather
// than being guessed at.
var ge100ProAmpHints = map[string]string{
	"65 US DX":       "Fender '65 Deluxe Reverb",
	"65 US TW":       "Fender '65 Twin Reverb",
	"62 US DLX":      "Fender '62 Deluxe (brownface)",
	"55 US TD":       "Fender '55 Tweed Deluxe",
	"59 US BASS":     "Fender '59 Bassman",
	"US SONIC":       "Fender Super-Sonic",
	"US BLUES CL":    "Fender Blues Deluxe (clean)",
	"US BLUES OD":    "Fender Blues Deluxe (drive)",
	"MATCHBOX 30 CL": "Matchless DC-30 (clean)",
	"MATCHBOX 30 OD": "Matchless DC-30 (drive)",
	"J800":           "Marshall JCM800",
	"J900":           "Marshall JCM900",
	"PLX 100":        "Marshall Plexi Super Lead 100",
	"E650 CL":        "Engl E650 (clean)",
	"E650 DS":        "Engl E650 (lead)",
	"MARKIII CL":     "Mesa Boogie Mark III (clean)",
	"MARKIII DS":     "Mesa Boogie Mark III (lead)",
	"MARKV CL":       "Mesa Boogie Mark V (clean)",
	"MARKV DS":       "Mesa Boogie Mark V (lead)",
	"TRI REC CL":     "Mesa Boogie Dual Rectifier (clean)",
	"TRI REC DS":     "Mesa Boogie Dual Rectifier (lead)",
	"ROCK VRB CL":    "Orange Rockerverb (clean)",
	"ROCK VRB DS":    "Orange Rockerverb (dirty)",
	"CITRUS 30":      "Orange AD30",
	"CITRUS 50":      "Orange Rockerverb 50",
	"SLOW 100 CR":    "Soldano SLO-100 (crunch)",
	"SLOW 100 DS":    "Soldano SLO-100 (lead)",
	"DR.ZEE 18 JR":   "Dr. Z MAZ 18 Jr",
	"JET 100H CL":    "Jet City JCA100H (clean)",
	"JET 100H OD":    "Jet City JCA100H (drive)",
	"JAZZ 120":       "Roland JC-120",
	"UK30 CL":        "Vox AC30 Top Boost (clean)",
	"UK30 OD":        "Vox AC30 Top Boost (drive)",
	"HWT 103":        "Hiwatt DR103",
	"PV 5050 CL":     "Peavey 5150 (clean)",
	"PV 5050 DS":     "Peavey 5150 (lead)",
	"EV 5050 CL":     "EVH 5150 III (clean)",
	"EV 5050 DS":     "EVH 5150 III (lead)",
	"HT CLUB CL":     "Blackstar HT Club (clean)",
	"HT CLUB DS":     "Blackstar HT Club (drive)",
	"KOCHE OD":       "Koch (drive)",
	"KOCHE DS":       "Koch (distortion)",
	"CAROL CL":       "Carol-Ann (clean)",
	"CAROL OD":       "Carol-Ann (drive)",
	"AMPOG B18 CL":   "Ampeg B-18 (clean)",
	"AMPOG B18 DS":   "Ampeg B-18 (drive)",
	"AMPOG SVT 4":    "Ampeg SVT-4",
	"AKUILA 750":     "Aguilar DB750",
	"MVRKBASS 500":   "Markbass Little Mark 500",
	"KARVIN B1500":   "Carvin B1500",
}

// ge100ProCabHints names the cabinets behind the models Mooer's own naming makes
// clear (the CT- prefix marks the classic-voiced "custom" range).
var ge100ProCabHints = map[string]string{
	"US DLX 112":    "Fender Deluxe 1x12",
	"US TWN 212":    "Fender Twin 2x12",
	"US BASS 410":   "Fender Bassman 4x10",
	"CT-BRIT 412":   "Marshall 4x12 (Celestion)",
	"CT-BOG OS 412": "Bogner Oversized 4x12",
	"CT-VOCS 212":   "Vox 2x12",
	"CT-FRAM 212":   "Framus 2x12",
}
