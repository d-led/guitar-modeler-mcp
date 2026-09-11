package mooer

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/golden"
)

// The device's own model numbering, as Mooer Studio For GE100 Pro publishes it:
// amp 27 is MARKV_DS, cab 42 is CT-BOG OS 412, and the noise gate's models are
// NOISE KILLER, INTEL REDUCER, NOISE GATE in that order.
const (
	ge100ProMarkVLeadAmp = 27
	ge100ProBognerCab    = 42
	ge100ProIntelReducer = 1
	ge100ProTapeDelay    = 3
	ge100ProHallReverb   = 1
)

// ge100ProReferenceFile is our own reference preset: a Mark V lead with a gate
// in front, a Bogner 4x12, a tape delay and a hall reverb, designed through the
// same path mooer_design uses. It exercises every part of the format - two
// slots of the same kind are avoided on purpose, since the shared preset holds
// one module per kind - and gives the format a fixture that belongs to this
// project rather than to someone's device.
const ge100ProReferenceName = "MV LEAD"

func ge100ProReferencePreset(t *testing.T) Preset {
	t.Helper()
	m, ok := ModelByName("ge100pro")
	if !ok {
		t.Fatal("ge100pro not registered")
	}
	p, err := m.BuildPreset(Spec{
		Name:      ge100ProReferenceName,
		Amp:       "MARKV DS",
		AmpParams: Params{"gain": 95, "bass": 41, "mid": 41, "treble": 43, "presence": 51, "master": 72},
		Cab:       "CT-BOG OS 412",
		FX: []FXSpec{
			{Module: "ns", Type: "INTEL REDUCER", Enabled: true, Params: Params{"threshold": 27}},
			{Module: "delay", Type: "TAPE", Enabled: true, Params: Params{"level": 32, "feedback": 30, "time_ms": 380}},
			{Module: "reverb", Type: "HALL", Enabled: true, Params: Params{"pre_delay": 20, "level": 28, "decay": 45, "tone": 55}},
		},
	})
	if err != nil {
		t.Fatalf("designing the reference tone failed: %v", err)
	}
	return p
}

// ge100ProReferenceBytes renders the reference tone in the device's layout.
func ge100ProReferenceBytes(t *testing.T) []byte {
	t.Helper()
	m, _ := ModelByName("ge100pro")
	return MarshalMOFor(m, ge100ProReferencePreset(t))
}

// The reference file pins the bytes we write, so an accidental change to the
// layout shows up as a diff instead of a preset that no longer imports.
// Regenerate it with UPDATE_GOLDEN=1.
func TestGE100ProReferenceFile(t *testing.T) {
	golden.Assert(t, "ge100pro-reference.mo", ge100ProReferenceBytes(t))
}

// Reading our own file back must yield the preset we designed - the round trip
// the editor performs when it exports and re-imports a preset.
func TestGE100ProReferenceReadsBack(t *testing.T) {
	m, _ := ModelByName("ge100pro")
	want := ge100ProReferencePreset(t)

	got, err := UnmarshalMOFor(m, ge100ProReferenceBytes(t))
	if err != nil {
		t.Fatalf("reading our own preset failed: %v", err)
	}
	if got != withoutFieldsTheDeviceDoesNotCarry(want) {
		t.Fatalf("round trip changed the preset:\n got %+v\nwant %+v", got, want)
	}
}

// withoutFieldsTheDeviceDoesNotCarry clears the shared preset fields the GE100
// Pro has no slot for: its cab block is LOW CUT / HIGH CUT / ROOM rather than
// mic and position knobs, and its delay has four knobs, so the extra fields come
// back at their zero value.
func withoutFieldsTheDeviceDoesNotCarry(p Preset) Preset {
	p.Cab.Mic, p.Cab.Center, p.Cab.Distance, p.Cab.Tube = 0, 0, 0, 0
	p.Delay.Param5, p.Delay.Param6 = 0, 0
	return p
}

// A cab's mic and position knobs have no wire field on the GE100 Pro, so the
// slot's Hz filters stay at the device's own default instead of being invented
// from unrelated values - a HIGH CUT of 50 Hz would mute the tone.
func TestGE100ProCabKnobsAreNotInvented(t *testing.T) {
	body := ge100ProReferenceBody(t)

	for _, slot := range body.Slots {
		if slot.empty() || slot.Kind != ge100ProKindCab {
			continue
		}
		for i, knob := range slot.Params {
			if knob != 0 {
				t.Fatalf("cab knob %d = %d, want the device default 0", i+1, knob)
			}
		}
		if slot.Model != ge100ProBognerCab {
			t.Fatalf("cab model = %d, want %d", slot.Model, ge100ProBognerCab)
		}
		return
	}
	t.Fatal("the reference tone has no cab slot")
}

