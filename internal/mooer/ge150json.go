package mooer

import (
	"encoding/json"
	"fmt"
)

// The classic Mooer GE150 exchanges presets as JSON files with the "GE150
// Preset" schema — the format the GE150 Edit app reads and writes (unlike the
// GE150 Pro Li's binary record layout). The schema was deduced from the app's
// own bundled resources: ge150-empty.mo (a real template) and preset.json (the
// catalog with per-effect knob lists and ranges).
//
// Every knob is a plain integer; most are on the 0-100 scale. The exceptions
// are the EQ bands (stored as 0..32 with 16 = flat, i.e. -16..+16 dB shifted)
// and the time knobs (delay time, reverb pre-delay), which are milliseconds.
// Each effect type has its own knob list (ge150knobs.go), so a MOD flanger
// writes MIX/FEEDBACK while a MOD phaser writes LEVEL/DEPTH — the shared
// Preset fields are mapped positionally onto the device's real knob names.
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

// ge150ModuleKeys lists the JSON module keys in the GE150 Edit app's order.
var ge150ModuleKeys = [...]string{
	ge150ModFX, ge150ModDrive, ge150ModAmp, ge150ModCab, ge150ModNS,
	ge150ModEQ, ge150ModMod, ge150ModDelay, ge150ModReverb,
}

// ge150CustomEQFreq is the centre frequency, in Hz, written for the custom
// parametric EQ's three bands. The GE150 stores a gain and a centre frequency
// per band, but the shared Preset model only carries gains, so the frequency
// is fixed at the geometric centre of the device's 60 Hz–18 kHz range. The
// gain round-trips; the frequency is a documented gap.
const ge150CustomEQFreq = 1000

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

// ge150KnobList returns the knob list for a module's effect type. A module
// whose list is shared by every type (DS/OD, AMP, CAB, REVERB) returns it for
// any type; a per-type module returns nil when the type is out of range.
func ge150KnobList(module string, typ uint8) []ge150Knob {
	lists := ge150Knobs[module]
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		return lists[0]
	}
	if int(typ) >= len(lists) {
		return nil
	}
	return lists[int(typ)]
}

// ge150Encode maps an internal field value onto a knob's stored range.
func ge150Encode(k ge150Knob, field int) int {
	switch k.Kind {
	case ge150KnobSel:
		return field
	case ge150KnobMs:
		return clampInt(field, k.Min, k.Max)
	default:
		if k.Max == k.Min {
			return field
		}
		return k.Min + (field*(k.Max-k.Min)+50)/100
	}
}

// ge150Decode maps a stored value back to the internal field value; the caller
// narrows the result to the field's own width.
func ge150Decode(k ge150Knob, v int) int {
	switch k.Kind {
	case ge150KnobSel, ge150KnobMs:
		return v
	default:
		if k.Max == k.Min {
			return v
		}
		return ((v - k.Min) * 100) / (k.Max - k.Min)
	}
}

// ge150Value reads a knob's stored value, tolerating the editor's "CEBTER"
// typo for the cab CENTER key.
func ge150Value(data map[string]int, k ge150Knob) int {
	if v, ok := data[k.Name]; ok {
		return v
	}
	if k.Name == "CENTER" {
		return data["CEBTER"]
	}
	return 0
}

// clampInt bounds v to [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// u8clamp narrows a decoded value to a 0..255 field, the shared Preset model's
// per-knob width.
func u8clamp(v int) uint8 {
	return uint8(clampInt(v, 0, 255)) // #nosec G115 -- bounded above
}

// ge150Type narrows a JSON TYPE integer to the effect_type index; the device
// numbers its models well below 256.
func ge150Type(v int) uint8 {
	return uint8(v) // #nosec G115 -- model indices are < 256
}

// ge150Time narrows a JSON TIME integer to the delay time in milliseconds.
func ge150Time(v int) uint16 {
	return uint16(clampInt(v, 0, 65535)) // #nosec G115 -- bounded above
}

// ge150Fields returns a module's knob values in the order of its knob list.
// The noise gate is semantic rather than positional: NOISE KILLER and INTEL
// REDUCER expose only one knob, so Attack/Release are unused there.
func ge150Fields(p Preset, module string) []int {
	switch module {
	case ge150ModFX:
		return []int{int(p.FX.Q), int(p.FX.Position), int(p.FX.Peak), int(p.FX.Level)}
	case ge150ModDrive:
		return []int{int(p.Drive.Volume), int(p.Drive.Tone), int(p.Drive.Gain)}
	case ge150ModAmp:
		return []int{int(p.Amp.Gain), int(p.Amp.Bass), int(p.Amp.Mid), int(p.Amp.Treble), int(p.Amp.Presence), int(p.Amp.Master)}
	case ge150ModCab:
		return []int{int(p.Cab.Tube), int(p.Cab.Mic), int(p.Cab.Center), int(p.Cab.Distance)}
	case ge150ModNS:
		return ge150NSFields(p.NoiseGate)
	case ge150ModMod:
		return []int{int(p.Mod.Rate), int(p.Mod.Level), int(p.Mod.Depth), int(p.Mod.Param4), int(p.Mod.Param5)}
	case ge150ModDelay:
		return []int{int(p.Delay.Level), int(p.Delay.Feedback), int(p.Delay.TimeMS), int(p.Delay.Subdivision), int(p.Delay.Param5), int(p.Delay.Param6)}
	case ge150ModReverb:
		return []int{int(p.Reverb.PreDelay), int(p.Reverb.Level), int(p.Reverb.Decay), int(p.Reverb.Tone)}
	}
	return nil
}

