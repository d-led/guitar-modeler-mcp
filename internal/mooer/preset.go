package mooer

import (
	"fmt"
)

// PresetSize is the fixed on-device preset record size in bytes.
const PresetSize = 0x200 // 512

// Byte offsets within a preset record, from the reverse-engineered layout.
// The first byte of every module is a header that is always zero in stored
// presets; each module's remaining bytes are, in order: enabled, effect_type,
// then the module's own parameters.
const (
	offEffectOrder = 0x00 // 10 bytes
	offSize        = 0x0A // 2 bytes, big-endian data size
	offName        = 0x0C // 14 bytes, null-padded ASCII
	offFX          = 0x1A // FXModule
	offDrive       = 0x27 // DistortionModule (DS)
	offAmp         = 0x32 // AmpModule
	offCab         = 0x43 // CabModule
	offNoiseGate   = 0x50 // NoiseGateModule (NS)
	offEQ          = 0x5B // EQModule
	offMod         = 0x72 // ModulationModule (MOD)
	offDelay       = 0x81 // DelayModule
	offReverb      = 0x92 // ReverbModule
	offTail        = 0x9F // bytes not yet reverse-engineered
)

const (
	nameSize = 14
	tailSize = PresetSize - offTail // 353
)

// NameLimit is the fixed length of a preset's name field in the .mo record:
// 14 ASCII bytes at offset 0x0C, null-padded. The device itself stores no
// more than this, so a longer name is truncated on the hardware.
const NameLimit = nameSize

// StoredName returns the name exactly as the device stores it — the first
// NameLimit bytes. The second result reports whether the input was truncated.
func StoredName(name string) (string, bool) {
	if len(name) <= NameLimit {
		return name, false
	}
	return name[:NameLimit], true
}

// Preset is one stored sound: the fixed nine-module chain plus the effect
// order, name and the opaque tail bytes the device keeps after the modules.
type Preset struct {
	EffectOrder [10]uint8
	Name        string
	FX          FX
	Drive       Drive
	Amp         Amp
	Cab         Cab
	NoiseGate   NoiseGate
	EQ          EQ
	Mod         Mod
	Delay       Delay
	Reverb      Reverb
	// Tail holds bytes 0x9F..0x1FF, which the reference implementation
	// deliberately preserves verbatim rather than silently zeroing.
	Tail [tailSize]byte
}

// New returns an empty preset with a sane effect order.
func New() Preset {
	var p Preset
	for i := range p.EffectOrder {
		p.EffectOrder[i] = uint8(i)
	}
	p.Name = "New Preset"
	return p
}

// FX is the miscellaneous-effects module (compressor, wah, volume, ...).
type FX struct {
	Enabled  bool
	Type     uint8
	Q        uint8
	Position uint8
	Peak     uint8
	Level    uint8
}

// Drive is the overdrive/distortion module (DS).
type Drive struct {
	Enabled bool
	Type    uint8
	Volume  uint8
	Tone    uint8
	Gain    uint8
}

// Amp is the amplifier model module.
type Amp struct {
	Enabled  bool
	Type     uint8
	Gain     uint8
	Bass     uint8
	Mid      uint8
	Treble   uint8
	Presence uint8
	Master   uint8
}

// Cab is the cabinet simulation module.
type Cab struct {
	Enabled  bool
	Type     uint8
	Mic      uint8
	Center   uint8
	Distance uint8
	Tube     uint8
}

// NoiseGate is the noise gate module (NS).
type NoiseGate struct {
	Enabled   bool
	Type      uint8
	Attack    uint8
	Release   uint8
	Threshold uint8
}

// EQ is the equaliser module.
type EQ struct {
	Enabled    bool
	Type       uint8
	Bands      [6]uint8
	BandsExtra [6]uint8
}

// Mod is the modulation module.
type Mod struct {
	Enabled bool
	Type    uint8
	Rate    uint8
	Level   uint8
	Depth   uint8
	Param4  uint8
	Param5  uint8
}

// Delay is the delay module. TimeMS is a 16-bit little-endian value.
type Delay struct {
	Enabled     bool
	Type        uint8
	Level       uint8
	Feedback    uint8
	TimeMS      uint16
	Subdivision uint8
	Param5      uint8
	Param6      uint8
}

// Reverb is the reverb module.
type Reverb struct {
	Enabled  bool
	Type     uint8
	PreDelay uint8
	Level    uint8
	Decay    uint8
	Tone     uint8
}

// Marshal serialises the preset to a PresetSize-byte record.
func (p Preset) Marshal() []byte {
	buf := make([]byte, PresetSize)

	for i, v := range p.EffectOrder {
		buf[offEffectOrder+i] = v
	}
	dataSize := PresetSize - offSize - 2
	buf[offSize] = byte(dataSize >> 8)
	buf[offSize+1] = byte(dataSize) // #nosec G115 -- dataSize is a small fixed record size

	copy(buf[offName:offName+nameSize], asciiName(p.Name, nameSize))

	p.FX.marshal(buf[offFX:])
	p.Drive.marshal(buf[offDrive:])
	p.Amp.marshal(buf[offAmp:])
	p.Cab.marshal(buf[offCab:])
	p.NoiseGate.marshal(buf[offNoiseGate:])
	p.EQ.marshal(buf[offEQ:])
	p.Mod.marshal(buf[offMod:])
	p.Delay.marshal(buf[offDelay:])
	p.Reverb.marshal(buf[offReverb:])

	copy(buf[offTail:], p.Tail[:])
	return buf
}