// What the editor resolves out of the file: the name it shows and the module in
// each slot.
func TestGE100ProReferenceChain(t *testing.T) {
	body := ge100ProReferenceBody(t)

	if body.Name != ge100ProReferenceName {
		t.Fatalf("preset name = %q, want %q", body.Name, ge100ProReferenceName)
	}

	kinds := make([]uint8, 0, ge100ProSlotCount)
	models := make([]uint8, 0, ge100ProSlotCount)
	for _, slot := range body.Slots {
		if slot.empty() {
			continue
		}
		kinds = append(kinds, slot.Kind)
		models = append(models, slot.Model)
	}
	wantKinds := []uint8{ge100ProKindNS, ge100ProKindAmp, ge100ProKindCab, ge100ProKindDelay, ge100ProKindReverb}
	wantModels := []uint8{ge100ProIntelReducer, ge100ProMarkVLeadAmp, ge100ProBognerCab, ge100ProTapeDelay, ge100ProHallReverb}
	if !equalBytes(kinds, wantKinds) {
		t.Fatalf("chain kinds = %v, want %v", kinds, wantKinds)
	}
	if !equalBytes(models, wantModels) {
		t.Fatalf("chain models = %v, want %v", models, wantModels)
	}
}

// The body sits in the second frame, in the layout the device reads. Pinning the
// offsets keeps a future edit from silently moving a field.
func TestGE100ProBodyLayout(t *testing.T) {
	file, err := parseGE100ProFile(ge100ProReferenceBytes(t))
	if err != nil {
		t.Fatalf("our output is not a frame dump: %v", err)
	}
	if len(file.Frames) != 2 {
		t.Fatalf("our output carries %d frames, want 2", len(file.Frames))
	}
	if file.BodyFrame != 1 {
		t.Fatalf("preset body is frame %d, want 1", file.BodyFrame)
	}

	body := file.Frames[file.BodyFrame]
	if len(body) != 528 {
		t.Fatalf("preset body is %d bytes, want 528", len(body))
	}
	if got := trimDeviceName(body[:16]); got != ge100ProReferenceName {
		t.Fatalf("name field = %q, want %q", got, ge100ProReferenceName)
	}
}

// A slot's fields sit at fixed offsets in its 50-byte record.
func TestGE100ProSlotLayout(t *testing.T) {
	body := ge100ProReferenceBytes(t)
	file, err := parseGE100ProFile(body)
	if err != nil {
		t.Fatalf("our output is not a frame dump: %v", err)
	}

	// Slot 0 is the gate: present and on, with the threshold in the first knob.
	slot := ge100ProSlotWire{t: t, raw: file.Frames[file.BodyFrame][16 : 16+50]}
	fields := struct {
		present, kind, on, model, threshold uint16
	}{slot.present(), slot.kind(), slot.on(), slot.model(), slot.knob(0)}
	want := struct {
		present, kind, on, model, threshold uint16
	}{
		present: 1, kind: ge100ProKindNS, on: 1,
		model: ge100ProIntelReducer, threshold: 27,
	}
	if fields != want {
		t.Fatalf("slot 0 = %+v, want %+v", fields, want)
	}
}

