package waza

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseTemplatePatch(t *testing.T) {
	b, err := ParseTSL(defaultPatchTSL)
	if err != nil {
		t.Fatalf("ParseTSL: %v", err)
	}
	if b.Device != WazaAirDeviceID || b.FormatRev != "0000" {
		t.Fatalf("device/formatRev = %q/%q", b.Device, b.FormatRev)
	}

	patches := b.Patches()
	if len(patches) != 1 {
		t.Fatalf("patches = %d, want 1", len(patches))
	}
	if patches[0].Name != "Init Tone" {
		t.Fatalf("patch name = %q, want Init Tone", patches[0].Name)
	}
	if len(patches[0].Raw) != 2335 {
		t.Fatalf("raw length = %d, want 2335", len(patches[0].Raw))
	}
}

// TestBackupRoundTrip proves a parse → marshal → parse cycle keeps every hex
// byte verbatim, so a backup can be read and written back losslessly.
func TestBackupRoundTrip(t *testing.T) {
	orig, err := ParseTSL(defaultPatchTSL)
	if err != nil {
		t.Fatalf("ParseTSL: %v", err)
	}
	data, err := orig.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, err := ParseTSL(data)
	if err != nil {
		t.Fatalf("re-ParseTSL: %v", err)
	}

	if !bytes.Equal(orig.Patches()[0].Raw, back.Patches()[0].Raw) {
		t.Fatal("round trip changed the patch bytes")
	}
	if back.Name != orig.Name || back.Device != orig.Device || back.FormatRev != orig.FormatRev {
		t.Fatalf("round trip changed the header: %+v", back)
	}
}

func TestPatchWithName(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	p := tmpl.WithName("My Tone")
	if p.Name != "My Tone" {
		t.Fatalf("name = %q, want My Tone", p.Name)
	}
	if !bytes.Equal(p.Raw[16:], tmpl.Raw[16:]) {
		t.Fatal("WithName changed bytes beyond the name field")
	}

	// Names longer than 16 bytes are truncated.
	long := tmpl.WithName("A Very Long Patch Name")
	if long.Name != "A Very Long Patc" {
		t.Fatalf("truncated name = %q", long.Name)
	}
}

func TestBackupMultiplePatches(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	b := NewBackup("Two Patches")
	b.SetPatches([]Patch{
		tmpl.WithName("Clean"),
		tmpl.WithName("Lead"),
	})

	data, err := b.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, err := ParseTSL(data)
	if err != nil {
		t.Fatalf("ParseTSL: %v", err)
	}

	var names []string
	for _, p := range back.Patches() {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "Clean,Lead" {
		t.Fatalf("patch names = %v, want [Clean Lead]", names)
	}
}

func TestTemplatePatchIsStable(t *testing.T) {
	a, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}
	b, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}
	if !bytes.Equal(a.Raw, b.Raw) {
		t.Fatal("TemplatePatch is not deterministic")
	}
}

func TestParseTSLRejectsNonTSL(t *testing.T) {
	if _, err := ParseTSL([]byte(`not json`)); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	if _, err := ParseTSL([]byte(`{"name":"x","formatRev":"0000"}`)); err == nil {
		t.Fatal("expected an error when device is missing")
	}
}
