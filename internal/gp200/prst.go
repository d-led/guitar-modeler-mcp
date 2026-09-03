package gp200

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

// The binary .prst layout below was reverse-engineered by the community and
// is cross-checked against real device exports (see NOTICE.md). User presets
// are 1224 bytes; factory presets are 1176 bytes and differ only by a shorter
// footer (no controls tail, no checksum).
const (
	FileSizeUser    = 1224
	FileSizeFactory = 1176

	magic = "TSRP"

	offFormatVersion = 0x0B // high byte of the 0x08 dword: 0x06 user, 0x03 factory
	offDeviceID      = 0x10
	offFWVersion     = 0x14 // 00 <minor> 01 00
	offTimestamp     = 0x1C // u32 LE (user format)
	offMRAPPtr       = 0x20 // u32 LE = 0x28
	offMRAPSize      = 0x24 // u32 LE = file size - 0x34
	offMRAP          = 0x28 // "MRAP"
	offMRAPSize2     = 0x2C // u32 LE, repeats offMRAPSize

	offPatchSlot   = 0x34
	offPatchTempo  = 0x36 // u16 LE
	offPatchVolume = 0x38 // u16 LE, value 0..100
	offPatchPan    = 0x3A // s16 LE
	offPatchStyle  = 0x3C // u16 LE
	offFXMode      = 0x42 // u8: 0 parallel, 1 serial (byte 0x43 is preserved)

	offPatchName = 0x44
	nameSize     = 16
	offAuthor    = 0x54
	authorSize   = 16
	offNote      = 0x64
	noteSize     = 40

	offRouting      = 0x8C // 08 00 10 00
	offRoutingSlot  = 0x90 // mirror of the patch slot
	offFXSend       = 0x92
	offFXReturn     = 0x93
	offRoutingOrder = 0x94 // 11 bytes: playback position -> physical slot

	effectBlockCount = 11
	effectBlockStart = 0xA0
	effectBlockSize  = 0x48 // 72 bytes
	blockParamsOff   = 0x0C
	blockParamsCount = 15

	offChecksum = 0x4C6 // BE u16 (user format only)
)

// NameLimit is the fixed length of a preset's name field in the .prst record:
// 16 ASCII bytes at offset 0x44, null-padded.
const NameLimit = nameSize

// StoredName returns the name exactly as the device stores it — the first
// NameLimit bytes. The second result reports whether the input was truncated.
func StoredName(name string) (string, bool) {
	if len(name) <= NameLimit {
		return name, false
	}
	return name[:NameLimit], true
}

// Block is one of the eleven fixed-function effect slots.
type Block struct {
	// Slot is the physical block position (0 = PRE ... 10 = VOL).
	Slot uint8 `json:"slot"`
	// EffectID is the 32-bit effect code (see catalog_data.go).
	EffectID uint32 `json:"effect_id"`
	// Enabled reports whether the block is active.
	Enabled bool `json:"enabled"`
	// Params holds up to 15 float32 parameter values.
	Params [blockParamsCount]float32 `json:"params"`
}

// ExpAssignment is one expression-pedal assignment record. The GP-200 has three
// EXP pages (0 = EXP1 Mode A, 1 = EXP1 Mode B, 2 = EXP2), each with three
// assignable parameters (items 0..2), for nine records per preset.
type ExpAssignment struct {
	Page int `json:"page"` // 0..2
	Item int `json:"item"` // 0..2
	// Block is the target block byte: 0..10 a modeled block (PRE..VOL), 11..254
	// a special target (e.g. patch volume/tempo) kept verbatim, -1 = unassigned
	// (stored as 0xFF).
	Block      int     `json:"block"`
	ParamIndex int     `json:"param_index"`
	Min        float32 `json:"min"`
	Max        float32 `json:"max"`
}

// CtrlAssignment is one CTRL footswitch assignment record. Each of the eight
// CTRL switches stores a 12-bit mask: bit n toggles the fixed block n
// (0 = PRE .. 10 = VOL) and bit 11 the FX loop.
type CtrlAssignment struct {
	Index     int    `json:"index"` // 0..7
	BlockMask uint16 `json:"block_mask"`
	State     uint8  `json:"state"` // saved toggle position 0/1
}

