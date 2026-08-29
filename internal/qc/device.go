// Package qc is the device backend for the Neural DSP Quad Cortex. The model
// catalog is transcribed from the device's own ModelRepo.xml (via the
// OpenCortex project), which pairs each on-screen model name with the real
// hardware it is "Based on". The Quad Cortex presets are encrypted protobufs,
// so file exchange is not implemented yet; this package supplies the catalog
// and the translation layer so real-world hardware can be mapped to QC models
// and a preset can be recommended from standard components.
package qc

// Item is one selectable model: the on-device name and the real hardware it
// emulates (empty when not documented).
type Item struct {
	Name       string `json:"name"`
	InspiredBy string `json:"inspired_by,omitempty"`
}

// Device describes the Quad Cortex and its model catalog. The QC uses a free
// 4-lane signal grid (each lane a serial chain that can split and merge),
// rather than a fixed module order, so there is no ModuleOrder here.
type Device struct {
	Name         string
	Display      string
	FileExchange bool
	FileExt      string
	// Topology describes the grid layout for the agent guide.
	Topology string
	// Amps is the guitar amp list.
	Amps []Item
	// BassAmps is the bass amp list.
	BassAmps []Item
	// Cabs is the guitar cabinet (cabsim) list.
	Cabs []Item
	// Effects maps a category to its models: drive, compressor, equalizer,
	// delay, modulation, reverb, wah, pitch, filter and gate.
	Effects map[string][]Item
}

// Default returns the Quad Cortex device model.
func Default() Device {
	return Device{
		Name:         "quad-cortex",
		Display:      "Neural DSP Quad Cortex",
		FileExchange: false, // encrypted protobuf presets are not implemented yet
		FileExt:      "",    // no file exchange yet
		Topology:     "free 4-lane grid; each lane is a serial chain that can split and merge",
		Amps:         guitarAmps,
		BassAmps:     bassAmps,
		Cabs:         guitarCabs,
		Effects: map[string][]Item{
			"drive":      guitarDrive,
			"compressor": compressors,
			"equalizer":  equalizers,
			"delay":      delays,
			"modulation": modulation,
			"reverb":     reverbs,
			"wah":        wahs,
			"pitch":      pitch,
			"filter":     filters,
			"gate":       gates,
		},
	}
}

func items(pairs ...[2]string) []Item {
	out := make([]Item, len(pairs))
	for i, p := range pairs {
		out[i] = Item{Name: p[0], InspiredBy: p[1]}
	}
	return out
}