// Unmarshal parses a preset record of at least PresetSize bytes. Bytes past
// PresetSize are ignored.
func Unmarshal(data []byte) (Preset, error) {
	if len(data) < PresetSize {
		return Preset{}, fmt.Errorf("preset record is %d bytes, need %d", len(data), PresetSize)
	}
	var p Preset
	copy(p.EffectOrder[:], data[offEffectOrder:offEffectOrder+len(p.EffectOrder)])
	p.Name = trimName(data[offName : offName+nameSize])
	p.FX = unmarshalFX(data[offFX:])
	p.Drive = unmarshalDrive(data[offDrive:])
	p.Amp = unmarshalAmp(data[offAmp:])
	p.Cab = unmarshalCab(data[offCab:])
	p.NoiseGate = unmarshalNoiseGate(data[offNoiseGate:])
	p.EQ = unmarshalEQ(data[offEQ:])
	p.Mod = unmarshalMod(data[offMod:])
	p.Delay = unmarshalDelay(data[offDelay:])
	p.Reverb = unmarshalReverb(data[offReverb:])
	copy(p.Tail[:], data[offTail:offTail+tailSize])
	return p, nil
}

// boolByte encodes a module's enabled state the way the device stores it.
func boolByte(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func asciiName(name string, size int) []byte {
	out := make([]byte, size)
	for i := 0; i < size && i < len(name); i++ {
		out[i] = name[i]
	}
	return out
}

func trimName(b []byte) string {
	end := len(b)
	for i, c := range b {
		if c == 0 {
			end = i
			break
		}
	}
	return string(b[:end])
}

func (m FX) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5], dst[6] = m.Q, m.Position, m.Peak, m.Level
}

func unmarshalFX(src []byte) FX {
	return FX{Enabled: src[1] != 0, Type: src[2], Q: src[3], Position: src[4], Peak: src[5], Level: src[6]}
}

func (m Drive) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Volume, m.Tone, m.Gain
}

func unmarshalDrive(src []byte) Drive {
	return Drive{Enabled: src[1] != 0, Type: src[2], Volume: src[3], Tone: src[4], Gain: src[5]}
}

func (m Amp) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Gain, m.Bass, m.Mid
	dst[6], dst[7], dst[8] = m.Treble, m.Presence, m.Master
}

func unmarshalAmp(src []byte) Amp {
	return Amp{Enabled: src[1] != 0, Type: src[2], Gain: src[3], Bass: src[4], Mid: src[5], Treble: src[6], Presence: src[7], Master: src[8]}
}

func (m Cab) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5], dst[6] = m.Mic, m.Center, m.Distance, m.Tube
}

func unmarshalCab(src []byte) Cab {
	return Cab{Enabled: src[1] != 0, Type: src[2], Mic: src[3], Center: src[4], Distance: src[5], Tube: src[6]}
}

func (m NoiseGate) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Attack, m.Release, m.Threshold
}

func unmarshalNoiseGate(src []byte) NoiseGate {
	return NoiseGate{Enabled: src[1] != 0, Type: src[2], Attack: src[3], Release: src[4], Threshold: src[5]}
}

func (m EQ) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	copy(dst[3:9], m.Bands[:])
	copy(dst[9:15], m.BandsExtra[:])
}

func unmarshalEQ(src []byte) EQ {
	var m EQ
	m.Enabled = src[1] != 0
	m.Type = src[2]
	copy(m.Bands[:], src[3:9])
	copy(m.BandsExtra[:], src[9:15])
	return m
}

func (m Mod) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Rate, m.Level, m.Depth
	dst[6], dst[7] = m.Param4, m.Param5
}

func unmarshalMod(src []byte) Mod {
	return Mod{Enabled: src[1] != 0, Type: src[2], Rate: src[3], Level: src[4], Depth: src[5], Param4: src[6], Param5: src[7]}
}

func (m Delay) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4] = m.Level, m.Feedback
	dst[5] = byte(m.TimeMS) // #nosec G115 -- low byte of a uint16 ms value
	dst[6] = byte(m.TimeMS >> 8)
	dst[7], dst[8], dst[9] = m.Subdivision, m.Param5, m.Param6
}

func unmarshalDelay(src []byte) Delay {
	return Delay{
		Enabled:     src[1] != 0,
		Type:        src[2],
		Level:       src[3],
		Feedback:    src[4],
		TimeMS:      uint16(src[5]) | uint16(src[6])<<8,
		Subdivision: src[7],
		Param5:      src[8],
		Param6:      src[9],
	}
}

func (m Reverb) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5], dst[6] = m.PreDelay, m.Level, m.Decay, m.Tone
}

func unmarshalReverb(src []byte) Reverb {
	return Reverb{Enabled: src[1] != 0, Type: src[2], PreDelay: src[3], Level: src[4], Decay: src[5], Tone: src[6]}
}
