package thr

import (
	"encoding/json"
	"fmt"
	"math"
)

// The Yamaha THR-II stores presets as JSON files with the .thrl6p extension
// (schema "L6Preset", version 5). The format was reverse-engineered by the
// community (f3sty/Yamaha_THRII_presets) and documented by
// mtremoulet/GuitarSkills. The .thrl6p is read and written by the THR Remote
// app; the `device` id must match the connected amp, otherwise THR Remote
// imports the FX but silently drops the amp and cabinet.
//
// Every control knob is a normalized 0..1 float; a 0-100 UI value maps to
// value/100. The noise-gate threshold is the one exception: the app stores it
// in decibels, dB = (ui - 100) * 0.96. The guitar/audio levels are device
// state, not preset data, so they are not part of the file.
const (
	thrl6pSchema    = "L6Preset"
	thrl6pVersion   = 5
	thrl6pDevice    = 2359296
	thrl6pDeviceVer = 22020194
	thrl6pCabBypass = 16
	thrl6pTempo     = 110 // the default tempo every real export carries
	thrl6pGateUnset = -50.0
)

// The app must resolve every effect group to a known model, even when the
// effect is off, so an unselected effect writes the app's own default asset
// with @enabled false (the empty @asset string we used to write is rejected).
const (
	thrl6pModDefault    = "StereoSquareChorus"
	thrl6pEchoDefault   = "TapeEcho"
	thrl6pReverbDefault = "StandardSpring"
	thrl6pAmpDefault    = "THR10C_Deluxe"
)

// thrl6pDoc is a .thrl6p file.
type thrl6pDoc struct {
	Schema  string     `json:"schema"`
	Version int        `json:"version"`
	Data    thrl6pData `json:"data"`
	Meta    thrl6pMeta `json:"meta"`
}

type thrl6pMeta struct {
	Original int `json:"original"`
	Pbn      int `json:"pbn"`
	Premium  int `json:"premium"`
}

type thrl6pData struct {
	Meta      thrl6pName `json:"meta"`
	Device    int        `json:"device"`
	DeviceVer int        `json:"device_version"`
	Tone      thrl6pTone `json:"tone"`
}

type thrl6pName struct {
	Name string `json:"name"`
	Tnid int    `json:"tnid"`
}

type thrl6pTone struct {
	Amp        thrl6pAmp        `json:"THRGroupAmp"`
	Cab        thrl6pCab        `json:"THRGroupCab"`
	Compressor thrl6pCompressor `json:"THRGroupFX1Compressor"`
	Mod        thrl6pMod        `json:"THRGroupFX2Effect"`
	Echo       thrl6pEcho       `json:"THRGroupFX3EffectEcho"`
	Reverb     thrl6pReverb     `json:"THRGroupFX4EffectReverb"`
	Gate       thrl6pGate       `json:"THRGroupGate"`
	Global     thrl6pGlobal     `json:"global"`
}

type thrl6pAmp struct {
	Asset  string  `json:"@asset"`
	Drive  float64 `json:"Drive"`
	Bass   float64 `json:"Bass"`
	Mid    float64 `json:"Mid"`
	Treble float64 `json:"Treble"`
	Master float64 `json:"Master"`
}

type thrl6pCab struct {
	Asset      string `json:"@asset"`
	SpkSimType int    `json:"SpkSimType"`
}

type thrl6pCompressor struct {
	Asset   string  `json:"@asset"`
	Enabled bool    `json:"@enabled"`
	Sustain float64 `json:"Sustain"`
	Level   float64 `json:"Level"`
}

type thrl6pMod struct {
	Asset    string  `json:"@asset"`
	Enabled  bool    `json:"@enabled"`
	WetDry   float64 `json:"@wetDry"`
	Depth    float64 `json:"Depth"`
	Feedback float64 `json:"Feedback"`
	Freq     float64 `json:"Freq"`
	Pre      float64 `json:"Pre"`
	Speed    float64 `json:"Speed"`
}

type thrl6pEcho struct {
	Asset    string  `json:"@asset"`
	Enabled  bool    `json:"@enabled"`
	WetDry   float64 `json:"@wetDry"`
	Time     float64 `json:"Time"`
	Bass     float64 `json:"Bass"`
	Treble   float64 `json:"Treble"`
	Feedback float64 `json:"Feedback"`
}

type thrl6pReverb struct {
	Asset    string  `json:"@asset"`
	Enabled  bool    `json:"@enabled"`
	WetDry   float64 `json:"@wetDry"`
	Decay    float64 `json:"Decay"`
	PreDelay float64 `json:"PreDelay"`
	// Time is the spring reverb's length knob; the hall/plate/room models carry
	// Decay/PreDelay instead.
	Time float64 `json:"Time"`
	Tone float64 `json:"Tone"`
}

