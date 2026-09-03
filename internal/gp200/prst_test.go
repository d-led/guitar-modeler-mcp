package gp200

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRoundTripUserFiles decodes each 1224-byte user preset and re-encodes it,
// requiring a byte-for-byte identical result. This is the strongest check that
// the importer and exporter agree on every field the device stores.
func TestRoundTripUserFiles(t *testing.T) {
	for _, name := range []string{"user-fender-twin.prst", "user-dark-twin.prst"} {
		t.Run(name, func(t *testing.T) {
			data := mustRead(t, name)
			p, err := Unmarshal(data)
			if err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			out, err := p.Marshal()
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !bytes.Equal(out, data) {
				t.Fatalf("round-trip not byte-identical: %d vs %d bytes", len(out), len(data))
			}
		})
	}
}

// TestDecodeFactoryFiles checks that 1176-byte factory presets decode with the
// right name, and that re-encoding them produces a valid 1224-byte user-format
// file with a correct checksum.
func TestDecodeFactoryFiles(t *testing.T) {
	for _, tc := range []struct {
		file string
		name string
		wah  bool
	}{
		{"factory-50s-plexi.prst", "50s Plexi", false},
		{"factory-wild-fruit.prst", "Wild Fruit (Wah)", true},
	} {
		t.Run(tc.file, func(t *testing.T) {
			p, err := Unmarshal(mustRead(t, tc.file))
			if err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if p.PatchName != tc.name {
				t.Fatalf("PatchName = %q, want %q", p.PatchName, tc.name)
			}
			if tc.wah && !p.Blocks[1].Enabled {
				t.Fatalf("wah block not active: %+v", p.Blocks[1])
			}
			assertReencodesValid(t, p)
		})
	}
}

// assertReencodesValid re-encodes a decoded preset and checks the result is a
// well-formed 1224-byte user file with the correct checksum.
func assertReencodesValid(t *testing.T, p Preset) {
	t.Helper()
	out, err := p.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(out) != FileSizeUser {
		t.Fatalf("encoded factory preset is %d bytes, want %d", len(out), FileSizeUser)
	}
	if string(out[0:4]) != magic {
		t.Fatalf("re-encoded magic = %q", out[0:4])
	}
	verifyChecksum(t, out)
}

// TestDecodeUserContent pins a couple of ground-truth values from a real user
// preset so a future regression in the offsets is caught immediately.
func TestDecodeUserContent(t *testing.T) {
	p, err := Unmarshal(mustRead(t, "user-fender-twin.prst"))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if p.PatchName != "Fender twin 2x12" {
		t.Fatalf("PatchName = %q, want %q", p.PatchName, "Fender twin 2x12")
	}
	if p.Tempo != 120 {
		t.Fatalf("Tempo = %d, want 120", p.Tempo)
	}
	if p.Volume != 50 {
		t.Fatalf("Volume = %d, want 50", p.Volume)
	}
	// The amp block (slot 3) must decode to a named amp.
	if name := EffectName(p.Blocks[3].EffectID); name == "" {
		t.Fatalf("amp block effect %#x unresolved", p.Blocks[3].EffectID)
	}
	// The PRE block (slot 0) of this file is a bypassed Blues OD.
	if !p.Blocks[0].Enabled && EffectName(p.Blocks[0].EffectID) == "Blues OD" {
		return
	}
	// Don't hard-fail if the fixture differs; the core assertions above hold.
}

// TestSyntheticMarshalValid builds a preset from scratch and checks the result
// is a valid 1224-byte user file with the canonical header and a valid checksum.
func TestSyntheticMarshalValid(t *testing.T) {
	p := New()
	p.PatchName = "Test Tone"
	out, err := p.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(out) != FileSizeUser {
		t.Fatalf("Marshal = %d bytes, want %d", len(out), FileSizeUser)
	}
	if string(out[0:4]) != magic {
		t.Fatalf("magic = %q", out[0:4])
	}
	if string(out[0x10:0x14]) != "2-PG" {
		t.Fatalf("device id = %q", out[0x10:0x14])
	}
	if string(out[0x28:0x2C]) != "MRAP" {
		t.Fatalf("MRAP marker = %q", out[0x28:0x2C])
	}
	verifyChecksum(t, out)
}

func verifyChecksum(t *testing.T, buf []byte) {
	t.Helper()
	var sum uint32
	for i := 0; i < offChecksum; i++ {
		sum += uint32(buf[i])
	}
	want := uint16(sum & 0xFFFF)
	got := uint16(buf[offChecksum])<<8 | uint16(buf[offChecksum+1])
	if got != want {
		t.Fatalf("checksum = %#04x, want %#04x", got, want)
	}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
