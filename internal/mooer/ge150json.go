package mooer

import (
	"encoding/json"
	"fmt"
)

// The classic Mooer GE150 exchanges presets as JSON files with the "GE150
// Preset" schema — the format the GE150 Edit app reads and writes (unlike the
// GE150 Pro Li's binary record layout). The schema was deduced from the app's
// own bundled resources: Empty.mo (a real template), preset.json (the catalog
// with per-knob ranges) and effects.json (the knob-index → name map).
//
// Every knob is a plain integer; most are on the 0-100 scale. The two
// exceptions are the EQ bands (stored as 0..32 with 16 = flat, i.e. -16..+16
// dB shifted) and the delay time / reverb pre-delay, which are milliseconds.
// The template ships a typo — "CEBTER" for the cab CENTER key — which the
// reader accepts alongside the canonical spelling.

// ge150JSONCodec is the classic GE150 .mo layout.
type ge150JSONCodec struct{}

func (ge150JSONCodec) Marshal(p Preset) []byte { return marshalGE150JSON(p) }

func (ge150JSONCodec) Unmarshal(data []byte) (Preset, error) { return unmarshalGE150JSON(data) }

func (ge150JSONCodec) Match(data []byte) bool {
	var doc ge150PresetJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	return doc.FileInfo.Schema == "GE150 Preset"
}

// The JSON effect-module keys, in the order the GE150 Edit app writes them.
const (
	ge150ModFX     = "FX/COMP"
	ge150ModDrive  = "DS/OD"
	ge150ModAmp    = "AMP"
	ge150ModCab    = "CAB"
	ge150ModNS     = "NS GATE"
	ge150ModEQ     = "EQ"
	ge150ModMod    = "MOD"
	ge150ModDelay  = "DELAY"
	ge150ModReverb = "REVERB"
)

type ge150PresetJSON struct {
	Exp          ge150Exp               `json:"Exp"`
	EffectModule map[string]ge150Module `json:"effectModule"`
	FileInfo     ge150FileInfo          `json:"fileInfo"`
}

type ge150Exp struct {
	FunSwitch  int `json:"FUN_SWITCH"`
	ModuleCtrl int `json:"MODULE_CTRL"`
	ParaCtrl   int `json:"PARA_CTRL"`
	VolMax     int `json:"VOL_MAX"`
	VolMin     int `json:"VOL_MIN"`
	VolSwitch  int `json:"VOL_SWITCH"`
}

type ge150Module struct {
	Type   int            `json:"TYPE"`
	Switch int            `json:"SWITCH"`
	Data   map[string]int `json:"Data"`
}

type ge150FileInfo struct {
	App        string `json:"app"`
	AppVersion string `json:"app_version"`
	Device     string `json:"device"`
	DeviceVer  string `json:"device_version"`
	Schema     string `json:"schema"`
}

// onOff encodes a module's enabled state as the JSON switch integer.
func onOff(b bool) int {
	if b {
		return 1
	}
	return 0
}

