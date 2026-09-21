package mooer

import "testing"

// DetectModel must tell the three file-capable models apart from their bytes
// alone: each writes a different layout, so a file can be assigned to its
// device before it is decoded.
func TestDetectModelIdentifiesEachLayout(t *testing.T) {
	for _, tc := range []struct {
		model string
	}{
		{"ge150pro"},
		{"ge200"},
		{"ge100pro"},
	} {
		m, ok := ModelByName(tc.model)
		if !ok {
			t.Fatalf("unknown model %q", tc.model)
		}
		det, err := DetectModel(MarshalMOFor(m, New()))
		if err != nil {
			t.Fatalf("DetectModel(%s output): %v", tc.model, err)
		}
		if det.Model.Name != tc.model {
			t.Fatalf("DetectModel(%s output) = %q, want %q", tc.model, det.Model.Name, tc.model)
		}
		if det.Reason == "" {
			t.Fatalf("DetectModel(%s output) returned no reason", tc.model)
		}
	}
}

// A real device export is detected by its header, never by the loose record
// layout that accepts anything long enough.
func TestDetectModelRealExports(t *testing.T) {
	for _, fixture := range []string{"ge200-clean.mo", "ge200-lead.mo"} {
		det, err := DetectMOFile("testdata/" + fixture)
		if err != nil {
			t.Fatalf("DetectMOFile(%s): %v", fixture, err)
		}
		if det.Model.Name != "ge200" {
			t.Fatalf("DetectMOFile(%s) = %q, want ge200", fixture, det.Model.Name)
		}
	}
}

// A file too short for any layout is refused rather than guessed.
func TestDetectModelRejectsUnknown(t *testing.T) {
	for _, data := range [][]byte{
		{},
		[]byte("not a preset"),
		make([]byte, MOPresetOffset+PresetSize-1), // one byte short of the record layout
	} {
		if _, err := DetectModel(data); err == nil {
			t.Fatalf("DetectModel(%d bytes) succeeded, want an error", len(data))
		}
	}
}

// The record layout is a catch-all: any .mo long enough to hold a preset and
// not carrying another layout's signature is a GE150 Pro Li record. That is the
// point of the heuristic - an all-zero-header file is the GE150 Pro Li layout,
// not a GE200 one, and the detection must say so.
func TestDetectModelDefaultsToGE150ProRecord(t *testing.T) {
	det, err := DetectModel(make([]byte, ge200FileSize))
	if err != nil {
		t.Fatalf("DetectModel(all-zero 2048 bytes): %v", err)
	}
	if det.Model.Name != "ge150pro" {
		t.Fatalf("DetectModel(all-zero 2048 bytes) = %q, want ge150pro", det.Model.Name)
	}
}
