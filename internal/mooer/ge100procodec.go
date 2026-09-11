package mooer

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// The GE100 Pro .mo file is not a preset record, it is a dump of the frames
// "Mooer Studio For GE100 Pro" collects from the device when a preset is
// exported: the editor asks the device for frame 1..N (USB command 171) and
// writes the payloads out verbatim:
//
//	u16 frameCount
//	u16 frameLen[frameCount]
//	frame payloads, concatenated
//
// Import is the same pipeline in reverse: the editor replays the frames to the
// device one by one (command 173) and expects an acknowledgement per frame. A
// file that is not a frame dump is rejected before a single byte is sent, which
// is why a GE150 Pro Li shaped record cannot be imported here.
//
// A single-preset export carries two frames. The first is device-defined and
// never interpreted by the editor; the second is the 528-byte preset body:
//
//	0..15    preset name, 16 bytes, NUL padded
//	16..515  ten 50-byte chain slots
//	516..527 pedal block, six little-endian u16
//
// and each slot is:
//
//	+0   u16 present   1 when a module occupies the slot
//	+2   u16 kind      0=FX 1=DS 2=AMP 3=CAB 4=NS 5=EQ 6=MOD 7=DLY 8=REV
//	+4   u16 switch    1 when the module is on
//	+6   u16 model     the model's index within its module's list
//	+8   u16 params[10]
//	+28  u8  memory flag / memory index (user IR or sample reference)
//	+30  20-byte memory name
//
// Because the device validates the frames it is handed, an export cannot be
// synthesised from scratch: new presets patch an embedded real export so the
// bytes we do not model keep device-accepted values.
const (
	ge100ProSlotCount   = 10
	ge100ProSlotSize    = 50
	ge100ProNameSize    = 16
	ge100ProParamCount  = 10
	ge100ProMemorySize  = 20
	ge100ProPedalSize   = 12
	ge100ProSlotsOff    = ge100ProNameSize
	ge100ProPedalOff    = ge100ProSlotsOff + ge100ProSlotCount*ge100ProSlotSize // 516
	ge100ProBodySize    = ge100ProPedalOff + ge100ProPedalSize                  // 528
	ge100ProPedalFields = ge100ProPedalSize / 2
	// ge100ProMaxFrames is the sanity bound the editor applies to a frame dump
	// before it starts replaying frames.
	ge100ProMaxFrames = 1000
)

// Module kinds as numbered on the wire.
const (
	ge100ProKindFX     = 0
	ge100ProKindDS     = 1
	ge100ProKindAmp    = 2
	ge100ProKindCab    = 3
	ge100ProKindNS     = 4
	ge100ProKindEQ     = 5
	ge100ProKindMod    = 6
	ge100ProKindDelay  = 7
	ge100ProKindReverb = 8
)

// ge100ProMemoryUnset is the memory name a slot carries when it references no
// user sample; the device itself writes this placeholder.
const ge100ProMemoryUnset = "MEMORY"

// The frame a single-preset export leads with is device-defined and the editor
// never interprets it, so a preset we write carries the value every export we
// have seen carries.
var ge100ProLeadingFrame = []byte{2, 0}

// Pedal block fields, in wire order: control module, control parameter, control
// switch, volume switch, volume minimum, volume maximum.
const (
	ge100ProPedalVolumeSwitch = 3
	ge100ProPedalVolumeMax    = 5
)

// ge100ProEmptyBody is the payload of a preset with an empty chain: no slots
// filled and the device's own pedal defaults. It is built here rather than
// copied from an export, so the package ships no preset content.
func ge100ProEmptyBody(name string) ge100ProBody {
	body := ge100ProBody{Name: name}
	for i := range body.Slots {
		body.Slots[i] = ge100ProSlot{Memory: emptyGE100ProMemory()}
	}
	body.Pedal[ge100ProPedalVolumeSwitch] = 1
	body.Pedal[ge100ProPedalVolumeMax] = 100
	return body
}

// ge100ProMemory is a slot's reference to a user sample (cab IR or amp GNR
// capture). It is carried verbatim: we do not model user samples, and the
// device round-trips the reference.
type ge100ProMemory struct {
	Flag  uint8
	Index uint8
	Name  string
}

// ge100ProSlot is one of the ten chain slots of a GE100 Pro preset.
type ge100ProSlot struct {
	Present bool
	Kind    uint8
	On      bool
	Model   uint8
	Params  [ge100ProParamCount]uint16
	Memory  ge100ProMemory
}

// empty reports whether the slot holds no module.
func (s ge100ProSlot) empty() bool { return !s.Present }

