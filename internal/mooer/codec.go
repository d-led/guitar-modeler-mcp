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

// moLayout ties one .mo layout to the model whose editor writes it, with a
// short reason a caller can show a user. The list is most specific first and is
// the single source of truth both for reading an unknown file and for detecting
// which device a file belongs to. The model is recorded by name and resolved at
// call time, so the detection never runs against a half-initialised catalog.
var moLayouts = []struct {
	layout codec
	model  string
	reason string
}{
	{ge200Codec{}, "ge200", "the GE200's checked header: 2048 bytes with the 08/01 magic"},
	{ge100ProCodec{}, "ge100pro", "a GE100 Pro frame dump"},
	{ge150JSONCodec{}, "ge150", "a JSON document with the GE150 Edit \"GE150 Preset\" schema"},
	{recordCodec{}, "ge150pro", "the GE150 Pro Li's zeroed 512-byte header and 512-byte preset record"},
}

// recordCodec is the GE150 Pro Li / GE150 Max layout: a zeroed 0x200-byte
// header, the 0x200-byte preset record, then zero padding out to 0x800.
type recordCodec struct{}

func (recordCodec) Marshal(p Preset) []byte               { return MarshalMO(p) }
func (recordCodec) Unmarshal(data []byte) (Preset, error) { return UnmarshalMO(data) }
func (recordCodec) Match(data []byte) bool {
	return len(data) >= MOPresetOffset+PresetSize
}

// Detection is the result of DetectModel: the model a .mo file was encoded for
// and the evidence behind the match.
type Detection struct {
	Model  Model
	Reason string
}

// DetectModel reports which Mooer model a .mo file was encoded for. The three
// file-capable models write different layouts, so a file's bytes say which
// device it belongs to. Layouts are tried most specific first - the same order
// a read uses - so a GE200 or GE100 Pro file is never mistaken for a GE150 Pro
// Li record.
func DetectModel(data []byte) (Detection, error) {
	for _, l := range moLayouts {
		if l.layout.Match(data) {
			m, ok := ModelByName(l.model)
			if !ok {
				return Detection{}, fmt.Errorf("layout matches %q but the model is unknown", l.model)
			}
			return Detection{Model: m, Reason: l.reason}, nil
		}
	}
	return Detection{}, fmt.Errorf("not a known Mooer .mo layout (%d bytes)", len(data))
}

// parseMOFile reads a .mo file whose device is not known, trying each layout in
// turn.
func parseMOFile(data []byte) (Preset, error) {
	for _, l := range moLayouts {
		if l.layout.Match(data) {
			return l.layout.Unmarshal(data)
		}
	}
	return Preset{}, fmt.Errorf(".mo file is %d bytes, too short for any known layout", len(data))
}