// Preset is one stored sound: the eleven fixed blocks plus the per-patch
// metadata and (on user files) the controls tail.
type Preset struct {
	Version   uint8
	PatchName string
	Author    string
	Note      string
	Tempo     uint16
	Volume    uint8 // 0..100
	Pan       int16 // -50..50
	Style     uint16
	FXMode    uint8 // 0 parallel, 1 serial
	SlotIndex uint8
	FXSend    uint8 // 1..11
	FXReturn  uint8 // 1..11
	// Blocks is indexed by physical slot (Blocks[0] = PRE ... Blocks[10] = VOL).
	Blocks [effectBlockCount]Block
	// Routing is the playback order: Routing[i] is the physical slot that plays
	// at position i. Identity order (0..10) is the default serial chain.
	Routing [effectBlockCount]uint8
	// Exp is the nine expression-pedal assignment records, in page/item order.
	Exp [9]ExpAssignment
	// Ctrl is the eight CTRL footswitch assignment records.
	Ctrl [8]CtrlAssignment

	// raw preserves the original file bytes so a decode -> encode round-trip is
	// byte-exact for every region the struct does not model.
	raw []byte
}

// defaultBlockEffects is the device's fresh-patch (INIT) effect per physical
// block, in slot order (0 = PRE ... 10 = VOL). An unplaced block keeps this
// effect, so an unused NR block is a noise gate, not a stray COMP.
var defaultBlockEffects = []string{
	"COMP", "V-Wah", "Green OD", "Tweedy", "Gate 1",
	"SUP ZEP", "Guitar EQ 1", "Detune", "Pure", "Room", "Volume",
}

// defaultBlockEnabled marks the physical blocks that are on in a fresh patch:
// AMP (3), CAB (5) and VOL (10).
var defaultBlockEnabled = [effectBlockCount]bool{
	false, false, false, true, false, true, false, false, false, false, true,
}

// New returns a blank preset with every block at its module default and the
// standard per-patch, routing and pedal defaults.
func New() Preset {
	var p Preset
	for i := range p.Blocks {
		p.Blocks[i].Slot = uint8(i)
		p.Routing[i] = uint8(i)
		code, _ := EffectCode(defaultBlockEffects[i])
		p.Blocks[i].EffectID = code
		p.Blocks[i].Enabled = defaultBlockEnabled[i]
		p.Blocks[i].Params = DefaultParams(code)
	}
	p.Version = 1
	p.PatchName = "INIT"
	p.Tempo = 120
	p.Volume = 50
	p.FXSend = 4
	p.FXReturn = 4
	p.Exp = defaultExpAssignments()
	p.Ctrl = defaultCtrlAssignments()
	return p
}

// EffectName resolves a block's effect code to its display name ("" if unknown).
func EffectName(code uint32) string {
	if e, ok := EffectByCode(code); ok {
		return e.Name
	}
	return ""
}

// ModuleForBlock returns the fixed module for a physical block index.
func ModuleForBlock(slot int) string {
	if slot >= 0 && slot < len(slotModules) {
		return slotModules[slot]
	}
	return ""
}