// guitarAmps is the "Guitar Amplifier" category, in ModelRepo order.
var guitarAmps = items(
	[2]string{"Marshall JCM800", "Marshall JCM800"},
	[2]string{"Peavey 6505 - Lead Channel", "Peavey 6505 lead channel"},
	[2]string{"Roland Jazz Chorus 120", "Roland JC-120"},
	[2]string{"Vox AC30 Normal", "Vox AC30"},
	[2]string{"Vox AC30 Top Boost", "Vox AC30 top boost"},
	[2]string{"EVH 5150 III 100S Red EL34", "EVH 5150 III 100S red channel (EL34)"},
	[2]string{"Mesa Boogie Dual Rectifier - Channel 3 Modern", "Mesa Boogie Dual Rectifier Ch3 Modern"},
	[2]string{"Mesa Boogie Dual Rectifier - Channel 3 Raw", "Mesa Boogie Dual Rectifier Ch3 Raw"},
	[2]string{"Mesa Boogie Dual Rectifier - Channel 3 Vintage", "Mesa Boogie Dual Rectifier Ch3 Vintage"},
	[2]string{"Friedman HBE100 - Clean Channel", "Friedman HBE100 clean channel"},
	[2]string{"Friedman HBE100 - BE Channel", "Friedman HBE100 BE channel"},
	[2]string{"Friedman HBE100 - HBE Channel", "Friedman HBE100 HBE channel"},
	[2]string{"Marshall JTM 45 - Normal Channel", "Marshall JTM45 normal channel"},
	[2]string{"Marshall JTM 45 - High Treble Channel", "Marshall JTM45 high treble channel"},
	[2]string{"Marshall JTM 45 - Jumpered Channel", "Marshall JTM45 jumpered"},
	[2]string{"Marshall JCM900 4100 - Clean", "Marshall JCM900 4100 clean"},
	[2]string{"Marshall JCM900 4100 - Lead", "Marshall JCM900 4100 lead"},
	[2]string{"Soldano SLO-100 - Lead Channel", "Soldano SLO-100 lead channel"},
	[2]string{"Soldano SLO-100 - Crunch Bright Channel", "Soldano SLO-100 crunch bright channel"},
	[2]string{"Soldano SLO-100 - Crunch Normal Channel", "Soldano SLO-100 crunch normal channel"},
	[2]string{"Hiwatt DR103 - Normal Channel", "Hiwatt DR103 normal channel"},
	[2]string{"Hiwatt DR103 - Bright Channel", "Hiwatt DR103 bright channel"},
	[2]string{"Marshall Lead 50 - Bright", "Marshall JMP50 lead bright"},
	[2]string{"Marshall Lead 50 - Jumped", "Marshall JMP50 lead jumped"},
	[2]string{"Marshall Lead 50 - Normal", "Marshall JMP50 lead normal"},
	[2]string{"Fender Twin Reverb", "Fender Twin Reverb"},
	[2]string{"Fender Twin Reverb - Vibrato", "Fender Twin Reverb vibrato"},
	[2]string{"Mesa Boogie Trem-O-Verb - Orange", "Mesa Boogie Trem-O-Verb orange"},
	[2]string{"Mesa Boogie Trem-O-Verb - Red", "Mesa Boogie Trem-O-Verb red"},
	[2]string{"Marshall Super Lead 100 - Bright", "Marshall Super Lead 100 bright"},
	[2]string{"Marshall Super Lead 100 - Jumped", "Marshall Super Lead 100 jumped"},
	[2]string{"Marshall Super Lead 100 - Normal", "Marshall Super Lead 100 normal"},
	[2]string{"Fender Blackface Deluxe Reverb - Normal", "Fender Blackface Deluxe Reverb normal"},
	[2]string{"Fender Blackface Deluxe Reverb - Vibrato", "Fender Blackface Deluxe Reverb vibrato"},
	[2]string{"Fender Super Reverb '65 - Normal", "Fender Super Reverb '65 normal"},
	[2]string{"Fender Super Reverb '65 - Vibrato", "Fender Super Reverb '65 vibrato"},
	[2]string{"EVH 5150 III 100S Blue EL34", "EVH 5150 III 100S blue channel (EL34)"},
	[2]string{"Mesa Boogie Lone Star Clean 50W Normal", "Mesa Boogie Lone Star clean"},
	[2]string{"Mesa Boogie Lone Star Clean 100W Normal", "Mesa Boogie Lone Star clean"},
	[2]string{"Mesa Boogie Lone Star Drive 50W Normal", "Mesa Boogie Lone Star drive"},
	[2]string{"Mesa Boogie Lone Star Drive 100W Normal", "Mesa Boogie Lone Star drive"},
	[2]string{"Diezel VH4 - Ch1 Bright", "Diezel VH4 channel 1 bright"},
	[2]string{"Diezel VH4 - Ch1 Normal", "Diezel VH4 channel 1 normal"},
	[2]string{"Diezel VH4 - Ch2 Bright", "Diezel VH4 channel 2 bright"},
	[2]string{"Diezel VH4 - Ch3", "Diezel VH4 channel 3"},
	[2]string{"Diezel VH4 - Ch4", "Diezel VH4 channel 4"},
	[2]string{"Peavey 6505 - Rhythm Channel", "Peavey 6505 rhythm channel"},
	[2]string{"Morgan SW50 - Clean", "Morgan SW50 clean"},
	[2]string{"Bogner Shiva 20th Anniversary - Clean", "Bogner Shiva clean"},
	[2]string{"EVH 5150 Blue 6L6", "EVH 5150 III 6L6 blue"},
	[2]string{"EVH 5150 Red 6L6", "EVH 5150 III 6L6 red"},
	[2]string{"Fender Princeton Reverb", "Fender Princeton Reverb"},
	[2]string{"Marshall Silver Jubilee", "Marshall Silver Jubilee"},
	[2]string{"Marshall Silver Jubilee Tight", "Marshall Silver Jubilee"},
	[2]string{"Fender 59 Bassman Bright Jumped", "Fender '59 Bassman bright jumped"},
	[2]string{"Fender 59 Bassman Bright", "Fender '59 Bassman bright"},
	[2]string{"Fender 59 Bassman Normal Jumped", "Fender '59 Bassman normal jumped"},
	[2]string{"Fender 59 Bassman Normal", "Fender '59 Bassman normal"},
	[2]string{"Vox AC15 Normal", "Vox AC15 normal"},
	[2]string{"Vox AC15 Boost", "Vox AC15 top boost"},
	[2]string{"Bogner Überschall Rev. Blue - Clean", "Bogner Uberschall clean"},
	[2]string{"Bogner Überschall Rev. Blue - Lead", "Bogner Uberschall lead"},
	[2]string{"Fender High Power Tweed Twin 5F8-A - Normal", "Fender High Power Tweed Twin normal"},
	[2]string{"Fender High Power Tweed Twin 5F8-A - Bright", "Fender High Power Tweed Twin bright"},
	[2]string{"Diezel Herbert - Ch1", "Diezel Herbert channel 1"},
	[2]string{"Diezel Herbert - Ch2", "Diezel Herbert channel 2"},
	[2]string{"Diezel Herbert - Ch3", "Diezel Herbert channel 3"},
	[2]string{"Mesa Boogie JP-2C - CH1", "Mesa Boogie JP-2C channel 1"},
	[2]string{"Mesa Boogie JP-2C - CH2", "Mesa Boogie JP-2C channel 2"},
	[2]string{"Mesa Boogie JP-2C - CH3", "Mesa Boogie JP-2C channel 3"},
	[2]string{"Matchless Chieftain", "Matchless Chieftain"},
	[2]string{"Matchless C-30", "Matchless C-30"},
	[2]string{"Victory Kraken - Ch1", "Victory Kraken channel 1"},
	[2]string{"Victory Kraken - Ch2", "Victory Kraken channel 2"},
)

