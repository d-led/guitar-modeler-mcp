package thr

import (
	"os"
	"path/filepath"
	"strings"
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

func TestUnmarshalThrl6pResolvesRealExports(t *testing.T) {
	cases := []struct {
		fixture string
		name    string
		amp     string
		ampKey  string
		cab     string
		thresh  int
	}{
		{"kj1.thrl6p", "kj1", "ACOUSTIC MODERN", "THR10_Aco_Dynamic1", "", 66},
		{"bassd dled.thrl6p", "bassd dled", "BASS BOUTIQUE", "THR10_Bass_Mesa", "", 66},
		{"dled heavy 3.thrl6p", "dled heavy 3", "SPECIAL CLASSIC", "THR10X_Brown1", "Vintage 4x12", 65},
		{"dled crunch 4.3.thrl6p", "dled crunch 4.3", "CRUNCH BOUTIQUE", "THR10C_Mini", "California 1x12", 60},
	}
	for _, tc := range cases {
		s, err := UnmarshalThrl6p(readFixture(t, tc.fixture))
		if err != nil {
			t.Fatalf("%s: %v", tc.fixture, err)
		}
		if s.Name != tc.name {
			t.Fatalf("%s name = %q, want %q", tc.fixture, s.Name, tc.name)
		}
		if s.Amp != tc.amp {
			t.Fatalf("%s amp = %q, want %q", tc.fixture, s.Amp, tc.amp)
		}
		if ampAsset[s.Amp] != tc.ampKey {
			t.Fatalf("%s amp asset = %q, want %q", tc.fixture, ampAsset[s.Amp], tc.ampKey)
		}
		if s.Cab != tc.cab {
			t.Fatalf("%s cab = %q, want %q", tc.fixture, s.Cab, tc.cab)
		}
		if s.GateParams.Threshold != tc.thresh {
			t.Fatalf("%s gate threshold = %d, want %d", tc.fixture, s.GateParams.Threshold, tc.thresh)
		}
	}
}

// The gate threshold is the one knob stored in decibels: dB = (ui - 100) * 0.96.
func TestGateThresholdDecibelMapping(t *testing.T) {
	for ui, want := range map[int]int{
		66: -32,
		65: -33,
		60: -38,
	} {
		if got := gateThresh(ui); got != want {
			t.Fatalf("gateThresh(%d) = %d, want %d", ui, got, want)
		}
		if got := ungateThresh(want); got != ui {
			t.Fatalf("ungateThresh(%d) = %d, want %d", want, got, ui)
		}
	}
}

// A preset that leaves a knob unset writes the noon default (50%), not zero.
func TestMarshalThrl6pWritesNoonForUnset(t *testing.T) {
	s := NewSpec()
	s.Name = "Fresh"
	s.Amp = "CLEAN CLASSIC"
	out := string(MarshalThrl6p(s))
	for _, want := range []string{
		`"schema": "L6Preset"`,
		`"device": 2359296`,
		`"@asset": "THR10C_Deluxe"`,
		`"@asset": "speakerSimulator"`,
		`"SpkSimType": 16`,
		`"Drive": 0.5`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

// A real export must survive a decode → encode → decode round trip at the
// 0-100 knob scale, even though the stored floats are quantized.
func TestThrl6pRoundTrip(t *testing.T) {
	for _, fixture := range []string{
		"kj1.thrl6p",
		"bassd dled.thrl6p",
		"dled heavy 3.thrl6p",
		"dled crunch 4.3.thrl6p",
	} {
		first, err := UnmarshalThrl6p(readFixture(t, fixture))
		if err != nil {
			t.Fatalf("%s first unmarshal: %v", fixture, err)
		}
		again, err := UnmarshalThrl6p(MarshalThrl6p(first))
		if err != nil {
			t.Fatalf("%s second unmarshal: %v", fixture, err)
		}
		if again != first {
			t.Fatalf("%s round trip mismatch:\n got %+v\nwant %+v", fixture, again, first)
		}
	}
}