type thrl6pGate struct {
	Asset   string  `json:"@asset"`
	Enabled bool    `json:"@enabled"`
	Thresh  float64 `json:"Thresh"`
	Decay   float64 `json:"Decay"`
}

type thrl6pGlobal struct {
	Tempo int `json:"THRPresetParamTempo"`
}

// ampAsset maps a THR-II amp cell name (the amp-selector position, e.g.
// "CLEAN CLASSIC") to the engine @asset key the app stores. The mapping is
// taken from the app's own amps.models resource (bundled with THR Remote), so
// it is authoritative; the community sheet swaps a few "Classic"/"Modern"
// labels and is not trusted here.
var ampAsset = map[string]string{
	"CLEAN CLASSIC":     "THR10C_Deluxe",
	"CLEAN BOUTIQUE":    "THR10C_BJunior2",
	"CLEAN MODERN":      "THR30_Carmen",
	"CRUNCH CLASSIC":    "THR10C_DC30",
	"CRUNCH BOUTIQUE":   "THR30_SR101",
	"CRUNCH MODERN":     "THR10C_Mini",
	"LEAD CLASSIC":      "THR10_Lead",
	"LEAD BOUTIQUE":     "THR30_Blondie",
	"LEAD MODERN":       "THR10X_Brown1",
	"HI GAIN CLASSIC":   "THR10_Modern",
	"HI GAIN BOUTIQUE":  "THR30_FLead",
	"HI GAIN MODERN":    "THR10X_Brown2",
	"SPECIAL CLASSIC":   "THR10_Brit",
	"SPECIAL BOUTIQUE":  "THR10X_South",
	"SPECIAL MODERN":    "THR30_Stealth",
	"BASS CLASSIC":      "THR10_Bass_Eden_Marcus",
	"BASS BOUTIQUE":     "THR10_Bass_Mesa",
	"BASS MODERN":       "THR30_JKBass2",
	"ACOUSTIC CLASSIC":  "THR10_Aco_Condenser1",
	"ACOUSTIC BOUTIQUE": "THR10_Aco_Tube1",
	"ACOUSTIC MODERN":   "THR10_Aco_Dynamic1",
	"FLAT CLASSIC":      "THR10_Flat",
	"FLAT BOUTIQUE":     "THR10_Flat_B",
	"FLAT MODERN":       "THR10_Flat_V",
}

// cabSpkSimType maps a THR-II cabinet name to the SpkSimType index the app
// stores (0..15; 16 is bypass).
var cabSpkSimType = map[string]int{
	"British 4x12":       0,
	"American 4x12":      1,
	"Brown 4x12":         2,
	"Vintage 4x12":       3,
	"Fuel 4x12":          4,
	"Juicy 4x12":         5,
	"Mods 4x12":          6,
	"American 2x12":      7,
	"British 2x12":       8,
	"British Blues 2x12": 9,
	"Boutique 2x12":      10,
	"Yamaha 2x12":        11,
	"California 1x12":    12,
	"American 1x12":      13,
	"American 4x10":      14,
	"Boutique 1x12":      15,
}

// modAsset maps a modulation name to its engine @asset key.
var modAsset = map[string]string{
	"CHORUS":  "StereoSquareChorus",
	"FLANGER": "L6Flanger",
	"PHASER":  "Phaser",
	"TREMOLO": "BiasTremolo",
}

// echoAsset maps an echo name to its engine @asset key.
var echoAsset = map[string]string{
	"Tape":          "TapeEcho",
	"Digital Delay": "L6DigitalDelay",
}

// reverbAsset maps a reverb name to its engine @asset key.
var reverbAsset = map[string]string{
	"Hall":   "ReallyLargeHall",
	"Plate":  "LargePlate1",
	"Room":   "SmallRoom1",
	"Spring": "StandardSpring",
}

// scaleKnob maps a 0-100 UI value to the 0..1 float the app stores. An unset
// knob (negative) becomes noon, matching the generator's "disabled settings at
// 50%".
func scaleKnob(ui int) float64 {
	if ui < 0 {
		return 0.5
	}
	return float64(ui) / 100
}

// unscaleKnob maps a 0..1 float back to a 0-100 UI value.
func unscaleKnob(f float64) int {
	return int(math.Round(f * 100))
}

// msToUnit maps a millisecond value to the 0..1 float the app stores for the
// time-based knobs (echo time and the reverb/mod pre-delays; 1.0 = 1 second).
func msToUnit(ms int) float64 {
	if ms < 0 {
		return 0.5
	}
	return float64(ms) / 1000
}

