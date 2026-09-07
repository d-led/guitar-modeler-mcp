package catalog

// amps is the complete list of amplifier models available on the HeadRush
// Gigboard, mapped to the real hardware they emulate. The mapping is the
// "translation layer" encoded from the device backup and the Gigboard Hints
// community table. Where the emulated amplifier is not publicly documented the
// Brand/RealModel fields are left empty rather than guessed.
var amps = []Amp{
	// Orange
	{Model: "05 Tangerine 30 Ch1", Brand: "Orange", RealModel: "AD30 Twin Channel", Channel: "Ch1", Wattage: "30W", Style: []string{"rock", "indie", "classic rock"}, Description: "Class A British twin-channel 30W head, channel 1."},
	{Model: "05 Tangerine 30 Ch2", Brand: "Orange", RealModel: "AD30 Twin Channel", Channel: "Ch2", Wattage: "30W", Style: []string{"rock", "indie", "classic rock"}, Description: "Class A British twin-channel 30W head, channel 2."},

	// Engl
	{Model: "11 EPB II Clean", Brand: "Engl", RealModel: "Powerball II", Channel: "Clean", Wattage: "100W", Style: []string{"metal", "high-gain"}, Description: "Engl Powerball II clean channel."},
	{Model: "11 EPB II Crunch", Brand: "Engl", RealModel: "Powerball II", Channel: "Crunch", Wattage: "100W", Style: []string{"metal", "high-gain"}, Description: "Engl Powerball II crunch channel."},
	{Model: "11 EPB II Hi-Lead", Brand: "Engl", RealModel: "Powerball II", Channel: "Hi-Lead", Wattage: "100W", Style: []string{"metal", "high-gain"}, Description: "Engl Powerball II high-gain lead channel."},
	{Model: "11 EPB II Lo-Lead", Brand: "Engl", RealModel: "Powerball II", Channel: "Lo-Lead", Wattage: "100W", Style: []string{"metal", "high-gain"}, Description: "Engl Powerball II low-gain lead channel."},

	// Bass
	{Model: "17 Trace Elliot Elf", Brand: "Trace Elliot", RealModel: "ELF", Wattage: "200W", Style: []string{"bass"}, Bass: true, Description: "Trace Elliot ELF micro bass head."},
	{Model: "66 Flip Bass", Brand: "Ampeg", RealModel: "B-15 Portaflex", Wattage: "30W", Style: []string{"bass"}, Bass: true, Description: "Flip-top portaflex tube bass amp."},
	{Model: "69 Blue Line Bass", Brand: "Ampeg", RealModel: "SVT", Wattage: "300W", Style: []string{"bass", "rock"}, Bass: true, Description: "Ampeg SVT 300W all-tube bass head."},
	{Model: "69 Blue Line Scoop", Brand: "Ampeg", RealModel: "SVT (scooped)", Wattage: "300W", Style: []string{"bass", "rock"}, Bass: true, Description: "Ampeg SVT voicing with a scooped midrange."},

	// Fender
	{Model: "59 Deluxe Gain Mod", Brand: "Fender", RealModel: "Tweed Deluxe (modded)", Wattage: "12W", Style: []string{"blues", "rock", "country"}, Description: "Modded 5E3 Tweed Deluxe with extra gain."},
	{Model: "59 Tweed Bass", Brand: "Fender", RealModel: "Tweed Bassman", Wattage: "50W", Style: []string{"blues", "rock", "country"}, Description: "Fender 5F6-A Tweed Bassman."},
	{Model: "59 Tweed Deluxe", Brand: "Fender", RealModel: "Tweed Deluxe", Wattage: "12W", Style: []string{"blues", "country"}, Description: "Fender 5E3 Tweed Deluxe combo."},
	{Model: "59 Tweed Prince", Brand: "Fender", RealModel: "Tweed Princeton", Wattage: "5W", Style: []string{"blues", "country"}, Description: "Fender 5F2-A Tweed Princeton."},
	{Model: "64 Black Lux Norm", Brand: "Fender", RealModel: "Deluxe Reverb", Channel: "Normal", Wattage: "22W", Style: []string{"clean", "blues", "country"}, Description: "Blackface Deluxe Reverb, normal channel."},
	{Model: "64 Black Lux Vib", Brand: "Fender", RealModel: "Deluxe Reverb", Channel: "Vibrato", Wattage: "22W", Style: []string{"clean", "blues", "country"}, Description: "Blackface Deluxe Reverb, vibrato channel."},
	{Model: "64 Black Vib", Brand: "Fender", RealModel: "Vibroverb", Wattage: "40W", Style: []string{"blues", "surf"}, Description: "Blackface Vibroverb combo."},
	{Model: "65 Black Mini", Brand: "Fender", RealModel: "Champ", Wattage: "6W", Style: []string{"blues", "classic rock"}, Description: "Blackface Champ practice combo."},
	{Model: "65 Black Prince", Brand: "Fender", RealModel: "Princeton", Wattage: "12W", Style: []string{"clean", "country"}, Description: "Blackface Princeton combo."},
	{Model: "65 Black Prince Rev", Brand: "Fender", RealModel: "Princeton Reverb", Wattage: "12W", Style: []string{"clean", "country", "blues"}, Description: "Blackface Princeton Reverb combo."},
	{Model: "65 Black SR", Brand: "Fender", RealModel: "Super Reverb", Wattage: "40W", Style: []string{"blues", "clean", "country"}, Description: "Blackface Super Reverb 4x10 combo."},
	{Model: "67 Black Duo", Brand: "Fender", RealModel: "Twin Reverb", Wattage: "85W", Style: []string{"clean"}, Description: "Blackface Twin Reverb combo."},
	{Model: "67 Black Shimmer", Brand: "Fender", RealModel: "Twin Reverb (shimmer)", Wattage: "85W", Style: []string{"clean", "ambient"}, Description: "Blackface Twin Reverb voicing with added shimmer."},

	// Marshall
	{Model: "65 J45", Brand: "Marshall", RealModel: "JTM45", Wattage: "30W", Style: []string{"blues", "rock"}, Description: "Marshall JTM45 head."},
	{Model: "67 Plexiglas Vari", Brand: "Marshall", RealModel: "Super Lead Plexi (Variac)", Wattage: "100W", Style: []string{"blues", "rock"}, Description: "Marshall Super Lead Plexi head, variac mod."},
	{Model: "68 Plexi EL84 Mod", Brand: "Marshall", RealModel: "Super Lead (EL84 mod)", Wattage: "50W", Style: []string{"rock", "crunch"}, Description: "Marshall Plexi voiced for EL84 power tubes."},
	{Model: "68 Plexiglas 50W", Brand: "Marshall", RealModel: "Super Lead 1987", Wattage: "50W", Style: []string{"crunch", "rock", "blues"}, Description: "Marshall 1987 50W Super Lead Plexi head."},
	{Model: "69 Plexiglas 100W", Brand: "Marshall", RealModel: "Super Lead 1959", Wattage: "100W", Style: []string{"rock", "blues"}, Description: "Marshall 1959 100W Super Lead Plexi head."},
	{Model: "82 Lead 800 50W", Brand: "Marshall", RealModel: "JCM800 2204", Wattage: "50W", Style: []string{"rock", "hard rock"}, Description: "Marshall JCM800 2204 50W head."},
	{Model: "82 Lead 800 100W", Brand: "Marshall", RealModel: "JCM800 2203", Wattage: "100W", Style: []string{"rock", "hard rock"}, Description: "Marshall JCM800 2203 100W head."},
	{Model: "82 Lead 800 Bright", Brand: "Marshall", RealModel: "JCM800 (bright)", Wattage: "100W", Style: []string{"rock", "hard rock"}, Description: "Marshall JCM800 voicing with a brighter top end."},
	{Model: "82 Lead 800 Bass Mod", Brand: "Marshall", RealModel: "JCM800 (bass mod)", Wattage: "100W", Style: []string{"rock"}, Description: "Marshall JCM800 voicing with extended low end."},
	{Model: "82 Lead 800 TS Mod", Brand: "Marshall", RealModel: "JCM800 (TS mod)", Wattage: "100W", Style: []string{"rock", "metal"}, Description: "Marshall JCM800 voicing with a tube-screamer style mid boost."},

	// Mesa Boogie
	{Model: "85 M-2 Lead", Brand: "Mesa Boogie", RealModel: "Mark IIC+", Channel: "Lead", Wattage: "75W", Style: []string{"rock", "metal"}, Description: "Mesa Boogie Mark IIC+ drive channel."},
	{Model: "85 M-2 Lead Cap Mod", Brand: "Mesa Boogie", RealModel: "Mark IIC+ (cap mod)", Channel: "Lead", Wattage: "75W", Style: []string{"rock", "metal"}, Description: "Mesa Boogie Mark IIC+ with the capacitor mod."},
	{Model: "92 Treadplate Modern", Brand: "Mesa Boogie", RealModel: "Dual Rectifier", Channel: "Red (Modern)", Wattage: "100W", Style: []string{"metal", "hard rock"}, Description: "Mesa Boogie Dual Rectifier, red/modern channel."},
	{Model: "92 Treadplate Raw", Brand: "Mesa Boogie", RealModel: "Dual Rectifier", Channel: "Raw", Wattage: "100W", Style: []string{"metal", "hard rock"}, Description: "Mesa Boogie Dual Rectifier, raw voicing."},
	{Model: "92 Treadplate Vintage", Brand: "Mesa Boogie", RealModel: "Dual Rectifier", Channel: "Orange (Vintage)", Wattage: "100W", Style: []string{"metal", "hard rock"}, Description: "Mesa Boogie Dual Rectifier, orange/vintage channel."},

	// Soldano
	{Model: "89 SL-100 Clean", Brand: "Soldano", RealModel: "SLO-100", Channel: "Clean", Wattage: "100W", Style: []string{"rock", "hard rock"}, Description: "Soldano SLO-100 clean channel."},
	{Model: "89 SL-100 Crunch", Brand: "Soldano", RealModel: "SLO-100", Channel: "Crunch", Wattage: "100W", Style: []string{"rock", "hard rock"}, Description: "Soldano SLO-100 crunch channel."},
	{Model: "89 SL-100 Drive", Brand: "Soldano", RealModel: "SLO-100", Channel: "Overdrive", Wattage: "100W", Style: []string{"rock", "hard rock", "metal"}, Description: "Soldano SLO-100 super lead overdrive channel."},
	{Model: "89 SL-100 Ext Range", Brand: "Soldano", RealModel: "SLO-100", Channel: "Extended range", Wattage: "100W", Style: []string{"rock", "metal"}, Description: "Soldano SLO-100 with extended frequency range."},

	// Boutique / other
	{Model: "93 MS30", Brand: "Matchless", RealModel: "DC30", Wattage: "30W", Style: []string{"clean", "country", "rock"}, Description: "Class A boutique 30W combo voiced after the Matchless DC30."},

	// Bogner
	{Model: "97 RB-01b Blue", Brand: "Bogner", RealModel: "Ecstasy 101B", Channel: "Blue", Wattage: "100W", Style: []string{"rock", "blues"}, Description: "Bogner Ecstasy 101B blue channel."},
	{Model: "97 RB-01b Green", Brand: "Bogner", RealModel: "Ecstasy 101B", Channel: "Green", Wattage: "100W", Style: []string{"rock", "hard rock"}, Description: "Bogner Ecstasy 101B green channel."},
	{Model: "97 RB-01b Red", Brand: "Bogner", RealModel: "Ecstasy 101B", Channel: "Red", Wattage: "100W", Style: []string{"rock", "hard rock", "metal"}, Description: "Bogner Ecstasy 101B red channel."},

	// Peavey
	{Model: "99 PV51 II Clean", Brand: "Peavey", RealModel: "5150 II", Channel: "Clean", Wattage: "120W", Style: []string{"metal", "hard rock"}, Description: "Peavey 5150 II clean channel."},
	{Model: "99 PV51 II Crunch", Brand: "Peavey", RealModel: "5150 II", Channel: "Crunch", Wattage: "120W", Style: []string{"metal", "hard rock"}, Description: "Peavey 5150 II crunch channel."},
	{Model: "99 PV51 II Lead", Brand: "Peavey", RealModel: "5150 II", Channel: "Lead", Wattage: "120W", Style: []string{"metal", "hard rock"}, Description: "Peavey 5150 II lead channel."},

	// Undocumented models — kept for completeness, brand intentionally empty.
	{Model: "83 400R", RealModel: "400R", Style: []string{"rock"}, Description: "High-gain British-style head."},
	{Model: "84 J-120H", RealModel: "J-120H", Style: []string{"rock"}, Description: "120W British-style head."},
}
