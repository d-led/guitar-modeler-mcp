package mooer

import (
	"fmt"
)

// PresetSize is the fixed on-device preset record size in bytes.
const PresetSize = 0x200 // 512

// Byte offsets within a preset record, from the reverse-engineered layout.
// Every module starts with a header byte (always zero in stored presets),
// then enabled, effect_type, then the module's own parameters. Each module
// has a fixed on-device size; the bytes we do not model are carried verbatim
// in each module's Reserved field so a real preset survives a read→write
// round trip instead of having those bytes silently zeroed.
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
	Reserved [6]byte // 13-byte module
}

// Drive is the overdrive/distortion module (DS).
type Drive struct {
	Enabled  bool
	Type     uint8
	Volume   uint8
	Tone     uint8
	Gain     uint8
	Reserved [5]byte // 11-byte module
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
	Reserved [8]byte // 17-byte module
}

// Cab is the cabinet simulation module.
type Cab struct {
	Enabled  bool
	Type     uint8
	Mic      uint8
	Center   uint8
	Distance uint8
	Tube     uint8
	Reserved [6]byte // 13-byte module
}

// NoiseGate is the noise gate module (NS).
type NoiseGate struct {
	Enabled   bool
	Type      uint8
	Attack    uint8
	Release   uint8
	Threshold uint8
	Reserved  [5]byte // 11-byte module
}

// EQ is the equaliser module.
type EQ struct {
	Enabled    bool
	Type       uint8
	Bands      [6]uint8
	BandsExtra [6]uint8
	Reserved   [8]byte // 23-byte module
}

// Mod is the modulation module.
type Mod struct {
	Enabled  bool
	Type     uint8
	Rate     uint8
	Level    uint8
	Depth    uint8
	Param4   uint8
	Param5   uint8
	Reserved [7]byte // 15-byte module
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
	Reserved    [7]byte // 17-byte module
}

// Reverb is the reverb module.
type Reverb struct {
	Enabled  bool
	Type     uint8
	PreDelay uint8
	Level    uint8
	Decay    uint8
	Tone     uint8
	Reserved [6]byte // 13-byte module
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
	copy(dst[7:], m.Reserved[:])
}

func unmarshalFX(src []byte) FX {
	var m FX
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.Q, m.Position, m.Peak, m.Level = src[3], src[4], src[5], src[6]
	copy(m.Reserved[:], src[7:13])
	return m
}

func (m Drive) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Volume, m.Tone, m.Gain
	copy(dst[6:], m.Reserved[:])
}

func unmarshalDrive(src []byte) Drive {
	var m Drive
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.Volume, m.Tone, m.Gain = src[3], src[4], src[5]
	copy(m.Reserved[:], src[6:11])
	return m
}

func (m Amp) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Gain, m.Bass, m.Mid
	dst[6], dst[7], dst[8] = m.Treble, m.Presence, m.Master
	copy(dst[9:], m.Reserved[:])
}

func unmarshalAmp(src []byte) Amp {
	var m Amp
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.Gain, m.Bass, m.Mid = src[3], src[4], src[5]
	m.Treble, m.Presence, m.Master = src[6], src[7], src[8]
	copy(m.Reserved[:], src[9:17])
	return m
}

func (m Cab) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5], dst[6] = m.Mic, m.Center, m.Distance, m.Tube
	copy(dst[7:], m.Reserved[:])
}

func unmarshalCab(src []byte) Cab {
	var m Cab
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.Mic, m.Center, m.Distance, m.Tube = src[3], src[4], src[5], src[6]
	copy(m.Reserved[:], src[7:13])
	return m
}

func (m NoiseGate) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Attack, m.Release, m.Threshold
	copy(dst[6:], m.Reserved[:])
}

func unmarshalNoiseGate(src []byte) NoiseGate {
	var m NoiseGate
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.Attack, m.Release, m.Threshold = src[3], src[4], src[5]
	copy(m.Reserved[:], src[6:11])
	return m
}

func (m EQ) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	copy(dst[3:9], m.Bands[:])
	copy(dst[9:15], m.BandsExtra[:])
	copy(dst[15:], m.Reserved[:])
}

func unmarshalEQ(src []byte) EQ {
	var m EQ
	m.Enabled = src[1] != 0
	m.Type = src[2]
	copy(m.Bands[:], src[3:9])
	copy(m.BandsExtra[:], src[9:15])
	copy(m.Reserved[:], src[15:23])
	return m
}

func (m Mod) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5] = m.Rate, m.Level, m.Depth
	dst[6], dst[7] = m.Param4, m.Param5
	copy(dst[8:], m.Reserved[:])
}

func unmarshalMod(src []byte) Mod {
	var m Mod
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.Rate, m.Level, m.Depth = src[3], src[4], src[5]
	m.Param4, m.Param5 = src[6], src[7]
	copy(m.Reserved[:], src[8:15])
	return m
}

func (m Delay) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4] = m.Level, m.Feedback
	dst[5] = byte(m.TimeMS) // #nosec G115 -- low byte of a uint16 ms value
	dst[6] = byte(m.TimeMS >> 8)
	dst[7], dst[8], dst[9] = m.Subdivision, m.Param5, m.Param6
	copy(dst[10:], m.Reserved[:])
}

func unmarshalDelay(src []byte) Delay {
	var m Delay
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.Level, m.Feedback = src[3], src[4]
	m.TimeMS = uint16(src[5]) | uint16(src[6])<<8
	m.Subdivision, m.Param5, m.Param6 = src[7], src[8], src[9]
	copy(m.Reserved[:], src[10:17])
	return m
}

func (m Reverb) marshal(dst []byte) {
	dst[0], dst[1], dst[2] = 0, boolByte(m.Enabled), m.Type
	dst[3], dst[4], dst[5], dst[6] = m.PreDelay, m.Level, m.Decay, m.Tone
	copy(dst[7:], m.Reserved[:])
}

func unmarshalReverb(src []byte) Reverb {
	var m Reverb
	m.Enabled = src[1] != 0
	m.Type = src[2]
	m.PreDelay, m.Level, m.Decay, m.Tone = src[3], src[4], src[5], src[6]
	copy(m.Reserved[:], src[7:13])
	return m
}
