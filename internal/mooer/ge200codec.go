package mooer

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
)

// The GE200 .mo format is structurally different from the GE150 Pro Li .mo
// format (which is what the rest of this package was reverse-engineered from).
// The GE200 layout, cross-checked against real device exports and the
// community GE200→GE150 converters (sonnm / mooerMoConvert):
//
//	0x000..0x1FB  header — implementation artifacts of the GE200 editor; a
//	              zeroed header with byte 1 = 8 and byte 8 = 1 loads fine.
//	0x1FC..0x1FD  checksum = sum of bytes 0x200..0x7FF (little-endian u16).
//	0x200..0x208  effect order: 9 module ids (1=FX … 9=REVERB).
//	0x20C..0x21B  preset name, 16 bytes null-padded.
//	0x21C..0x263  9 modules × 8 bytes: [type+1, switch, 6 param bytes].
//	0x35E..0x35F  delay time, little-endian u16 milliseconds.
const (
	ge200FileSize    = 0x800 // 2048
	ge200NameSize    = 16
	ge200NameOff     = 524 // 0x20C
	ge200ModulesOff  = 540 // 0x21C
	ge200ModuleSize  = 8
	ge200OrderOff    = 512 // 0x200
	ge200ChecksumOff = 508 // 0x1FC
	ge200DelayOff    = 862 // 0x35E
)

// ge200ModuleOrder matches ModuleOrder: the nine modules in their fixed
// on-wire positions.
var ge200ModuleOrder = []string{"fx", "od", "amp", "cab", "ns", "eq", "mod", "delay", "reverb"}

// ge200Template is a real GE200 single-preset export ("P2-FIFTY1 FIFVI.mo",
// a clean header with no editor pointer garbage). New presets start from it so
// every region we do not model — the header flags, footswitch assignments and
// the tail — carries a device-accepted value instead of being invented.
//
//go:embed ge200-template.mo
var ge200Template []byte

// isGE200 reports whether a model's .mo files use the GE200 layout.
func isGE200(m Model) bool { return m.Name == "ge200" }

// ge200ModuleOff returns the file offset of a module's 8-byte record.
func ge200ModuleOff(module string) (int, bool) {
	for i, m := range ge200ModuleOrder {
		if m == module {
			return ge200ModulesOff + i*ge200ModuleSize, true
		}
	}
	return 0, false
}

// marshalGE200 renders a preset in the GE200 .mo format. It patches a
// known-good template in place so unmodeled regions stay device-valid.
func marshalGE200(p Preset) []byte {
	if len(ge200Template) != ge200FileSize {
		return marshalGE200FromScratch(p)
	}
	buf := bytes.Clone(ge200Template)
	patchGE200(buf, p)
	return buf
}

// marshalGE200FromScratch builds a GE200 file without the embedded template.
// It is a fallback used only when the template is unavailable (e.g. in a
// stripped build); the header stays zeroed except the two magic bytes.
func marshalGE200FromScratch(p Preset) []byte {
	buf := make([]byte, ge200FileSize)
	buf[1] = 8
	buf[8] = 1
	patchGE200(buf, p)
	return buf
}

// patchGE200 writes every modeled field of a preset into a GE200 record: the
// identity effect order, name, nine modules, delay time and the trailing
// checksum. Unmodeled regions (header flags, footswitch assignments, tail)
// keep whatever the buffer already held.
func patchGE200(buf []byte, p Preset) {
	// Effect order: identity (the GE200's fixed nine-module chain).
	writeGE200Order(buf)

	// Name, null-padded 16 bytes.
	copy(buf[ge200NameOff:ge200NameOff+ge200NameSize], asciiName(p.Name, ge200NameSize))

	// Modules.
	putGE200Module(buf, "fx", p.FX.Type, p.FX.Enabled, p.FX.Q, p.FX.Position, p.FX.Peak, p.FX.Level)
	putGE200Module(buf, "od", p.Drive.Type, p.Drive.Enabled, p.Drive.Volume, p.Drive.Tone, p.Drive.Gain)
	putGE200Module(buf, "amp", p.Amp.Type, p.Amp.Enabled, p.Amp.Gain, p.Amp.Bass, p.Amp.Mid, p.Amp.Treble, p.Amp.Presence, p.Amp.Master)
	putGE200Module(buf, "cab", p.Cab.Type, p.Cab.Enabled, p.Cab.Mic, p.Cab.Center, p.Cab.Distance, p.Cab.Tube)
	putGE200Module(buf, "ns", p.NoiseGate.Type, p.NoiseGate.Enabled, p.NoiseGate.Attack, p.NoiseGate.Release, p.NoiseGate.Threshold)
	putGE200Module(buf, "eq", p.EQ.Type, p.EQ.Enabled, ge200EQBand(p.EQ.Bands[0]), ge200EQBand(p.EQ.Bands[1]), ge200EQBand(p.EQ.Bands[2]), ge200EQBand(p.EQ.Bands[3]), ge200EQBand(p.EQ.Bands[4]), ge200EQBand(p.EQ.Bands[5]))
	putGE200Module(buf, "mod", p.Mod.Type, p.Mod.Enabled, p.Mod.Rate, p.Mod.Level, p.Mod.Depth, p.Mod.Param4)
	putGE200Module(buf, "delay", p.Delay.Type, p.Delay.Enabled, p.Delay.Level, p.Delay.Feedback, p.Delay.Param5, p.Delay.Param6, 0, p.Delay.Subdivision)
	putGE200Module(buf, "reverb", p.Reverb.Type, p.Reverb.Enabled, p.Reverb.PreDelay, p.Reverb.Level, p.Reverb.Decay, p.Reverb.Tone)

	// Delay time lives outside the delay module.
	binary.LittleEndian.PutUint16(buf[ge200DelayOff:], p.Delay.TimeMS)

	// Checksum: sum of everything from the effect-order region to the end.
	var sum uint32
	for _, b := range buf[ge200OrderOff:] {
		sum += uint32(b)
	}
	binary.LittleEndian.PutUint16(buf[ge200ChecksumOff:], uint16(sum&0xFFFF))
}

