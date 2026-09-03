package mooer

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func mustReadGE200Clean(t *testing.T) Preset {
	t.Helper()
	m, _ := ModelByName("ge200")
	p, err := UnmarshalMOFor(m, readFixture(t, "ge200-clean.mo"))
	if err != nil {
		t.Fatalf("UnmarshalMOFor: %v", err)
	}
	return p
}

func TestGE200ReadsRealExportName(t *testing.T) {
	if p := mustReadGE200Clean(t); p.Name != "ANTO CLEAN" {
		t.Fatalf("name = %q, want ANTO CLEAN", p.Name)
	}
}

func TestGE200ReadsRealExportAmp(t *testing.T) {
	want := Amp{Enabled: true, Type: 11, Gain: 37, Bass: 63, Mid: 80, Treble: 70, Presence: 60, Master: 80}
	if p := mustReadGE200Clean(t); p.Amp != want {
		t.Fatalf("amp = %+v, want %+v", p.Amp, want)
	}
}

func TestGE200ReadsRealExportCab(t *testing.T) {
	want := Cab{Enabled: true, Type: 7, Mic: 0, Center: 5, Distance: 40, Tube: 3}
	if p := mustReadGE200Clean(t); p.Cab != want {
		t.Fatalf("cab = %+v, want %+v", p.Cab, want)
	}
}

func TestGE200ReadsRealExportOffDrive(t *testing.T) {
	if p := mustReadGE200Clean(t); p.Drive.Enabled {
		t.Fatal("drive should be off in this preset")
	}
}

func TestGE200ReadsRealExportDelay(t *testing.T) {
	p := mustReadGE200Clean(t)
	if !p.Delay.Enabled || p.Delay.TimeMS != 500 || p.Delay.Feedback != 20 {
		t.Fatalf("delay = %+v, want on, 500 ms, feedback 20", p.Delay)
	}
}

func TestGE200ReadsRealExportReverb(t *testing.T) {
	if p := mustReadGE200Clean(t); !p.Reverb.Enabled || p.Reverb.Decay != 40 {
		t.Fatalf("reverb = %+v, want on, decay 40", p.Reverb)
	}
}

func TestGE200ReadsRealExportEQ(t *testing.T) {
	p := mustReadGE200Clean(t)
	if !p.EQ.Enabled || p.EQ.Bands[4] != ge200EQBandInverse(15) {
		t.Fatalf("eq = %+v, want band 5 = %d", p.EQ, ge200EQBandInverse(15))
	}
}

func TestGE200ReadsFreeLeadExport(t *testing.T) {
	m, _ := ModelByName("ge200")
	p, err := UnmarshalMOFor(m, readFixture(t, "ge200-lead.mo"))
	if err != nil {
		t.Fatalf("UnmarshalMOFor: %v", err)
	}
	if p.Name != "LEAD LIVEPLAYRO" {
		t.Fatalf("name = %q", p.Name)
	}
	if !p.Amp.Enabled || p.Amp.Gain != 88 || p.Amp.Master != 92 {
		t.Fatalf("amp = %+v, want gain 88, master 92", p.Amp)
	}
	if p.Delay.TimeMS != 408 {
		t.Fatalf("delay time = %d ms, want 408", p.Delay.TimeMS)
	}
}

func TestGE200RoundTripSemantic(t *testing.T) {
	m, _ := ModelByName("ge200")
	raw := readFixture(t, "ge200-clean.mo")

	first, err := UnmarshalMOFor(m, raw)
	if err != nil {
		t.Fatalf("first unmarshal: %v", err)
	}
	again, err := UnmarshalMOFor(m, MarshalMOFor(m, first))
	if err != nil {
		t.Fatalf("second unmarshal: %v", err)
	}
	if again != first {
		t.Fatalf("GE200 round trip mismatch:\n got %+v\nwant %+v", again, first)
	}
}

func TestGE200WriteLayout(t *testing.T) {
	m, _ := ModelByName("ge200")
	p := New()
	p.Name = "Tone"
	p.Amp = Amp{Enabled: true, Type: 21, Gain: 80, Bass: 60, Mid: 45, Treble: 70, Presence: 55, Master: 65}
	p.Cab = Cab{Enabled: true, Type: 7, Mic: 0, Center: 50, Distance: 50, Tube: 50}
	p.Delay = Delay{Enabled: true, Type: 0, Level: 50, Feedback: 40, TimeMS: 450, Subdivision: 1}

	raw := MarshalMOFor(m, p)

	// Name at 524, 16 bytes.
	if got := trimName(raw[ge200NameOff : ge200NameOff+ge200NameSize]); got != "Tone" {
		t.Fatalf("name bytes = %q", got)
	}
	// Type is stored as type+1; switch is 1 when enabled.
	if raw[ge200ModulesOff+0] != p.FX.Type+1 {
		t.Fatalf("fx type = %d, want %d", raw[ge200ModulesOff+0], p.FX.Type+1)
	}
	if raw[ge200ModulesOff+2*ge200ModuleSize] != p.Amp.Type+1 || raw[ge200ModulesOff+2*ge200ModuleSize+1] != 1 {
		t.Fatalf("amp module = % x", raw[ge200ModulesOff+2*ge200ModuleSize:ge200ModulesOff+3*ge200ModuleSize])
	}
	if got := binary.LittleEndian.Uint16(raw[ge200DelayOff:]); got != 450 {
		t.Fatalf("delay time = %d, want 450", got)
	}
	// Effect order is identity 1..9.
	for i := range ge200ModuleOrder {
		if raw[ge200OrderOff+i] != byte(i+1) {
			t.Fatalf("effect order[%d] = %d, want %d", i, raw[ge200OrderOff+i], i+1)
		}
	}
	// Checksum = sum of bytes 512..2047.
	var sum uint32
	for _, b := range raw[ge200OrderOff:] {
		sum += uint32(b)
	}
	if got := binary.LittleEndian.Uint16(raw[ge200ChecksumOff:]); got != uint16(sum&0xFFFF) {
		t.Fatalf("checksum = %#x, want %#x", got, uint16(sum&0xFFFF))
	}
}

func TestGE200UnmarshalRejectsShortFile(t *testing.T) {
	m, _ := ModelByName("ge200")
	if _, err := UnmarshalMOFor(m, make([]byte, ge200FileSize-1)); err == nil {
		t.Fatal("expected an error for a short GE200 .mo file")
	}
}