// The pedal block closes the body: control module, control parameter, control
// switch, volume switch, volume minimum, volume maximum.
func TestGE100ProPedalBlock(t *testing.T) {
	file, err := parseGE100ProFile(ge100ProReferenceBytes(t))
	if err != nil {
		t.Fatalf("our output is not a frame dump: %v", err)
	}
	pedal := file.Frames[file.BodyFrame][ge100ProPedalOff:]

	if got := binary.LittleEndian.Uint16(pedal[2*ge100ProPedalVolumeSwitch:]); got != 1 {
		t.Fatalf("pedal volume switch = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(pedal[2*ge100ProPedalVolumeMax:]); got != 100 {
		t.Fatalf("pedal volume maximum = %d, want 100", got)
	}
}

// Modules the tone does not use must leave their slot empty rather than inherit
// settings from anywhere: the reference tone has no drive, FX, EQ or mod.
func TestGE100ProUnusedSlotsStayEmpty(t *testing.T) {
	body := ge100ProReferenceBody(t)

	for i, slot := range body.Slots {
		if slot.empty() {
			continue
		}
		switch slot.Kind {
		case ge100ProKindDS, ge100ProKindFX, ge100ProKindEQ, ge100ProKindMod:
			t.Fatalf("slot %d holds an unused module of kind %d", i, slot.Kind)
		}
	}
}

// A frame dump we read and write back unchanged must come out byte for byte the
// same, so re-importing a file we exported is lossless.
func TestGE100ProFrameDumpRoundTripsByteForByte(t *testing.T) {
	want := ge100ProReferenceBytes(t)

	file, err := parseGE100ProFile(want)
	if err != nil {
		t.Fatalf("parsing our own file failed: %v", err)
	}
	if got := file.marshal(); !bytes.Equal(got, want) {
		t.Fatalf("re-exported file differs:\n got %x\nwant %x", got, want)
	}
}

// The leading frame is device-defined and the editor never interprets it, so a
// preset we write carries the value real exports carry rather than an invention.
func TestGE100ProWriterCarriesTheDeviceFrameLayout(t *testing.T) {
	m, _ := ModelByName("ge100pro")

	file, err := parseGE100ProFile(MarshalMOFor(m, New()))
	if err != nil {
		t.Fatalf("our own output is not a readable frame dump: %v", err)
	}
	if len(file.Frames) != 2 || file.BodyFrame != 1 {
		t.Fatalf("our output carries %d frames with the body in %d, want 2 frames and body 1", len(file.Frames), file.BodyFrame)
	}
	if !bytes.Equal(file.Frames[0], ge100ProLeadingFrame) {
		t.Fatalf("leading frame = %x, want %x", file.Frames[0], ge100ProLeadingFrame)
	}
}

// Auto-detection must keep the three layouts apart: the GE150 Pro Li and GE200
// files lead with a zeroed header and carry their own signature, neither of
// which can pass the frame-table check.
func TestLayoutAutoDetection(t *testing.T) {
	ge200, err := ReadMOFileAny("testdata/ge200-clean.mo")
	if err != nil {
		t.Fatalf("reading the GE200 fixture failed: %v", err)
	}
	if ge200.Name == "" || ge200.Amp.Type == 0 {
		t.Fatalf("GE200 fixture decoded as %+v", ge200)
	}

	if !(ge100ProCodec{}).Match(ge100ProReferenceBytes(t)) {
		t.Fatal("our own GE100 Pro file was not recognised")
	}
	for _, name := range []string{"ge150pro", "ge200"} {
		m, _ := ModelByName(name)
		if (ge100ProCodec{}).Match(MarshalMOFor(m, New())) {
			t.Fatalf("a %s file was mistaken for a GE100 Pro export", name)
		}
	}
}

// A file that is not a frame dump is rejected with a message that names what is
// wrong, rather than being parsed into a nonsensical preset.
func TestGE100ProRejectsForeignFiles(t *testing.T) {
	m, _ := ModelByName("ge100pro")

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"an empty file", nil},
		{"a GE150 Pro Li file", MarshalMO(New())},
		{"a truncated frame dump", ge100ProReferenceBytes(t)[:100]},
	} {
		if _, err := UnmarshalMOFor(m, tc.data); err == nil {
			t.Fatalf("%s was accepted as a GE100 Pro preset", tc.name)
		}
	}
}

// The catalog has to agree with the device, because a preset stores a model
// index: the model at that index is the tone the device loads, so a list that is
// short or out of order loads something else.
func TestGE100ProCatalogMatchesTheDevice(t *testing.T) {
	m, _ := ModelByName("ge100pro")

	for _, tc := range []struct {
		kind        string
		index       uint8
		want        string
		description string
	}{
		{"amp", ge100ProMarkVLeadAmp, "MARKV DS", "amp"},
		{"cab", ge100ProBognerCab, "CT-BOG OS 412", "cab"},
		{"ns", ge100ProIntelReducer, "INTEL REDUCER", "noise gate"},
		{"delay", ge100ProTapeDelay, "TAPE", "delay"},
		{"reverb", ge100ProHallReverb, "HALL", "reverb"},
	} {
		if got := m.EffectName(tc.kind, tc.index); got != tc.want {
			t.Fatalf("%s model %d = %q, want %q", tc.description, tc.index, got, tc.want)
		}
	}
}