// unitToMs maps a stored 0..1 time value back to milliseconds.
func unitToMs(f float64) int {
	return int(math.Round(f * 1000))
}

// gateThresh maps a 0-100 UI gate threshold to the decibel value the app
// stores: dB = (ui - 100) * 0.96, kept as a float.
func gateThresh(ui int) float64 {
	if ui < 0 {
		return thrl6pGateUnset
	}
	return float64(ui-100) * 0.96
}

// ungateThresh maps a stored decibel threshold back to the 0-100 UI scale,
// flooring so a truncated export (e.g. -32) reads back as the same knob.
func ungateThresh(dB float64) int {
	ui := int(math.Floor(100 + dB/0.96))
	if ui < 0 {
		return 0
	}
	if ui > 100 {
		return 100
	}
	return ui
}

// MarshalThrl6p renders a resolved Spec as a .thrl6p JSON document.
func MarshalThrl6p(s Spec) []byte {
	doc := thrl6pDoc{
		Schema:  thrl6pSchema,
		Version: thrl6pVersion,
		Data: thrl6pData{
			Meta:      thrl6pName{Name: s.Name},
			Device:    thrl6pDevice,
			DeviceVer: thrl6pDeviceVer,
			Tone:      marshalThrl6pTone(s),
		},
		Meta: thrl6pMeta{},
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		// Marshalling a fixed set of JSON-safe types cannot fail.
		panic(err)
	}
	return b
}

func marshalThrl6pTone(s Spec) thrl6pTone {
	ampKey := ampAsset[s.Amp]
	if ampKey == "" {
		ampKey = thrl6pAmpDefault
	}
	modKey := modAsset[s.Mod]
	if modKey == "" {
		modKey = thrl6pModDefault
	}
	echoKey := echoAsset[s.Echo]
	if echoKey == "" {
		echoKey = thrl6pEchoDefault
	}
	reverbKey := reverbAsset[s.Reverb]
	if reverbKey == "" {
		reverbKey = thrl6pReverbDefault
	}

	tone := thrl6pTone{
		Amp: thrl6pAmp{
			Asset:  ampKey,
			Drive:  scaleKnob(s.AmpParams.Gain),
			Bass:   scaleKnob(s.AmpParams.Bass),
			Mid:    scaleKnob(s.AmpParams.Mid),
			Treble: scaleKnob(s.AmpParams.Treble),
			Master: scaleKnob(s.AmpParams.Master),
		},
		Cab: thrl6pCab{
			Asset:      "speakerSimulator",
			SpkSimType: thrl6pCabBypass,
		},
		Compressor: thrl6pCompressor{
			Asset:   "RedComp",
			Enabled: s.Compressor,
			Sustain: scaleKnob(s.CompParams.Sustain),
			Level:   scaleKnob(s.CompParams.Level),
		},
		Mod: thrl6pMod{
			Asset:    modKey,
			Enabled:  s.Mod != "",
			WetDry:   scaleKnob(s.ModParams.Mix),
			Depth:    scaleKnob(s.ModParams.Depth),
			Feedback: scaleKnob(s.ModParams.Feedback),
			Freq:     scaleKnob(s.ModParams.Speed),
			Pre:      scaleKnob(s.ModParams.PreDelay),
			Speed:    scaleKnob(s.ModParams.Speed),
		},
		Echo: thrl6pEcho{
			Asset:    echoKey,
			Enabled:  s.Echo != "",
			WetDry:   scaleKnob(s.EchoParams.Mix),
			Time:     msToUnit(s.EchoParams.Time),
			Bass:     scaleKnob(s.EchoParams.Bass),
			Treble:   scaleKnob(s.EchoParams.Treble),
			Feedback: scaleKnob(s.EchoParams.Feedback),
		},
		Reverb: thrl6pReverb{
			Asset:   reverbKey,
			Enabled: s.Reverb != "",
			WetDry:  scaleKnob(s.ReverbParams.Mix),
			Tone:    scaleKnob(s.ReverbParams.Tone),
		},
		Gate: thrl6pGate{
			Asset:   "noiseGate",
			Enabled: s.NoiseGate,
			Thresh:  gateThresh(s.GateParams.Threshold),
			Decay:   scaleKnob(s.GateParams.Decay),
		},
		Global: thrl6pGlobal{Tempo: thrl6pTempo},
	}
	// The spring reverb carries Time/Tone; the hall, plate and room models carry
	// Decay/PreDelay/Tone.
	if reverbKey == "StandardSpring" {
		tone.Reverb.Time = scaleKnob(s.ReverbParams.Decay)
	} else {
		tone.Reverb.Decay = scaleKnob(s.ReverbParams.Decay)
		tone.Reverb.PreDelay = msToUnit(s.ReverbParams.PreDelay)
	}
	if id, ok := cabSpkSimType[s.Cab]; ok {
		tone.Cab.SpkSimType = id
	}
	return tone
}

