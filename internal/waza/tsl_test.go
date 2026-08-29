package waza

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestParseFixtureHeavyBrown(t *testing.T) {
	f, err := ParseTSL(readFixture(t, "Sample_Heavy_Brown.tsl"))
	if err != nil {
		t.Fatalf("ParseTSL: %v", err)
	}
	if f.Device != WazaAirDeviceID || f.Version != "1.0.0" {
		t.Fatalf("device/version = %q/%q", f.Device, f.Version)
	}
	if f.Liveset.Name != "Sample Heavy Brown" {
		t.Fatalf("liveset name = %q", f.Liveset.Name)
	}
	if len(f.Liveset.Patches) != 1 {
		t.Fatalf("patches = %d, want 1", len(f.Liveset.Patches))
	}

	s := f.FirstSpec()
	if s.Name != "Rock Legend" {
		t.Fatalf("patch name = %q", s.Name)
	}
	if s.Amp != "BROWN" || s.Booster != "T-SCREAM" || s.Reverb != "ROOM" {
		t.Fatalf("spec = %+v", s)
	}
	if s.Gain != 85 || s.Volume != 50 {
		t.Fatalf("gain/volume = %d/%d, want 85/50", s.Gain, s.Volume)
	}
}

func TestParseFixtureCleanAmbient(t *testing.T) {
	f, err := ParseTSL(readFixture(t, "Sample_Clean_Ambient.tsl"))
	if err != nil {
		t.Fatalf("ParseTSL: %v", err)
	}

	s := f.FirstSpec()
	if s.Amp != "CLEAN" || s.Mod != "CHORUS" || s.Reverb != "HALL" {
		t.Fatalf("spec = %+v", s)
	}
	if s.DelayTime != 450 {
		t.Fatalf("delay time = %d, want 450", s.DelayTime)
	}
}

// TestRoundTripFixtures proves the parser keeps every parameter verbatim so a
// parse → marshal → parse cycle is lossless for the two known samples.
func TestRoundTripFixtures(t *testing.T) {
	for _, name := range []string{"Sample_Heavy_Brown.tsl", "Sample_Clean_Ambient.tsl"} {
		t.Run(name, func(t *testing.T) {
			f, err := ParseTSL(readFixture(t, name))
			if err != nil {
				t.Fatalf("ParseTSL: %v", err)
			}
			data, err := f.Marshal()
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			f2, err := ParseTSL(data)
			if err != nil {
				t.Fatalf("re-ParseTSL: %v", err)
			}
			if !reflect.DeepEqual(f, f2) {
				t.Fatalf("round trip changed the liveset:\nfirst:  %+v\nsecond: %+v", f, f2)
			}
		})
	}
}

func TestNewTSLFileWritesKnownParams(t *testing.T) {
	d := Default()
	spec, err := d.Resolve(Spec{
		Name:      "Brown Practice",
		Amp:       "BROWN",
		Booster:   "T-SCREAM",
		Delay:     "TAPE ECHO",
		DelayTime: 420,
		Reverb:    "HALL REVERB",
		Gain:      85,
		Volume:    50,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	f := NewTSLFile(spec)
	if f.Device != WazaAirDeviceID || f.Version != "1.0.0" {
		t.Fatalf("device/version = %q/%q", f.Device, f.Version)
	}
	p := f.Liveset.Patches[0]
	if p.Name != "Brown Practice" {
		t.Fatalf("patch name = %q", p.Name)
	}

	want := map[string]any{
		"AMP_TYPE":     "BROWN",
		"AMP_GAIN":     float64(85),
		"AMP_VOLUME":   float64(50),
		"FX1_TYPE":     "BOOSTER",
		"FX1_SW":       "ON",
		"BOOSTER_TYPE": "T-SCREAM",
		"DELAY_SW":     "ON",
		"DELAY_TIME":   float64(420),
		"REVERB_SW":    "ON",
		"REVERB_TYPE":  "HALL",
	}
	got := map[string]any{}
	for _, prm := range p.Param {
		got[prm.ID] = prm.Value
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("params = %#v, want %#v", got, want)
	}
}

func TestNewTSLFileModInsteadOfBooster(t *testing.T) {
	f := NewTSLFile(Spec{Name: "Ambient", Amp: "CLEAN", Mod: "CHORUS"})
	p := f.Liveset.Patches[0]
	if p.String("FX1_TYPE") != "CHORUS" || p.String("BOOSTER_TYPE") != "" {
		t.Fatalf("FX1_TYPE/BOOSTER_TYPE = %q/%q", p.String("FX1_TYPE"), p.String("BOOSTER_TYPE"))
	}
	if p.String("FX1_SW") != "ON" {
		t.Fatalf("FX1_SW = %q, want ON", p.String("FX1_SW"))
	}
}

func TestWriteReadTSLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Brown Practice.tsl")

	f := NewTSLFile(Spec{Name: "Brown Practice", Amp: "BROWN", Booster: "T-SCREAM"})
	if err := WriteTSLFile(path, f); err != nil {
		t.Fatalf("WriteTSLFile: %v", err)
	}

	back, err := ReadTSLFile(path)
	if err != nil {
		t.Fatalf("ReadTSLFile: %v", err)
	}
	if !reflect.DeepEqual(f, back) {
		t.Fatalf("write/read round trip changed the liveset:\nfirst:  %+v\nsecond: %+v", f, back)
	}
}

func TestParseTSLRejectsNonTSL(t *testing.T) {
	if _, err := ParseTSL([]byte(`not json`)); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	if _, err := ParseTSL([]byte(`{"device":"WAZA-AIR"}`)); err == nil {
		t.Fatal("expected an error when version is missing")
	}
}