// Index and name are the same fact seen from either side, which is what the
// writer relies on when it turns a model name into a slot value.
func TestGE100ProModelLookupRoundTrips(t *testing.T) {
	m, _ := ModelByName("ge100pro")

	for _, tc := range []struct {
		module string
		name   string
		want   uint8
	}{
		{"amp", "MARKV DS", ge100ProMarkVLeadAmp},
		{"cab", "CT-BOG OS 412", ge100ProBognerCab},
		{"ns", "INTEL REDUCER", ge100ProIntelReducer},
		{"delay", "TAPE", ge100ProTapeDelay},
		{"reverb", "HALL", ge100ProHallReverb},
	} {
		got, ok := m.EffectIndex(tc.module, tc.name)
		if !ok || got != tc.want {
			t.Fatalf("EffectIndex(%s, %s) = %d, %v; want %d", tc.module, tc.name, got, ok, tc.want)
		}
	}
}

// The hardware behind a model comes from the editor for effects and from our own
// reading of Mooer's amp names, so both routes are worth pinning.
func TestGE100ProHardwareHints(t *testing.T) {
	m, _ := ModelByName("ge100pro")

	if inspired, ok := m.InspiredAmp("MARKV DS"); !ok || inspired != "Mesa Boogie Mark V (lead)" {
		t.Fatalf("InspiredAmp(MARKV DS) = %q, %v", inspired, ok)
	}
	if inspired, ok := m.InspiredFX("od", "808"); !ok || inspired != "TS808" {
		t.Fatalf("InspiredFX(od, 808) = %q, %v; want the editor's own reference", inspired, ok)
	}
}

// A developer with the hardware can check the reader against a real export
// without this project shipping one:
//
//	MOOER_GE100PRO_EXPORT=~/path/to/export.mo go test ./internal/mooer/
func TestGE100ProReadsADeviceExport(t *testing.T) {
	path := os.Getenv("MOOER_GE100PRO_EXPORT")
	if path == "" {
		t.Skip("set MOOER_GE100PRO_EXPORT to a preset exported from the device")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s failed: %v", path, err)
	}

	m, _ := ModelByName("ge100pro")
	p, err := UnmarshalMOFor(m, data)
	if err != nil {
		t.Fatalf("the device's own export did not read as a GE100 Pro preset: %v", err)
	}
	if p.Name == "" {
		t.Fatalf("the export carries no preset name (read %+v)", p)
	}

	// Every module the export assigns must name a model in our tables, or the
	// index space has drifted from the device's.
	for _, module := range ModuleOrder {
		model, used := moduleModel(p, module)
		if !used {
			continue
		}
		if name := m.EffectName(module, model); name == "" {
			t.Fatalf("%s model %d is not in the catalog", module, model)
		}
	}
}

// ge100ProReferenceBody decodes the payload of the reference file.
func ge100ProReferenceBody(t *testing.T) ge100ProBody {
	t.Helper()
	file, err := parseGE100ProFile(ge100ProReferenceBytes(t))
	if err != nil {
		t.Fatalf("parsing our own file failed: %v", err)
	}
	body, err := file.body()
	if err != nil {
		t.Fatalf("our own file carries no preset body: %v", err)
	}
	return body
}

// ge100ProSlotWire reads a slot's fields straight off the wire, so a test can
// state where a field lives rather than only what it decodes to.
type ge100ProSlotWire struct {
	t   *testing.T
	raw []byte
}

func (s ge100ProSlotWire) present() uint16 { return s.field(0) }
func (s ge100ProSlotWire) kind() uint16    { return s.field(2) }
func (s ge100ProSlotWire) on() uint16      { return s.field(4) }
func (s ge100ProSlotWire) model() uint16   { return s.field(6) }
func (s ge100ProSlotWire) knob(n int) uint16 {
	return s.field(8 + 2*n)
}

func (s ge100ProSlotWire) field(offset int) uint16 {
	s.t.Helper()
	if len(s.raw) < offset+2 {
		s.t.Fatalf("slot record is %d bytes, too short for the field at %d", len(s.raw), offset)
	}
	return binary.LittleEndian.Uint16(s.raw[offset:])
}

// equalBytes compares two small byte slices.
func equalBytes(got, want []uint8) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