// bassAmps is the "Bass Amplifier" category (subset of the full list).
var bassAmps = items(
	[2]string{"Marshall Super Bass 50 - Bright", "Marshall Super Bass 50 bright"},
	[2]string{"Marshall Super Bass 50 - Jumped", "Marshall Super Bass 50 jumped"},
	[2]string{"Marshall Super Bass 50 - Normal", "Marshall Super Bass 50 normal"},
	[2]string{"Gallien Krueger 800RB", "Gallien Krueger 800RB"},
	[2]string{"Mesa Boogie Bass 400+ - Ch1", "Mesa Boogie Bass 400+ channel 1"},
	[2]string{"Mesa Boogie Bass 400+ - Ch2", "Mesa Boogie Bass 400+ channel 2"},
	[2]string{"Ampeg Heritage SVT-CL", "Ampeg Heritage SVT-CL"},
	[2]string{"Ampeg Heritage B15N", "Ampeg Heritage B15N"},
	[2]string{"Hiwatt DR103 - Normal Channel Bass Mod", "Hiwatt DR103 bass mod normal"},
	[2]string{"Hiwatt DR103 - Bright Channel Bass Mod", "Hiwatt DR103 bass mod bright"},
)

// guitarCabs is the "Cabsim Guitar" category, in ModelRepo order.
var guitarCabs = items(
	[2]string{"Mesa Standard OS Straight V30", "Mesa Standard OS 4x12 with Celestion Vintage 30"},
	[2]string{"Diezel Front Loaded V30", "Diezel 4x12 with Celestion Vintage 30"},
	[2]string{"EVH Straight G12EVH", "EVH 4x12 with Celestion G12EVH"},
	[2]string{"Fender Cab A-Type 12", "Fender cab with Celestion A-Type 12"},
	[2]string{"Marshall 1960B Greenback", "Marshall 1960B with Celestion Greenback"},
	[2]string{"Marshall 1960B Pulsonic Greenback", "Marshall 1960B with Celestion Pulsonic Greenback"},
	[2]string{"Mesa Traditional Straight V30", "Mesa Traditional 4x12 with Celestion Vintage 30"},
	[2]string{"Zilla Cab Creamback G12H-75", "Zilla cab with Celestion Creamback G12H-75"},
	[2]string{"Marshall 2551B", "Marshall 2551B"},
	[2]string{"Orange PPC412 V30", "Orange PPC412 with Celestion Vintage 30"},
	[2]string{"Bogner Ubercab T75", "Bogner Ubercab with Celestion T75"},
	[2]string{"Suhr Cab V-Type", "Suhr cab with Celestion V-Type"},
	[2]string{"VOX AC30 Pre-Rola Greenback", "Vox AC30 cab with Celestion Pre-Rola Greenback"},
	[2]string{"Zilla Custom V30", "Zilla cab with Celestion Vintage 30"},
	[2]string{"ENGL V30", "Engl cab with Celestion Vintage 30"},
	[2]string{"Fender Deluxe Blackface C12K", "Fender Deluxe Blackface with Jensen C12K"},
	[2]string{"Fender Deluxe Tweed WGS G12Q", "Fender Deluxe Tweed with WGS G12Q"},
	[2]string{"Zilla Mini Modern Redback", "Zilla Mini Modern with Celestion G12H150 Redback"},
	[2]string{"Fender Tremolux Alnico", "Fender Tremolux with Oxford Alnico"},
	[2]string{"Mesa Rectifier V30", "Mesa Rectifier with Celestion Vintage 30"},
	[2]string{"Fender Twin Reverb C12Q", "Fender Twin Reverb with Jensen C12Q"},
	[2]string{"Roland JC-120", "Roland JC-120 cab"},
	[2]string{"VOX AC30 Top Boost Silver Bell", "Vox AC30 Top Boost with Celestion Alnico Silver Bell"},
	[2]string{"Zilla Open Alnico Gold", "Zilla open with Celestion Alnico Gold"},
	[2]string{"Fender Princeton FatJimmy C1060", "Fender Princeton with FatJimmy C1060"},
	[2]string{"Marshall 1960A G12M25", "Marshall 1960A with Celestion G12M25"},
	[2]string{"Marshall 1960B V30", "Marshall 1960B with Celestion Vintage 30"},
	[2]string{"Marshall 1960TV G12M25", "Marshall 1960TV with Celestion G12M25"},
	[2]string{"Mesa Standard OS Angled V30", "Mesa Standard OS angled with Celestion Vintage 30"},
	[2]string{"Mesa Traditional Angled V30", "Mesa Traditional angled with Celestion Vintage 30"},
	[2]string{"Mesa Traditional Straight G12H30", "Mesa Traditional straight with Celestion G12H30"},
	[2]string{"Mesa Oversize Angle 2003 UK V30", "Mesa Oversize Angle 2003 with Celestion UK Vintage 30"},
	[2]string{"Fender Deluxe 1x12 GA-SC64", "Fender Deluxe 1x12 with Eminence GA-SC64"},
	[2]string{"Marshall 1935B Alnico Cream", "Marshall 1935B Alnico Cream"},
	[2]string{"Mesa Rectifier 2x12 Legend V12", "Mesa Rectifier 2x12 Legend V12"},
	[2]string{"Zilla Fatboy 2x12 2002 UK V30", "Zilla Fatboy 2x12 with Celestion UK Vintage 30"},
	[2]string{"Fender Twin Reverb 2x12 C12K-2", "Fender Twin Reverb 2x12 with Jensen C12K-2"},
	[2]string{"Hiwatt SE4123 4x12", "Hiwatt SE4123 4x12"},
	[2]string{"VOX AC15 Alnico Blue", "Vox AC15 with Celestion Alnico Blue"},
	[2]string{"Fender Bassman Tweed P10R", "Fender Bassman Tweed with Jensen P10R"},
	[2]string{"Fender Princeton C10R", "Fender Princeton with Jensen C10R"},
	[2]string{"Matchless Chieftain Signature", "Matchless Chieftain with signature drivers"},
	[2]string{"Matchless C-30 Signature", "Matchless C-30 with signature drivers"},
)