// ge100ProBody is the preset payload of a GE100 Pro export: the name, the ten
// chain slots and the pedal block.
type ge100ProBody struct {
	Name  string
	Slots [ge100ProSlotCount]ge100ProSlot
	Pedal [ge100ProPedalFields]uint16
}

// ge100ProCodec is the GE100 Pro .mo layout: a dump of the frames the device
// hands the editor. A file that is a consistent frame dump cannot be one of the
// record layouts, whose zeroed 0x200-byte header reads as "no frames".
type ge100ProCodec struct{}

func (ge100ProCodec) Marshal(p Preset) []byte { return marshalGE100Pro(p) }

func (ge100ProCodec) Unmarshal(data []byte) (Preset, error) { return unmarshalGE100Pro(data) }

func (ge100ProCodec) Match(data []byte) bool {
	_, err := parseGE100ProFile(data)
	return err == nil
}

// parseGE100ProSlot decodes one 50-byte slot record.
func parseGE100ProSlot(src []byte) ge100ProSlot {
	slot := ge100ProSlot{
		Present: binary.LittleEndian.Uint16(src[0:]) != 0,
		Kind:    uint8(binary.LittleEndian.Uint16(src[2:])), // #nosec G115 -- u16 field holding 0..8
		On:      binary.LittleEndian.Uint16(src[4:]) != 0,
		Model:   uint8(binary.LittleEndian.Uint16(src[6:])), // #nosec G115 -- u16 field holding a small index
		Memory: ge100ProMemory{
			Flag:  src[28],
			Index: src[29],
			Name:  trimDeviceName(src[30 : 30+ge100ProMemorySize]),
		},
	}
	for i := range slot.Params {
		slot.Params[i] = binary.LittleEndian.Uint16(src[8+2*i:])
	}
	return slot
}

// marshal writes the slot back into its 50-byte wire form.
func (s ge100ProSlot) marshal(dst []byte) {
	binary.LittleEndian.PutUint16(dst[0:], boolU16(s.Present))
	binary.LittleEndian.PutUint16(dst[2:], uint16(s.Kind))
	binary.LittleEndian.PutUint16(dst[4:], boolU16(s.On))
	binary.LittleEndian.PutUint16(dst[6:], uint16(s.Model))
	for i, p := range s.Params {
		binary.LittleEndian.PutUint16(dst[8+2*i:], p)
	}
	dst[28], dst[29] = s.Memory.Flag, s.Memory.Index
	copy(dst[30:30+ge100ProMemorySize], asciiName(s.Memory.Name, ge100ProMemorySize))
}

// parseGE100ProBody decodes the 528-byte preset payload.
func parseGE100ProBody(src []byte) (ge100ProBody, error) {
	if len(src) != ge100ProBodySize {
		return ge100ProBody{}, fmt.Errorf("GE100 Pro preset body is %d bytes, need %d", len(src), ge100ProBodySize)
	}
	var body ge100ProBody
	body.Name = trimDeviceName(src[:ge100ProNameSize])
	for i := range body.Slots {
		off := ge100ProSlotsOff + i*ge100ProSlotSize
		body.Slots[i] = parseGE100ProSlot(src[off : off+ge100ProSlotSize])
	}
	for i := range body.Pedal {
		body.Pedal[i] = binary.LittleEndian.Uint16(src[ge100ProPedalOff+2*i:])
	}
	return body, nil
}

// trimDeviceName reads a name field the way the device writes it: the text,
// then either NULs or the spaces the device pads the field with.
func trimDeviceName(b []byte) string {
	return strings.TrimRight(trimName(b), " ")
}

// Marshal renders the preset payload back to its 528 bytes.
func (b ge100ProBody) marshal() []byte {
	out := make([]byte, ge100ProBodySize)
	copy(out[:ge100ProNameSize], asciiName(b.Name, ge100ProNameSize))
	for i, slot := range b.Slots {
		off := ge100ProSlotsOff + i*ge100ProSlotSize
		slot.marshal(out[off : off+ge100ProSlotSize])
	}
	for i, v := range b.Pedal {
		binary.LittleEndian.PutUint16(out[ge100ProPedalOff+2*i:], v)
	}
	return out
}

// ge100ProFile is a decoded .mo: the device frames in import order plus the
// frame index that carries the preset body.
type ge100ProFile struct {
	Frames    [][]byte
	BodyFrame int
}

// parseGE100ProFile validates and splits a frame dump.
func parseGE100ProFile(data []byte) (ge100ProFile, error) {
	count, err := ge100ProFrameCount(data)
	if err != nil {
		return ge100ProFile{}, err
	}
	lengths, err := ge100ProFrameLengths(data, count)
	if err != nil {
		return ge100ProFile{}, err
	}
	return splitGE100ProFrames(data, lengths)
}