// putGE200Module writes one 8-byte module record: type is stored as type+1,
// the switch byte is 1 when enabled, and the remaining bytes are the module's
// parameters left-to-right (unsupplied trailing bytes stay zero).
func putGE200Module(buf []byte, module string, typ uint8, enabled bool, params ...uint8) {
	off, ok := ge200ModuleOff(module)
	if !ok {
		return
	}
	dst := buf[off : off+ge200ModuleSize]
	dst[0] = typ + 1
	dst[1] = boolByte(enabled)
	copy(dst[2:], params)
}

// ge200EQBand maps an internal EQ band (0–100, 50 = flat) to the GE200's
// stored scale (12 = flat, roughly 0..24).
func ge200EQBand(v uint8) uint8 {
	return uint8((uint16(v)*24 + 50) / 100) // #nosec G115 -- 0..100 × 24 / 100 fits in a byte
}

// ge200EQBandInverse maps a GE200 stored band back to the internal 0–100 scale.
func ge200EQBandInverse(v uint8) uint8 {
	return uint8((uint16(v)*100 + 12) / 24) // #nosec G115 -- 0..24 × 100 / 24 fits in a byte
}

// writeGE200Order writes the identity effect order 1..9.
func writeGE200Order(buf []byte) {
	order := make([]byte, len(ge200ModuleOrder))
	for i := range order {
		order[i] = byte(i + 1)
	}
	copy(buf[ge200OrderOff:ge200OrderOff+len(order)], order)
}

// unmarshalGE200 parses a GE200 .mo file.
func unmarshalGE200(data []byte) (Preset, error) {
	if len(data) != ge200FileSize {
		return Preset{}, fmt.Errorf(".mo file is %d bytes, need %d", len(data), ge200FileSize)
	}
	var p Preset
	p.Name = trimName(data[ge200NameOff : ge200NameOff+ge200NameSize])

	read := func(module string) [6]uint8 {
		off, _ := ge200ModuleOff(module)
		var out [6]uint8
		copy(out[:], data[off+2:off+ge200ModuleSize])
		return out
	}
	typ := func(module string) uint8 {
		off, _ := ge200ModuleOff(module)
		if data[off] == 0 {
			return 0
		}
		return data[off] - 1
	}
	on := func(module string) bool {
		off, _ := ge200ModuleOff(module)
		return data[off+1] != 0
	}

	fx := read("fx")
	p.FX = FX{Enabled: on("fx"), Type: typ("fx"), Q: fx[0], Position: fx[1], Peak: fx[2], Level: fx[3]}
	dr := read("od")
	p.Drive = Drive{Enabled: on("od"), Type: typ("od"), Volume: dr[0], Tone: dr[1], Gain: dr[2]}
	am := read("amp")
	p.Amp = Amp{Enabled: on("amp"), Type: typ("amp"), Gain: am[0], Bass: am[1], Mid: am[2], Treble: am[3], Presence: am[4], Master: am[5]}
	cb := read("cab")
	p.Cab = Cab{Enabled: on("cab"), Type: typ("cab"), Mic: cb[0], Center: cb[1], Distance: cb[2], Tube: cb[3]}
	ns := read("ns")
	p.NoiseGate = NoiseGate{Enabled: on("ns"), Type: typ("ns"), Attack: ns[0], Release: ns[1], Threshold: ns[2]}
	eq := read("eq")
	for i := range p.EQ.Bands {
		p.EQ.Bands[i] = ge200EQBandInverse(eq[i])
	}
	p.EQ.Enabled = on("eq")
	p.EQ.Type = typ("eq")
	md := read("mod")
	p.Mod = Mod{Enabled: on("mod"), Type: typ("mod"), Rate: md[0], Level: md[1], Depth: md[2], Param4: md[3], Param5: noon}
	dl := read("delay")
	p.Delay = Delay{
		Enabled:     on("delay"),
		Type:        typ("delay"),
		Level:       dl[0],
		Feedback:    dl[1],
		Param5:      dl[2],
		Param6:      dl[3],
		Subdivision: dl[5],
		TimeMS:      binary.LittleEndian.Uint16(data[ge200DelayOff:]),
	}
	rv := read("reverb")
	p.Reverb = Reverb{Enabled: on("reverb"), Type: typ("reverb"), PreDelay: rv[0], Level: rv[1], Decay: rv[2], Tone: rv[3]}
	return p, nil
}