// ge150NSFields returns the noise gate's knob values: NOISE KILLER and INTEL
// REDUCER expose only the threshold, NOISE GATE exposes attack, release and
// threshold.
func ge150NSFields(n NoiseGate) []int {
	if n.Type <= 1 {
		return []int{int(n.Threshold)}
	}
	return []int{int(n.Attack), int(n.Release), int(n.Threshold)}
}

// ge150SetFields folds a module's decoded knob values back into the preset.
// Fields the module's type does not expose fall back to noon.
func ge150SetFields(p *Preset, module string, v ge150Module, values []int) {
	enabled := v.Switch != 0
	typ := ge150Type(v.Type)
	switch module {
	case ge150ModFX:
		p.FX = FX{Enabled: enabled, Type: typ, Q: ge150U8At(values, 0), Position: ge150U8At(values, 1), Peak: ge150U8At(values, 2), Level: ge150U8At(values, 3)}
	case ge150ModDrive:
		p.Drive = Drive{Enabled: enabled, Type: typ, Volume: ge150U8At(values, 0), Tone: ge150U8At(values, 1), Gain: ge150U8At(values, 2)}
	case ge150ModAmp:
		p.Amp = Amp{Enabled: enabled, Type: typ, Gain: ge150U8At(values, 0), Bass: ge150U8At(values, 1), Mid: ge150U8At(values, 2), Treble: ge150U8At(values, 3), Presence: ge150U8At(values, 4), Master: ge150U8At(values, 5)}
	case ge150ModCab:
		p.Cab = Cab{Enabled: enabled, Type: typ, Tube: ge150U8At(values, 0), Mic: ge150U8At(values, 1), Center: ge150U8At(values, 2), Distance: ge150U8At(values, 3)}
	case ge150ModNS:
		p.NoiseGate = ge150SetNS(enabled, typ, values)
	case ge150ModMod:
		p.Mod = Mod{Enabled: enabled, Type: typ, Rate: ge150U8At(values, 0), Level: ge150U8At(values, 1), Depth: ge150U8At(values, 2), Param4: ge150U8At(values, 3), Param5: ge150U8At(values, 4)}
	case ge150ModDelay:
		p.Delay = Delay{
			Enabled: enabled, Type: typ, Level: ge150U8At(values, 0), Feedback: ge150U8At(values, 1),
			TimeMS: ge150Time(intValue(values, 2)), Subdivision: ge150U8At(values, 3), Param5: ge150U8At(values, 4), Param6: ge150U8At(values, 5),
		}
	case ge150ModReverb:
		p.Reverb = Reverb{Enabled: enabled, Type: typ, PreDelay: ge150U8At(values, 0), Level: ge150U8At(values, 1), Decay: ge150U8At(values, 2), Tone: ge150U8At(values, 3)}
	}
}

// ge150SetNS folds the noise gate's decoded knobs back: the one-knob gates
// keep Attack/Release at noon.
func ge150SetNS(enabled bool, typ uint8, values []int) NoiseGate {
	if typ <= 1 {
		return NoiseGate{Enabled: enabled, Type: typ, Attack: noon, Release: noon, Threshold: ge150U8At(values, 0)}
	}
	return NoiseGate{Enabled: enabled, Type: typ, Attack: ge150U8At(values, 0), Release: ge150U8At(values, 1), Threshold: ge150U8At(values, 2)}
}

// ge150U8At returns the i-th decoded value as a 0..255 field, noon when the
// list is shorter.
func ge150U8At(values []int, i int) uint8 {
	if i >= len(values) {
		return noon
	}
	return u8clamp(values[i])
}