// Unmarshal parses a .prst file (1224-byte user or 1176-byte factory).
func Unmarshal(data []byte) (Preset, error) {
	var p Preset
	if len(data) != FileSizeUser && len(data) != FileSizeFactory {
		return p, fmt.Errorf(".prst file is %d bytes, need %d or %d", len(data), FileSizeUser, FileSizeFactory)
	}
	if string(data[0:4]) != magic {
		return p, fmt.Errorf("invalid .prst magic %q", data[0:4])
	}

	p.Version = data[offFWVersion+1]
	p.PatchName = asciiField(data[offPatchName : offPatchName+nameSize])
	p.Author = asciiField(data[offAuthor : offAuthor+authorSize])
	p.Note = asciiField(data[offNote : offNote+noteSize])
	p.Tempo = binary.LittleEndian.Uint16(data[offPatchTempo:])
	vol := binary.LittleEndian.Uint16(data[offPatchVolume:])
	if vol > 100 {
		vol = 50
	}
	p.Volume = uint8(vol)
	p.Pan = int16(binary.LittleEndian.Uint16(data[offPatchPan:])) // #nosec G115 -- stored as a signed s16
	p.Style = binary.LittleEndian.Uint16(data[offPatchStyle:])
	if data[offFXMode] == 1 {
		p.FXMode = 1
	}
	p.SlotIndex = data[offPatchSlot]
	p.FXSend = data[offFXSend]
	p.FXReturn = data[offFXReturn]

	for i := 0; i < effectBlockCount; i++ {
		b := decodeBlock(data, i)
		// Index by physical position; the slot byte must agree, but a corrupt
		// byte must not reorder the block out of its own storage region.
		if b.Slot < effectBlockCount {
			p.Blocks[b.Slot] = b
		} else {
			p.Blocks[i] = b
		}
	}
	p.Routing = decodeRouting(data)

	// Factory files (1176 bytes) have no controls tail; keep the defaults.
	if len(data) == FileSizeUser {
		p.Exp, p.Ctrl = decodeControls(data)
	}

	p.raw = append([]byte(nil), data...)
	return p, nil
}

// decodeBlock parses one 72-byte effect block from its physical position.
func decodeBlock(data []byte, i int) Block {
	base := effectBlockStart + i*effectBlockSize
	b := Block{
		Slot:     data[base+4],
		Enabled:  data[base+5] == 1,
		EffectID: binary.LittleEndian.Uint32(data[base+8:]),
	}
	for prm := 0; prm < blockParamsCount; prm++ {
		bits := binary.LittleEndian.Uint32(data[base+blockParamsOff+prm*4:])
		f := math.Float32frombits(bits)
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
			f = 0 // real files occasionally store NaN in unused slots
		}
		b.Params[prm] = f
	}
	return b
}

// decodeRouting parses the 11 playback-order bytes, defaulting any out-of-range
// value to its own position so a corrupt byte cannot drop a block.
func decodeRouting(data []byte) [effectBlockCount]uint8 {
	var routing [effectBlockCount]uint8
	for i := 0; i < effectBlockCount; i++ {
		v := data[offRoutingOrder+i]
		if v < effectBlockCount {
			routing[i] = v
		} else {
			routing[i] = uint8(i)
		}
	}
	return routing
}

// Marshal encodes the preset back to a .prst file. When the preset was decoded
// from a real file the original bytes are patched in place, so every region the
// struct does not model (header padding, checksum footer region) round-trips
// byte-exact. Synthetic presets build a full 1224-byte user file from scratch
// with the canonical controls tail.
func (p *Preset) Marshal() ([]byte, error) {
	hasRaw := len(p.raw) == FileSizeUser
	buf := make([]byte, FileSizeUser)
	if hasRaw {
		copy(buf, p.raw)
	} else {
		seedHeader(buf)
		buildTail(buf, p.Exp, p.Ctrl)
	}
	p.writeModeled(buf, hasRaw)
	if hasRaw {
		patchControls(buf, p.Exp, p.Ctrl)
	}
	writeChecksum(buf)
	return buf, nil
}

// writeModeled writes every field the struct owns. It is shared by the
// raw-source and synthetic paths so the two stay in lockstep.
func (p *Preset) writeModeled(buf []byte, hasRaw bool) {
	buf[offFWVersion+1] = p.Version
	writeASCII(buf[offPatchName:offPatchName+nameSize], p.PatchName, nameSize)
	writeASCII(buf[offAuthor:offAuthor+authorSize], p.Author, authorSize)
	writeASCII(buf[offNote:offNote+noteSize], p.Note, noteSize)

	binary.LittleEndian.PutUint16(buf[offPatchTempo:], p.Tempo)
	binary.LittleEndian.PutUint16(buf[offPatchVolume:], uint16(p.Volume))
	binary.LittleEndian.PutUint16(buf[offPatchPan:], uint16(p.Pan)) // #nosec G115 -- stored as a signed s16
	binary.LittleEndian.PutUint16(buf[offPatchStyle:], p.Style)
	buf[offFXMode] = p.FXMode
	buf[offFXSend] = p.FXSend
	buf[offFXReturn] = p.FXReturn
	buf[offPatchSlot] = p.SlotIndex
	if !hasRaw {
		buf[offRoutingSlot] = p.SlotIndex
	}

	for i := 0; i < effectBlockCount; i++ {
		buf[offRoutingOrder+i] = p.Routing[i]
	}
	for slot := 0; slot < effectBlockCount; slot++ {
		b := p.Blocks[slot]
		base := effectBlockStart + slot*effectBlockSize
		buf[base+0] = 0x14
		buf[base+2] = 0x44
		buf[base+4] = uint8(slot)
		if b.Enabled {
			buf[base+5] = 1
		} else {
			buf[base+5] = 0
		}
		buf[base+6] = 0x0F
		buf[base+7] = 0x00
		binary.LittleEndian.PutUint32(buf[base+8:], b.EffectID)
		for prm := 0; prm < blockParamsCount; prm++ {
			binary.LittleEndian.PutUint32(buf[base+blockParamsOff+prm*4:], math.Float32bits(b.Params[prm]))
		}
	}
}