// UnmarshalThrl6p parses a .thrl6p document into a Spec. The Spec's amp,
// cabinet and effect selections are resolved to the on-device names; the
// guitar/audio levels stay Unset (they are not part of the preset file).
func UnmarshalThrl6p(data []byte) (Spec, error) {
	var doc thrl6pDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return Spec{}, fmt.Errorf("parse .thrl6p: %w", err)
	}
	if doc.Schema != thrl6pSchema {
		return Spec{}, fmt.Errorf("not a THR preset: schema %q", doc.Schema)
	}

	s := NewSpec()
	s.Name = doc.Data.Meta.Name
	t := doc.Data.Tone

	s.Amp = ampCellByAsset(t.Amp.Asset)
	s.AmpParams = AmpParams{
		Gain:   unscaleKnob(t.Amp.Drive),
		Master: unscaleKnob(t.Amp.Master),
		Bass:   unscaleKnob(t.Amp.Bass),
		Mid:    unscaleKnob(t.Amp.Mid),
		Treble: unscaleKnob(t.Amp.Treble),
	}
	s.Cab = cabBySpkSimType(t.Cab.SpkSimType)
	s.Compressor = t.Compressor.Enabled
	s.CompParams = CompressorParams{
		Sustain: unscaleKnob(t.Compressor.Sustain),
		Level:   unscaleKnob(t.Compressor.Level),
	}
	s.Mod = ""
	if t.Mod.Enabled {
		s.Mod = modByAsset(t.Mod.Asset)
	}
	s.ModParams = ModParams{
		Speed:    unscaleKnob(t.Mod.Freq),
		Depth:    unscaleKnob(t.Mod.Depth),
		PreDelay: unscaleKnob(t.Mod.Pre),
		Feedback: unscaleKnob(t.Mod.Feedback),
		Mix:      unscaleKnob(t.Mod.WetDry),
	}
	s.Echo = ""
	if t.Echo.Enabled {
		s.Echo = echoByAsset(t.Echo.Asset)
	}
	s.EchoParams = EchoParams{
		Time:     unitToMs(t.Echo.Time),
		Feedback: unscaleKnob(t.Echo.Feedback),
		Bass:     unscaleKnob(t.Echo.Bass),
		Treble:   unscaleKnob(t.Echo.Treble),
		Mix:      unscaleKnob(t.Echo.WetDry),
	}
	s.Reverb = ""
	if t.Reverb.Enabled {
		s.Reverb = reverbByAsset(t.Reverb.Asset)
	}
	s.ReverbParams = ReverbParams{
		Tone: unscaleKnob(t.Reverb.Tone),
		Mix:  unscaleKnob(t.Reverb.WetDry),
	}
	// The spring reverb's length knob is Time; the others use Decay/PreDelay.
	if t.Reverb.Asset == "StandardSpring" {
		s.ReverbParams.Decay = unscaleKnob(t.Reverb.Time)
	} else {
		s.ReverbParams.Decay = unscaleKnob(t.Reverb.Decay)
		s.ReverbParams.PreDelay = unitToMs(t.Reverb.PreDelay)
	}
	s.NoiseGate = t.Gate.Enabled
	s.GateParams = GateParams{
		Threshold: ungateThresh(t.Gate.Thresh),
		Decay:     unscaleKnob(t.Gate.Decay),
	}
	return s, nil
}

// invert returns a name→key map's inverse (key→name).
func invert(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

var (
	ampAssetInverse    = invert(ampAsset)
	modAssetInverse    = invert(modAsset)
	echoAssetInverse   = invert(echoAsset)
	reverbAssetInverse = invert(reverbAsset)
)

// ampCellByAsset resolves an engine @asset key to the amp cell name, or "" when
// the asset is unknown (a preset from a newer firmware, say).
func ampCellByAsset(asset string) string { return ampAssetInverse[asset] }

// cabBySpkSimType resolves a SpkSimType index to the cabinet name; 16 (bypass)
// and out-of-range indices resolve to "" (no cabinet).
func cabBySpkSimType(index int) string {
	for name, id := range cabSpkSimType {
		if id == index {
			return name
		}
	}
	return ""
}

func modByAsset(asset string) string    { return modAssetInverse[asset] }
func echoByAsset(asset string) string   { return echoAssetInverse[asset] }
func reverbByAsset(asset string) string { return reverbAssetInverse[asset] }