// ge100ProFrameCount reads the leading frame count, bounded the way the editor
// bounds it before it starts replaying frames.
func ge100ProFrameCount(data []byte) (int, error) {
	if len(data) < 2 {
		return 0, fmt.Errorf(".mo file is %d bytes, too short for a GE100 Pro frame dump", len(data))
	}
	count := int(binary.LittleEndian.Uint16(data))
	if count == 0 || count > ge100ProMaxFrames {
		return 0, fmt.Errorf(".mo file declares %d frames, want 1..%d", count, ge100ProMaxFrames)
	}
	return count, nil
}

// ge100ProFrameLengths reads the frame length table that follows the count.
func ge100ProFrameLengths(data []byte, count int) ([]int, error) {
	if len(data) < 2+2*count {
		return nil, fmt.Errorf(".mo file is %d bytes, too short for %d frame lengths", len(data), count)
	}
	lengths := make([]int, count)
	for i := range lengths {
		lengths[i] = int(binary.LittleEndian.Uint16(data[2+2*i:]))
	}
	return lengths, nil
}

// splitGE100ProFrames cuts the file into its frames and finds the one carrying
// the preset body. The lengths must account for the file exactly: a dump the
// device wrote has no trailing bytes, and insisting on that is what keeps a
// record-layout file from being read as a frame dump.
func splitGE100ProFrames(data []byte, lengths []int) (ge100ProFile, error) {
	file := ge100ProFile{BodyFrame: -1}
	off := 2 + 2*len(lengths)
	for i, size := range lengths {
		if off+size > len(data) {
			return ge100ProFile{}, fmt.Errorf("frame %d declares %d bytes but only %d remain", i+1, size, len(data)-off)
		}
		file.Frames = append(file.Frames, data[off:off+size])
		if size == ge100ProBodySize {
			if file.BodyFrame >= 0 {
				return ge100ProFile{}, fmt.Errorf("frames %d and %d both look like preset bodies", file.BodyFrame+1, i+1)
			}
			file.BodyFrame = i
		}
		off += size
	}
	if off != len(data) {
		return ge100ProFile{}, fmt.Errorf("frame lengths account for %d bytes, file holds %d", off, len(data))
	}
	if file.BodyFrame < 0 {
		return ge100ProFile{}, fmt.Errorf("no %d-byte preset body frame in %d frames", ge100ProBodySize, len(lengths))
	}
	return file, nil
}

// Marshal renders the frames back as a .mo file.
func (f ge100ProFile) marshal() []byte {
	total := 2 + 2*len(f.Frames)
	for _, frame := range f.Frames {
		total += len(frame)
	}
	out := make([]byte, 0, total)
	out = binary.LittleEndian.AppendUint16(out, uint16(len(f.Frames))) // #nosec G115 -- frame count is bounded by ge100ProMaxFrames
	for _, frame := range f.Frames {
		out = binary.LittleEndian.AppendUint16(out, uint16(len(frame))) // #nosec G115 -- frame sizes are at most ge100ProBodySize
	}
	for _, frame := range f.Frames {
		out = append(out, frame...)
	}
	return out
}

// body decodes the frame that carries the preset payload.
func (f ge100ProFile) body() (ge100ProBody, error) {
	if f.BodyFrame < 0 || f.BodyFrame >= len(f.Frames) {
		return ge100ProBody{}, fmt.Errorf("no preset body frame")
	}
	return parseGE100ProBody(f.Frames[f.BodyFrame])
}

// marshalGE100Pro renders a preset as a GE100 Pro .mo: the two-frame dump the
// editor exports, with the preset's own payload in the body frame.
func marshalGE100Pro(p Preset) []byte {
	file := ge100ProFile{
		Frames: [][]byte{
			ge100ProLeadingFrame,
			presetToGE100ProBody(p).marshal(),
		},
		BodyFrame: 1,
	}
	return file.marshal()
}

// unmarshalGE100Pro parses a GE100 Pro .mo file into a preset.
func unmarshalGE100Pro(data []byte) (Preset, error) {
	file, err := parseGE100ProFile(data)
	if err != nil {
		return Preset{}, err
	}
	body, err := file.body()
	if err != nil {
		return Preset{}, err
	}
	return ge100ProBodyToPreset(body), nil
}

// boolU16 encodes a flag the way the device stores it.
func boolU16(b bool) uint16 {
	if b {
		return 1
	}
	return 0
}