// seedHeader writes the canonical user-format header and routing constants for
// a synthetic preset (matching a firmware 1.8.0 export).
func seedHeader(buf []byte) {
	copy(buf[0:4], magic)
	buf[offFormatVersion] = 0x06
	copy(buf[offDeviceID:offDeviceID+4], "2-PG")
	buf[offFWVersion+0] = 0x00
	buf[offFWVersion+2] = 0x01
	buf[offFWVersion+3] = 0x00
	binary.LittleEndian.PutUint32(buf[offTimestamp:], uint32(time.Now().Unix())) // #nosec G115 -- wraps to the 32-bit file timestamp
	binary.LittleEndian.PutUint32(buf[offMRAPPtr:], offMRAP)
	binary.LittleEndian.PutUint32(buf[offMRAPSize:], FileSizeUser-0x34)
	copy(buf[offMRAP:offMRAP+4], "MRAP")
	binary.LittleEndian.PutUint32(buf[offMRAPSize2:], FileSizeUser-0x34)
	buf[0x30] = 0x02
	buf[0x32] = 0x58
	buf[offRouting+0] = 0x08
	buf[offRouting+1] = 0x00
	buf[offRouting+2] = 0x10
	buf[offRouting+3] = 0x00
}

// writeChecksum writes the BE16 sum of bytes 0..offChecksum-1 at offChecksum.
func writeChecksum(buf []byte) {
	var sum uint32
	for i := 0; i < offChecksum; i++ {
		sum += uint32(buf[i])
	}
	cs := uint16(sum & 0xFFFF)
	buf[offChecksum] = byte(cs >> 8) // #nosec G115 -- high byte of the BE16 checksum
	buf[offChecksum+1] = byte(cs)    // #nosec G115 -- low byte of the BE16 checksum
}

// Control-tail record layout (0x3B0..0x4C5), hex-verified against real exports:
//
//	0x3B0  8 zero padding bytes
//	0x3B8  9 × 16-byte EXP records   (header 0C 00 0C 00, payload 12 bytes)
//	0x448  3 × 8-byte opaque records (header 10 00 04 00, payload 4 bytes)
//	0x460  8 × 12-byte CTRL records  (header 0F 00 08 00, payload 8 bytes)
//	0x4C0  footer C0 04 00 00 00 00
const (
	ctrlTailStart = 0x3B0
	typeExp       = 0x000C
	typeUnknown10 = 0x0010
	typeCtrl      = 0x000F
)

// defaultExpAssignments returns the device-default EXP wiring: EXP1 Mode A
// Para 1 sweeps the VOL block, EXP1 Mode B Para 1 sweeps the WAH block, and
// everything else is unassigned (range 0..100).
func defaultExpAssignments() [9]ExpAssignment {
	var exp [9]ExpAssignment
	for i := range exp {
		exp[i] = ExpAssignment{Page: i / 3, Item: i % 3, Block: -1, Min: 0, Max: 100}
	}
	exp[0] = ExpAssignment{Page: 0, Item: 0, Block: 10, ParamIndex: 0, Min: 0, Max: 100}
	exp[3] = ExpAssignment{Page: 1, Item: 0, Block: 1, ParamIndex: 3, Min: 0, Max: 100}
	return exp
}