// guitarDrive is the "Guitar Overdrive" category.
var guitarDrive = items(
	[2]string{"Klon Centaur", "Klon Centaur"},
	[2]string{"Fulltone OCD", "Fulltone OCD"},
	[2]string{"DOD Overdrive Preamp 250", "DOD Overdrive Preamp 250"},
	[2]string{"ProCo Rat", "ProCo Rat"},
	[2]string{"Ibanez TS808", "Ibanez TS808"},
	[2]string{"Xotic BB Preamp", "Xotic BB Preamp"},
	[2]string{"Friedman BE-OD", "Friedman BE-OD"},
	[2]string{"Marshall BluesBreaker", "Marshall BluesBreaker"},
	[2]string{"BOSS DS-1", "BOSS DS-1"},
	[2]string{"Marshall Guv'nor", "Marshall Guv'nor"},
	[2]string{"BOSS MT-2", "BOSS Metal Zone MT-2"},
	[2]string{"BOSS OD-1", "BOSS OD-1"},
	[2]string{"BOSS SD-1", "BOSS SD-1"},
	[2]string{"Electro-Harmonix Big Muff Pi", "Electro-Harmonix Big Muff Pi"},
	[2]string{"BOSS BD-2", "BOSS BD-2"},
	[2]string{"Dallas Rangemaster", "Dallas Rangemaster"},
	[2]string{"Dunlop Fuzzface", "Dunlop Fuzz Face"},
	[2]string{"Xotic RC Booster", "Xotic RC Booster"},
	[2]string{"Keeley Red Dirt", "Keeley Red Dirt"},
	[2]string{"Vemuram Jan Ray", "Vemuram Jan Ray"},
	[2]string{"Nobels ODR-1", "Nobels ODR-1"},
	[2]string{"Mr Black Thunderclaw", "Mr Black Thunderclaw"},
	[2]string{"JHS Bender Fuzz", "JHS Bender Fuzz"},
)