// marshalGE150JSON renders a preset as a GE150 Edit JSON document.
func marshalGE150JSON(p Preset) []byte {
	doc := ge150PresetJSON{
		Exp: ge150Exp{
			FunSwitch:  0,
			ModuleCtrl: 0,
			ParaCtrl:   1,
			VolMax:     100,
			VolMin:     0,
			VolSwitch:  0,
		},
		EffectModule: map[string]ge150Module{
			ge150ModFX: ge150Module{
				Type: int(p.FX.Type), Switch: onOff(p.FX.Enabled),
				Data: map[string]int{"Q": int(p.FX.Q), "POSITION": int(p.FX.Position), "PEAK": int(p.FX.Peak), "LEVEL": int(p.FX.Level)},
			},
			ge150ModDrive: ge150Module{
				Type: int(p.Drive.Type), Switch: onOff(p.Drive.Enabled),
				Data: map[string]int{"VOLUME": int(p.Drive.Volume), "TONE": int(p.Drive.Tone), "GAIN": int(p.Drive.Gain)},
			},
			ge150ModAmp: ge150Module{
				Type: int(p.Amp.Type), Switch: onOff(p.Amp.Enabled),
				Data: map[string]int{"GAIN": int(p.Amp.Gain), "BASS": int(p.Amp.Bass), "MID": int(p.Amp.Mid), "TREBLE": int(p.Amp.Treble), "PRES": int(p.Amp.Presence), "MST": int(p.Amp.Master)},
			},
			ge150ModCab: ge150Module{
				Type: int(p.Cab.Type), Switch: onOff(p.Cab.Enabled),
				Data: map[string]int{"TUBE": int(p.Cab.Tube), "MIC": int(p.Cab.Mic), "CENTER": int(p.Cab.Center), "DISTANCE": int(p.Cab.Distance)},
			},
			ge150ModNS: ge150Module{
				Type: int(p.NoiseGate.Type), Switch: onOff(p.NoiseGate.Enabled),
				Data: map[string]int{"THRES": int(p.NoiseGate.Threshold)},
			},
			ge150ModEQ: marshalGE150EQ(p),
			ge150ModMod: ge150Module{
				Type: int(p.Mod.Type), Switch: onOff(p.Mod.Enabled),
				Data: map[string]int{"RATE": int(p.Mod.Rate), "LEVEL": int(p.Mod.Level), "DEPTH": int(p.Mod.Depth)},
			},
			ge150ModDelay: ge150Module{
				Type: int(p.Delay.Type), Switch: onOff(p.Delay.Enabled),
				Data: map[string]int{"TIME": int(p.Delay.TimeMS), "FEEDBACK": int(p.Delay.Feedback), "LEVEL": int(p.Delay.Level), "SUB-D": int(p.Delay.Subdivision)},
			},
			ge150ModReverb: ge150Module{
				Type: int(p.Reverb.Type), Switch: onOff(p.Reverb.Enabled),
				Data: map[string]int{"PRE DELAY": int(p.Reverb.PreDelay), "LEVEL": int(p.Reverb.Level), "DECAY": int(p.Reverb.Decay), "TONE": int(p.Reverb.Tone)},
			},
		},
		FileInfo: ge150FileInfo{
			App:        "GE150 Edit",
			AppVersion: "V1.1.0",
			Device:     "MOOER GE150",
			DeviceVer:  "V1.1.0",
			Schema:     "GE150 Preset",
		},
	}
	b, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		// Marshalling a fixed set of JSON-safe types cannot fail.
		panic(err)
	}
	return b
}

// marshalGE150EQ writes the EQ module: five bands on the 0..32 scale (16 = flat).
func marshalGE150EQ(p Preset) ge150Module {
	bands := map[string]int{
		"100Hz":  ge150EQBand(p.EQ.Bands[0]),
		"250Hz":  ge150EQBand(p.EQ.Bands[1]),
		"630Hz":  ge150EQBand(p.EQ.Bands[2]),
		"1.6KHz": ge150EQBand(p.EQ.Bands[3]),
		"4KHz":   ge150EQBand(p.EQ.Bands[4]),
	}
	return ge150Module{Type: int(p.EQ.Type), Switch: onOff(p.EQ.Enabled), Data: bands}
}

// ge150EQBand maps an internal 0-100 band (50 = flat) to the GE150's stored
// 0..32 scale (16 = flat).
func ge150EQBand(v uint8) int {
	return int(uint16(v)*32+50) / 100 // #nosec G115 -- 0..100 x 32 / 100 fits in an int
}

// ge150EQBandInverse maps a stored 0..32 band back to the internal 0-100 scale.
func ge150EQBandInverse(v int) uint8 {
	if v < 0 {
		v = 0
	}
	if v > 32 {
		v = 32
	}
	return uint8((v*100 + 16) / 32) // #nosec G115 -- 0..32 x 100 / 32 fits in a byte
}

// unmarshalGE150JSON parses a GE150 Edit JSON document into a Preset.
func unmarshalGE150JSON(data []byte) (Preset, error) {
	var doc ge150PresetJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return Preset{}, fmt.Errorf("parse GE150 preset: %w", err)
	}
	if doc.FileInfo.Schema != "GE150 Preset" {
		return Preset{}, fmt.Errorf("not a GE150 preset: schema %q", doc.FileInfo.Schema)
	}

	var p Preset
	m := doc.EffectModule
	for _, mod := range ge150ModuleReaders(&p) {
		if v, ok := m[mod.key]; ok {
			mod.apply(v)
		}
	}
	return p, nil
}

// ge150ModuleReader ties one JSON module key to the closure that folds its
// record into the Preset, so the unmarshal loop stays flat.
type ge150ModuleReader struct {
	key   string
	apply func(ge150Module)
}

// ge150Type narrows a JSON TYPE integer to the effect_type index; the device
// numbers its models well below 256.
func ge150Type(v int) uint8 {
	return uint8(v) // #nosec G115 -- model indices are < 256
}