// defaultCtrlAssignments returns eight unassigned CTRL footswitches.
func defaultCtrlAssignments() [8]CtrlAssignment {
	var ctrl [8]CtrlAssignment
	for i := range ctrl {
		ctrl[i].Index = i
	}
	return ctrl
}

// decodeControls parses the EXP and CTRL assignment records from the controls
// tail. It returns the defaults when the stream is malformed or absent.
func decodeControls(data []byte) ([9]ExpAssignment, [8]CtrlAssignment) {
	exp := defaultExpAssignments()
	ctrl := defaultCtrlAssignments()
	off := skipPadding(data, ctrlTailStart)

	expSeen := 0
	ctrlSeen := 0
	for off+4 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint16(data[off+2:]))
		payloadSize, known := payloadSizeFor(typ)
		if !known {
			return exp, ctrl // footer or an unknown record type
		}
		if size != payloadSize || off+4+payloadSize > len(data) {
			return exp, ctrl
		}
		switch typ {
		case typeExp:
			if !parseExpPayload(data[off+4:], &exp, &expSeen) {
				return exp, ctrl
			}
		case typeCtrl:
			if !parseCtrlPayload(data[off+4:], &ctrl, &ctrlSeen) {
				return exp, ctrl
			}
		}
		off += 4 + payloadSize
	}
	if !controlsComplete(expSeen, ctrlSeen) {
		return defaultExpAssignments(), defaultCtrlAssignments()
	}
	return exp, ctrl
}

// controlsComplete reports whether the tail contained all nine EXP and eight
// CTRL records.
func controlsComplete(expSeen, ctrlSeen int) bool {
	return expSeen == 9 && ctrlSeen == 8
}

// parseExpPayload parses one 12-byte EXP record payload and stores it by its
// page/item slot.
func parseExpPayload(p []byte, exp *[9]ExpAssignment, seen *int) bool {
	page := int(p[0]>>4) & 0x0F
	item := int(p[0] & 0x0F)
	if page > 2 || item > 2 {
		return false // stream misaligned
	}
	block := int(p[1])
	if block == 0xFF {
		block = -1
	}
	exp[page*3+item] = ExpAssignment{
		Page:       page,
		Item:       item,
		Block:      block,
		ParamIndex: int(binary.LittleEndian.Uint16(p[2:])),
		Max:        math.Float32frombits(binary.LittleEndian.Uint32(p[4:])),
		Min:        math.Float32frombits(binary.LittleEndian.Uint32(p[8:])),
	}
	*seen = *seen + 1
	return true
}

// parseCtrlPayload parses one 8-byte CTRL record payload and stores it by its
// switch index.
func parseCtrlPayload(p []byte, ctrl *[8]CtrlAssignment, seen *int) bool {
	idx := p[0]
	if idx > 7 {
		return false // stream misaligned
	}
	state := uint8(0)
	if p[1] == 1 {
		state = 1
	}
	ctrl[idx] = CtrlAssignment{
		Index:     int(idx),
		State:     state,
		BlockMask: binary.LittleEndian.Uint16(p[4:]) & 0x0FFF,
	}
	*seen = *seen + 1
	return true
}

// patchControls overwrites the modeled assignment fields of an existing record
// stream in place, preserving record headers, the opaque 0x0010 records and the
// uninitialized bytes so a decoded file round-trips byte-exact.
func patchControls(buf []byte, exp [9]ExpAssignment, ctrl [8]CtrlAssignment) {
	off := skipPadding(buf, ctrlTailStart)
	expSeen := 0
	ctrlSeen := 0
	for off+4 <= len(buf) {
		typ := binary.LittleEndian.Uint16(buf[off:])
		size := int(binary.LittleEndian.Uint16(buf[off+2:]))
		payloadSize, known := payloadSizeFor(typ)
		if !known {
			return
		}
		if size != payloadSize || off+4+payloadSize > len(buf) {
			return
		}
		p := off + 4
		switch typ {
		case typeExp:
			if expSeen >= 9 {
				return
			}
			a := exp[expSeen]
			buf[p+1] = expBlockByte(a.Block)
			binary.LittleEndian.PutUint16(buf[p+2:], uint16(a.ParamIndex)) // #nosec G115 -- u16 field
			binary.LittleEndian.PutUint32(buf[p+4:], math.Float32bits(a.Max))
			binary.LittleEndian.PutUint32(buf[p+8:], math.Float32bits(a.Min))
			expSeen++
		case typeCtrl:
			if ctrlSeen >= 8 {
				return
			}
			a := ctrl[ctrlSeen]
			buf[p+1] = a.State
			buf[p+4] = byte(a.BlockMask)                               // #nosec G115 -- low byte of the 12-bit mask
			buf[p+5] = (buf[p+5] & 0xF0) | byte((a.BlockMask>>8)&0x0F) // #nosec G115 -- preserve the garbage high nibble
			ctrlSeen++
		}
		off += 4 + payloadSize
	}
}