// intValue returns values[i], or zero when i is out of range.
func intValue(values []int, i int) int {
	if i >= len(values) {
		return 0
	}
	return values[i]
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
			ge150ModFX:     marshalGE150Knobs(ge150ModFX, p.FX.Type, p.FX.Enabled, ge150Fields(p, ge150ModFX)),
			ge150ModDrive:  marshalGE150Knobs(ge150ModDrive, p.Drive.Type, p.Drive.Enabled, ge150Fields(p, ge150ModDrive)),
			ge150ModAmp:    marshalGE150Knobs(ge150ModAmp, p.Amp.Type, p.Amp.Enabled, ge150Fields(p, ge150ModAmp)),
			ge150ModCab:    marshalGE150Knobs(ge150ModCab, p.Cab.Type, p.Cab.Enabled, ge150Fields(p, ge150ModCab)),
			ge150ModNS:     marshalGE150Knobs(ge150ModNS, p.NoiseGate.Type, p.NoiseGate.Enabled, ge150Fields(p, ge150ModNS)),
			ge150ModEQ:     marshalGE150EQ(p),
			ge150ModMod:    marshalGE150Knobs(ge150ModMod, p.Mod.Type, p.Mod.Enabled, ge150Fields(p, ge150ModMod)),
			ge150ModDelay:  marshalGE150Knobs(ge150ModDelay, p.Delay.Type, p.Delay.Enabled, ge150Fields(p, ge150ModDelay)),
			ge150ModReverb: marshalGE150Knobs(ge150ModReverb, p.Reverb.Type, p.Reverb.Enabled, ge150Fields(p, ge150ModReverb)),
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

// marshalGE150Knobs writes a module whose knobs come from the knob table, in
// the device's own key order.
func marshalGE150Knobs(module string, typ uint8, enabled bool, fields []int) ge150Module {
	knobs := ge150KnobList(module, typ)
	data := make(map[string]int, len(knobs))
	for i, k := range knobs {
		if i >= len(fields) {
			break
		}
		data[k.Name] = ge150Encode(k, fields[i])
	}
	return ge150Module{Type: int(typ), Switch: onOff(enabled), Data: data}
}

// marshalGE150EQ writes the EQ module. The graphic EQ types (0-2) use the knob
// table; the custom parametric EQ (type 3) maps our first three bands onto its
// three gains, with the centre frequencies fixed at ge150CustomEQFreq.
func marshalGE150EQ(p Preset) ge150Module {
	if p.EQ.Type == 3 {
		return ge150Module{
			Type: 3, Switch: onOff(p.EQ.Enabled),
			Data: map[string]int{
				"GAIN1": ge150EQBand(p.EQ.Bands[0]),
				"FREQ1": ge150CustomEQFreq,
				"GAIN2": ge150EQBand(p.EQ.Bands[1]),
				"FREQ2": ge150CustomEQFreq,
				"GAIN3": ge150EQBand(p.EQ.Bands[2]),
				"FREQ3": ge150CustomEQFreq,
			},
		}
	}
	fields := []int{
		int(p.EQ.Bands[0]), int(p.EQ.Bands[1]), int(p.EQ.Bands[2]),
		int(p.EQ.Bands[3]), int(p.EQ.Bands[4]), int(p.EQ.Bands[5]),
	}
	return marshalGE150Knobs(ge150ModEQ, p.EQ.Type, p.EQ.Enabled, fields)
}

// ge150EQBand maps an internal 0-100 band (50 = flat) to the GE150's stored
// 0..32 scale (16 = flat).
func ge150EQBand(v uint8) int {
	return int(uint16(v)*32+50) / 100 // #nosec G115 -- 0..100 x 32 / 100 fits in an int
}

// ge150EQBandInverse maps a stored 0..32 band back to the internal 0-100 scale.
func ge150EQBandInverse(v int) uint8 {
	return u8clamp((v*100 + 16) / 32)
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
	for _, key := range ge150ModuleKeys {
		if v, ok := doc.EffectModule[key]; ok {
			readGE150Module(&p, key, v)
		}
	}
	return p, nil
}

// readGE150Module decodes one module's knobs through the knob table and folds
// them into the preset. The EQ module is special: its custom parametric type
// stores gain/frequency pairs our model only partly represents.
func readGE150Module(p *Preset, module string, v ge150Module) {
	if module == ge150ModEQ {
		readGE150EQ(p, v)
		return
	}
	knobs := ge150KnobList(module, ge150Type(v.Type))
	values := make([]int, len(knobs))
	for i, k := range knobs {
		values[i] = ge150Decode(k, ge150Value(v.Data, k))
	}
	ge150SetFields(p, module, v, values)
}

// readGE150EQ folds the EQ module into the preset.
func readGE150EQ(p *Preset, v ge150Module) {
	p.EQ.Enabled = v.Switch != 0
	p.EQ.Type = ge150Type(v.Type)
	if v.Type == 3 {
		// CUSTOM EQ: three parametric bands; only the gains map onto our
		// model, the centre frequencies are dropped.
		p.EQ.Bands = [6]uint8{
			ge150EQBandInverse(v.Data["GAIN1"]),
			ge150EQBandInverse(v.Data["GAIN2"]),
			ge150EQBandInverse(v.Data["GAIN3"]),
			noon, noon, noon,
		}
		return
	}
	knobs := ge150KnobList(ge150ModEQ, p.EQ.Type)
	for i := range p.EQ.Bands {
		if i < len(knobs) {
			p.EQ.Bands[i] = u8clamp(ge150Decode(knobs[i], ge150Value(v.Data, knobs[i])))
		} else {
			p.EQ.Bands[i] = noon
		}
	}
}