// ge150Time narrows a JSON TIME integer to the delay time in milliseconds.
func ge150Time(v int) uint16 {
	return uint16(v) // #nosec G115 -- the delay time is 0..2000 ms
}

func ge150ModuleReaders(p *Preset) []ge150ModuleReader {
	return []ge150ModuleReader{
		{ge150ModFX, func(v ge150Module) {
			p.FX = FX{Enabled: v.Switch != 0, Type: ge150Type(v.Type), Q: dataU8(v.Data, "Q"), Position: dataU8(v.Data, "POSITION"), Peak: dataU8(v.Data, "PEAK"), Level: dataU8(v.Data, "LEVEL")}
		}},
		{ge150ModDrive, func(v ge150Module) {
			p.Drive = Drive{Enabled: v.Switch != 0, Type: ge150Type(v.Type), Volume: dataU8(v.Data, "VOLUME"), Tone: dataU8(v.Data, "TONE"), Gain: dataU8(v.Data, "GAIN")}
		}},
		{ge150ModAmp, func(v ge150Module) {
			p.Amp = Amp{Enabled: v.Switch != 0, Type: ge150Type(v.Type), Gain: dataU8(v.Data, "GAIN"), Bass: dataU8(v.Data, "BASS"), Mid: dataU8(v.Data, "MID"), Treble: dataU8(v.Data, "TREBLE"), Presence: dataU8(v.Data, "PRES"), Master: dataU8(v.Data, "MST")}
		}},
		{ge150ModCab, func(v ge150Module) {
			p.Cab = Cab{Enabled: v.Switch != 0, Type: ge150Type(v.Type), Tube: dataU8(v.Data, "TUBE"), Mic: dataU8(v.Data, "MIC"), Center: cabCenter(v.Data), Distance: dataU8(v.Data, "DISTANCE")}
		}},
		{ge150ModNS, func(v ge150Module) {
			p.NoiseGate = NoiseGate{Enabled: v.Switch != 0, Type: ge150Type(v.Type), Threshold: dataU8(v.Data, "THRES")}
		}},
		{ge150ModEQ, func(v ge150Module) {
			p.EQ.Enabled = v.Switch != 0
			p.EQ.Type = ge150Type(v.Type)
			p.EQ.Bands = [6]uint8{
				ge150EQBandInverse(dataInt(v.Data, "100Hz")),
				ge150EQBandInverse(dataInt(v.Data, "250Hz")),
				ge150EQBandInverse(dataInt(v.Data, "630Hz")),
				ge150EQBandInverse(dataInt(v.Data, "1.6KHz")),
				ge150EQBandInverse(dataInt(v.Data, "4KHz")),
			}
		}},
		{ge150ModMod, func(v ge150Module) {
			p.Mod = Mod{Enabled: v.Switch != 0, Type: ge150Type(v.Type), Rate: dataU8(v.Data, "RATE"), Level: dataU8(v.Data, "LEVEL"), Depth: dataU8(v.Data, "DEPTH"), Param4: noon, Param5: noon}
		}},
		{ge150ModDelay, func(v ge150Module) {
			p.Delay = Delay{Enabled: v.Switch != 0, Type: ge150Type(v.Type), TimeMS: ge150Time(dataInt(v.Data, "TIME")), Feedback: dataU8(v.Data, "FEEDBACK"), Level: dataU8(v.Data, "LEVEL"), Subdivision: dataU8(v.Data, "SUB-D"), Param5: noon, Param6: noon}
		}},
		{ge150ModReverb, func(v ge150Module) {
			p.Reverb = Reverb{Enabled: v.Switch != 0, Type: ge150Type(v.Type), PreDelay: dataU8(v.Data, "PRE DELAY"), Level: dataU8(v.Data, "LEVEL"), Decay: dataU8(v.Data, "DECAY"), Tone: dataU8(v.Data, "TONE")}
		}},
	}
}

// dataU8 reads a 0-100 knob value by name, defaulting to zero when absent.
func dataU8(data map[string]int, key string) uint8 {
	return uint8(data[key]) // #nosec G115 -- knob values are 0..100
}

// dataInt reads a knob value by name, defaulting to zero when absent.
func dataInt(data map[string]int, key string) int {
	return data[key]
}

// cabCenter reads the cab CENTER knob, tolerating the editor's "CEBTER" typo.
func cabCenter(data map[string]int) uint8 {
	if v, ok := data["CENTER"]; ok {
		return uint8(v) // #nosec G115 -- knob values are 0..100
	}
	return uint8(data["CEBTER"]) // #nosec G115 -- knob values are 0..100
}