// buildTail writes the full controls tail for a synthetic preset, using the
// given EXP and CTRL assignments.
func buildTail(buf []byte, exp [9]ExpAssignment, ctrl [8]CtrlAssignment) {
	off := ctrlTailStart + 8 // skip the 8 zero padding bytes

	for i := 0; i < 9; i++ {
		a := exp[i]
		binary.LittleEndian.PutUint16(buf[off:], typeExp)
		binary.LittleEndian.PutUint16(buf[off+2:], 12)
		buf[off+4] = byte((a.Page << 4) | a.Item) // #nosec G115 -- slot id is 0..0x22
		buf[off+5] = expBlockByte(a.Block)
		binary.LittleEndian.PutUint16(buf[off+6:], uint16(a.ParamIndex)) // #nosec G115 -- u16 field
		binary.LittleEndian.PutUint32(buf[off+8:], math.Float32bits(a.Max))
		binary.LittleEndian.PutUint32(buf[off+12:], math.Float32bits(a.Min))
		off += 16
	}

	unknown := [3]byte{0xFF, 0xFF, 0x0B}
	for id := 0; id < 3; id++ {
		binary.LittleEndian.PutUint16(buf[off:], typeUnknown10)
		binary.LittleEndian.PutUint16(buf[off+2:], 4)
		buf[off+4] = byte(id) // #nosec G115 -- record id is 0..2
		buf[off+5] = unknown[id]
		off += 8
	}

	for i := 0; i < 8; i++ {
		a := ctrl[i]
		binary.LittleEndian.PutUint16(buf[off:], typeCtrl)
		binary.LittleEndian.PutUint16(buf[off+2:], 8)
		buf[off+4] = byte(i) // #nosec G115 -- ctrl index is 0..7
		buf[off+5] = a.State
		binary.LittleEndian.PutUint16(buf[off+8:], a.BlockMask)
		off += 12
	}

	buf[off] = 0xC0
	buf[off+1] = 0x04
}

// expBlockByte encodes an assignment's target block: 0xFF for unassigned (-1),
// otherwise the raw block byte (0..254).
func expBlockByte(block int) byte {
	if block < 0 {
		return 0xFF
	}
	return byte(block) // #nosec G115 -- block is a validated 0..254 value
}

// skipPadding advances past the leading zero-padding bytes of a record stream,
// tolerating any even amount up to 32 bytes before giving up.
func skipPadding(buf []byte, start int) int {
	off := start
	for off+1 < len(buf) && buf[off] == 0 && buf[off+1] == 0 {
		off += 2
		if off-start > 32 {
			return start
		}
	}
	return off
}

// payloadSizeFor returns the payload size for a known record type.
func payloadSizeFor(typ uint16) (int, bool) {
	switch typ {
	case typeExp:
		return 12, true
	case typeUnknown10:
		return 4, true
	case typeCtrl:
		return 8, true
	}
	return 0, false
}

// asciiField trims a null-padded ASCII field.
func asciiField(b []byte) string {
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimSpace(string(b))
}

// writeASCII copies a name into a fixed-size field, truncating and
// null-padding as the device does.
func writeASCII(dst []byte, s string, size int) {
	for i := range dst {
		dst[i] = 0
	}
	if len(s) > size {
		s = s[:size]
	}
	copy(dst, s)
}
