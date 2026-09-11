package mooer

import "fmt"

// A Mooer device stores a preset in its own .mo layout, and the layouts are
// different enough to matter: the GE150 family writes a 0x200-byte record behind
// a zeroed header, the GE200 writes its own record with a checksum, and the
// GE100 Pro writes a dump of the frames the device hands the editor. Reads and
// writes therefore go through the device's codec - chosen once, when the device
// tables are built - instead of every call site re-deciding which layout it is
// looking at.
type codec interface {
	// Marshal renders a preset in this layout.
	Marshal(Preset) []byte
	// Unmarshal parses a file in this layout.
	Unmarshal([]byte) (Preset, error)
	// Match reports whether a file is written in this layout. Detection asks
	// the specific layouts first; see moLayouts.
	Match([]byte) bool
}

// moLayouts lists every layout a .mo file may use, most specific first. It is
// consulted only when the device is unknown - a known device reads and writes
// through its own codec - and the GE150 family record layout comes last because
// it is the one that accepts anything long enough to be a preset.
var moLayouts = []codec{ge200Codec{}, ge100ProCodec{}, recordCodec{}}

// recordCodec is the GE150 Pro Li / GE150 Max layout: a zeroed 0x200-byte
// header, the 0x200-byte preset record, then zero padding out to 0x800.
type recordCodec struct{}

func (recordCodec) Marshal(p Preset) []byte               { return MarshalMO(p) }
func (recordCodec) Unmarshal(data []byte) (Preset, error) { return UnmarshalMO(data) }
func (recordCodec) Match(data []byte) bool {
	return len(data) >= MOPresetOffset+PresetSize
}

// parseMOFile reads a .mo file whose device is not known, trying each layout in
// turn.
func parseMOFile(data []byte) (Preset, error) {
	for _, layout := range moLayouts {
		if layout.Match(data) {
			return layout.Unmarshal(data)
		}
	}
	return Preset{}, fmt.Errorf(".mo file is %d bytes, too short for any known layout", len(data))
}