// compressors is the "Compressor" category.
var compressors = items(
	[2]string{"Universal Audio 1176", "Universal Audio 1176"},
	[2]string{"SSL Bus Compressor", "SSL Bus Compressor"},
	[2]string{"VCA Compressor", "VCA compressor"},
	[2]string{"Opto Compressor", "Opto compressor"},
	[2]string{"Diamond Compressor", "Diamond Compressor"},
	[2]string{"BOSS CS-3", "BOSS CS-3 Compression Sustainer"},
)

// equalizers is the "Equalizer" category.
var equalizers = items(
	[2]string{"Parametric-8", "8-band parametric EQ"},
	[2]string{"Parametric-3", "3-band parametric EQ"},
	[2]string{"Graphic-9", "9-band graphic EQ"},
	[2]string{"Low-High Cut", "low/high cut filter"},
)

// delays is the "Delay" category.
var delays = items(
	[2]string{"Digital Delay", "digital delay"},
	[2]string{"Simple Delay", "simple delay"},
	[2]string{"Simple Ping Pong Delay", "ping-pong delay"},
	[2]string{"Tape Delay", "tape delay"},
	[2]string{"Slapback Delay", "slapback (analog) delay"},
	[2]string{"Analog Delay", "analog BBD delay"},
	[2]string{"Dual Delay", "dual delay"},
	[2]string{"Reverse Delay", "reverse delay"},
	[2]string{"Dual Reverse Delay", "dual reverse delay"},
)

