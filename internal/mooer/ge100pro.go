package mooer

// ge100pro is the Mooer GE100 Pro. It is a newer, larger device (72 amps, 67
// cabs and 240+ effects); the tables below are the core models from its
// published mapping. It supports preset transfer via USB.
func ge100pro() Model {
	return Model{
		Name:         "ge100pro",
		Display:      "Mooer GE100 Pro",
		FileExchange: true,
		FileExt:      ".mo",
		ModuleOrder:  append([]string(nil), ModuleOrder...),
		Amps: items(
			[2]string{"65 US TWIN", "Fender '65 Twin Reverb"},
			[2]string{"59 US BASS", "Fender '59 Bassman"},
			[2]string{"US SONIC", "Fender Super Reverb"},
			[2]string{"UK 30", "Vox AC30 Top Boost"},
			[2]string{"JAZZ 120", "Roland JC-120"},
			[2]string{"UK 800", "Marshall JCM800"},
			[2]string{"UK 900", "Marshall JCM900"},
			[2]string{"PLX 100", "Marshall Plexi Super Lead 100"},
			[2]string{"US RECO", "Mesa Boogie Dual Rectifier"},
			[2]string{"US MARK III", "Mesa Boogie Mark III"},
			[2]string{"E650 CL", "Engl Powerball / Blackmore"},
			[2]string{"EDDY 5150", "Peavey / EVH 5150"},
			[2]string{"CITRUS 30", "Orange AD30"},
			[2]string{"BE 100", "Friedman BE-100"},
			[2]string{"GAS STATION", "Diezel Hagen / Herbert"},
		),
		Cabs: items(
			[2]string{"US TWIN 212", "Fender 2x12 Open Back (Jensen C12K)"},
			[2]string{"UK 412 V", "Marshall 4x12 Closed Back (Celestion V30)"},
			[2]string{"CITRUS 412", "Orange 4x12 (Celestion V30)"},
			[2]string{"US RECO 412", "Mesa Boogie Rectifier 4x12 (Celestion V30)"},
			[2]string{"UK 212 V", "Vox 2x12 Open Back (Celestion Alnico Blue)"},
		),
		Effects: map[string][]Item{
			"od": items(
				[2]string{"808 / TS9", "Ibanez Tube Screamer"},
				[2]string{"PURE OD", "Mooer Pure Boost (Xotic RC Booster)"},
				[2]string{"FLEX OD", "Mooer Flex Boost (Xotic AC Booster)"},
				[2]string{"JUICER", "Mooer Juicer"},
				[2]string{"BLACK RAT", "ProCo Rat"},
				[2]string{"MOD MODERN", "BOSS DS-1"},
				[2]string{"MUFF FUZZ", "EHX Big Muff"},
				[2]string{"METAL ZONE", "BOSS MT-2 Metal Zone"},
				[2]string{"GW VAMP", "Digitech Whammy"},
			),
		},
	}
}
