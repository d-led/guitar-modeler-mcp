package mooer

import "testing"

func fullPreset() Preset {
	return Preset{
		EffectOrder: [10]uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		Name:        "My Tone",
		FX:          FX{Enabled: true, Type: 5, Q: 10, Position: 20, Peak: 30, Level: 40},
		Drive:       Drive{Enabled: true, Type: 11, Volume: 60, Tone: 70, Gain: 80},
		Amp:         Amp{Enabled: true, Type: 21, Gain: 90, Bass: 50, Mid: 50, Treble: 60, Presence: 40, Master: 70},
		Cab:         Cab{Enabled: true, Type: 17, Mic: 1, Center: 2, Distance: 3, Tube: 4},
		NoiseGate:   NoiseGate{Enabled: false, Type: 0, Attack: 5, Release: 6, Threshold: 7},
		EQ:          EQ{Enabled: false, Type: 0, Bands: [6]uint8{1, 2, 3, 4, 5, 6}, BandsExtra: [6]uint8{7, 8, 9, 10, 11, 12}},
		Mod:         Mod{Enabled: true, Type: 0, Rate: 13, Level: 14, Depth: 15, Param4: 16, Param5: 17},
		Delay:       Delay{Enabled: true, Type: 2, Level: 18, Feedback: 19, TimeMS: 0x1234, Subdivision: 20, Param5: 21, Param6: 22},
		Reverb:      Reverb{Enabled: true, Type: 1, PreDelay: 23, Level: 24, Decay: 25, Tone: 26},
	}
}

func TestPresetRoundTrip(t *testing.T) {
	want := fullPreset()

	got, err := Unmarshal(want.Marshal())
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != want {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestPresetWireLayout(t *testing.T) {
	p := fullPreset()
	raw := p.Marshal()

	// The reference layout places each module at a fixed offset; guard the
	// exact positions so a future edit cannot silently shift the format.
	checks := []struct {
		offset int
		value  byte
		what   string
	}{
		{offAmp + 2, 21, "amp type"},
		{offAmp + 3, 90, "amp gain"},
		{offCab + 2, 17, "cab type"},
		{offDelay + 5, 0x34, "delay time low byte"},
		{offDelay + 6, 0x12, "delay time high byte"},
		{offReverb + 2, 1, "reverb type"},
	}
	for _, c := range checks {
		if raw[c.offset] != c.value {
			t.Fatalf("%s at 0x%X = %d, want %d", c.what, c.offset, raw[c.offset], c.value)
		}
	}

	if got := string(raw[offName : offName+nameSize]); got != "My Tone\x00\x00\x00\x00\x00\x00\x00" {
		t.Fatalf("name bytes = %q", got)
	}
}

func TestPresetNameTruncatedToFourteenBytes(t *testing.T) {
	p := New()
	p.Name = "A Very Long Preset Name Beyond Limits"

	got, err := Unmarshal(p.Marshal())
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Name != "A Very Long Pr" {
		t.Fatalf("name = %q, want truncated to 14 bytes", got.Name)
	}
}

func TestTailPreserved(t *testing.T) {
	p := fullPreset()
	p.Tail[0] = 0xAA
	p.Tail[tailSize-1] = 0x55

	got, err := Unmarshal(p.Marshal())
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Tail != p.Tail {
		t.Fatal("tail bytes were not preserved verbatim")
	}
}

func TestUnmarshalRejectsShortRecord(t *testing.T) {
	if _, err := Unmarshal(make([]byte, PresetSize-1)); err == nil {
		t.Fatal("expected an error for a short record")
	}
}

func TestDefaultPresetEnabledStates(t *testing.T) {
	// A fresh preset stores all modules disabled.
	p := New()
	got, err := Unmarshal(p.Marshal())
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Amp.Enabled || got.Cab.Enabled || got.FX.Enabled {
		t.Fatalf("fresh preset should have modules disabled, got %+v", got)
	}
	if got.Name != "New Preset" {
		t.Fatalf("fresh preset name = %q", got.Name)
	}
}