// modulation is the "Modulation" category.
var modulation = items(
	[2]string{"Vintage Chorus", "vintage chorus"},
	[2]string{"Dual Chorus", "dual chorus"},
	[2]string{"Chorus Engine", "advanced chorus"},
	[2]string{"BOSS CE2W", "BOSS CE-2W Chorus"},
	[2]string{"BOSS DC-2W", "BOSS DC-2W Dimension Chorus"},
	[2]string{"TC Electronic 2290", "TC Electronic 2290 chorus"},
	[2]string{"TC Electronic The Dreamscape", "TC Electronic Dreamscape"},
	[2]string{"Vibrato", "vibrato"},
	[2]string{"Flangerish", "flanger"},
	[2]string{"Digital Flanger", "digital flanger"},
	[2]string{"Flanger Engine", "advanced flanger"},
	[2]string{"MXR M-117R Flanger", "MXR M-117R Flanger"},
	[2]string{"Phaser", "MXR Phase 90"},
	[2]string{"MXR Phase 95", "MXR Phase 95"},
	[2]string{"NuVibes", "Uni-Vibe"},
	[2]string{"MXR Uni-Vibe", "MXR Uni-Vibe"},
	[2]string{"Tremolo", "tremolo"},
	[2]string{"Rotary", "Leslie rotary speaker"},
)

// reverbs is the "Reverb" category.
var reverbs = items(
	[2]string{"Room", "room reverb"},
	[2]string{"Hall", "hall reverb"},
	[2]string{"Cave", "cave reverb"},
	[2]string{"Plate", "plate reverb"},
	[2]string{"Plate Lush", "lush plate reverb"},
	[2]string{"Plate Tight", "tight plate reverb"},
	[2]string{"Spring", "spring reverb"},
	[2]string{"Modulated", "modulated reverb"},
	[2]string{"Ambience", "ambience reverb"},
	[2]string{"Shimmer", "shimmer reverb"},
	[2]string{"Mind Hall", "mind hall reverb"},
)

// wahs is the "Wah" category.
var wahs = items(
	[2]string{"Dunlop Budda Budwah", "Dunlop Budda Budwah"},
	[2]string{"Bass Wah", "Dunlop Bass Wah"},
	[2]string{"Dunlop Cry Baby GCB-95", "Dunlop Cry Baby GCB-95"},
	[2]string{"Dunlop Cry Baby Clyde McCoy", "Dunlop Cry Baby Clyde McCoy"},
	[2]string{"Morley Bad Horsie", "Morley Bad Horsie"},
)

// pitch is the "Pitch" category.
var pitch = items(
	[2]string{"Pitch Shifter", "pitch shifter"},
	[2]string{"Electro-Harmonix POG", "Electro-Harmonix POG"},
	[2]string{"Digitech Whammy", "Digitech Whammy"},
	[2]string{"Minivoicer", "polyphonic voicer"},
)

// filters is the "Filter" category.
var filters = items(
	[2]string{"Lovetone Meatball", "Lovetone Meatball"},
	[2]string{"Moog Moogerfooger MF-101", "Moog Moogerfooger MF-101"},
	[2]string{"Envelope Filter", "envelope filter"},
)

// gates is the "Utility" noise-gate subset.
var gates = items(
	[2]string{"Adaptive Gate", "adaptive noise gate"},
	[2]string{"Utility Gate", "noise gate"},
	[2]string{"Simple Gate", "simple noise gate"},
)
