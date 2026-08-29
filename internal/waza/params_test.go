package waza

import (
	"strings"
	"testing"
)

// TestReadParamsTemplate decodes the built-in template patch and checks that
// it is a neutral CLEAN patch with every effect off.
func TestReadParamsTemplate(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}
	p := tmpl.ReadParams()

	if p.AmpType != "CLEAN" {
		t.Fatalf("amp type = %q, want CLEAN", p.AmpType)
	}
	for k, v := range map[string]int{
		"gain": p.AmpGain, "volume": p.AmpVolume, "bass": p.AmpBass,
		"middle": p.AmpMiddle, "treble": p.AmpTreble, "presence": p.AmpPresence,
	} {
		if v != 50 {
			t.Fatalf("%s = %d, want 50 (noon)", k, v)
		}
	}
	if p.BoosterType != "CLEAN BOOST" || p.BoosterDrive != 0 || p.BoosterLevel != 0 {
		t.Fatalf("booster = %q/drive %d/level %d, want a transparent CLEAN BOOST", p.BoosterType, p.BoosterDrive, p.BoosterLevel)
	}
	for name, v := range map[string]string{"mod": p.ModType, "fx": p.FXType, "delay": p.DelayType, "reverb": p.ReverbType} {
		if v != "" {
			t.Fatalf("%s type = %q, want empty (off)", name, v)
		}
	}
}

// TestWriteParamsRoundTrip writes a full set of parameters onto the template
// and reads them back, proving the offsets agree for both directions.
func TestWriteParamsRoundTrip(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{
		AmpType:       "BROWN",
		AmpGain:       55,
		AmpVolume:     68,
		AmpBass:       42,
		AmpMiddle:     50,
		AmpTreble:     60,
		AmpPresence:   75,
		BoosterType:   "T-SCREAM",
		BoosterDrive:  30,
		BoosterTone:   50,
		BoosterLevel:  80,
		ModType:       "CHORUS",
		FXType:        "FLANGER",
		DelayType:     "TAPE ECHO",
		DelayTime:     380,
		DelayFeedback: 32,
		DelayLevel:    40,
		ReverbType:    "HALL REVERB",
		ReverbLevel:   45,
	}).WithName("Round Trip")

	got := out.ReadParams()
	if got.AmpType != "BROWN" || got.AmpGain != 55 || got.AmpVolume != 68 || got.AmpBass != 42 ||
		got.AmpMiddle != 50 || got.AmpTreble != 60 || got.AmpPresence != 75 {
		t.Fatalf("amp params round-trip mismatch: %+v", got)
	}
	if got.BoosterType != "T-SCREAM" || got.BoosterDrive != 30 || got.BoosterTone != 50 || got.BoosterLevel != 80 {
		t.Fatalf("booster params round-trip mismatch: %+v", got)
	}
	if got.ModType != "CHORUS" || got.FXType != "FLANGER" {
		t.Fatalf("mod/fx round-trip mismatch: %q/%q", got.ModType, got.FXType)
	}
	if got.DelayType != "TAPE ECHO" || got.DelayTime != 380 || got.DelayFeedback != 32 || got.DelayLevel != 40 {
		t.Fatalf("delay params round-trip mismatch: %+v", got)
	}
	if got.ReverbType != "HALL REVERB" || got.ReverbLevel != 45 {
		t.Fatalf("reverb params round-trip mismatch: %+v", got)
	}
	if out.Name != "Round Trip" {
		t.Fatalf("name = %q, want Round Trip", out.Name)
	}
}

// TestWriteParamsLeavesUnsetUntouched verifies that unspecified numeric knobs
// keep the template's bytes while unspecified effect blocks are turned off.
func TestWriteParamsLeavesUnsetUntouched(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{AmpType: "CRUNCH", AmpGain: 99})
	got := out.ReadParams()

	if got.AmpType != "CRUNCH" || got.AmpGain != 99 {
		t.Fatalf("specified params not applied: %+v", got)
	}
	// Unspecified amp knobs keep the template's noon values.
	if got.AmpVolume != 50 || got.AmpBass != 50 || got.AmpPresence != 50 {
		t.Fatalf("unspecified amp knobs were changed: %+v", got)
	}
	// Unspecified effect blocks must be OFF, not inherited from the template.
	for name, v := range map[string]string{"mod": got.ModType, "fx": got.FXType, "delay": got.DelayType, "reverb": got.ReverbType} {
		if v != "" {
			t.Fatalf("unspecified %s block stayed on (%q)", name, v)
		}
	}
}

// TestWriteParamsAlignsDelayTaps proves the two Dual-Delay tap lines are
// written to the requested time so a single delay never keeps the template's
// double-delay taps.
func TestWriteParamsAlignsDelayTaps(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{DelayType: "ANALOG DELAY", DelayTime: 380, DelayLevel: 40})

	if out.Raw[offDelayOnOff] != 1 {
		t.Fatalf("delay on/off = %d, want 1", out.Raw[offDelayOnOff])
	}
	if got := int(out.Raw[offDelayTimeHi])<<7 | int(out.Raw[offDelayTimeLo]); got != 380 {
		t.Fatalf("main delay time = %d, want 380", got)
	}
	for off, what := range map[int]string{offDelayD1TimeHi: "D1", offDelayD2TimeHi: "D2"} {
		if got := int(out.Raw[off])<<7 | int(out.Raw[off+1]); got != 380 {
			t.Fatalf("%s tap time = %d, want 380", what, got)
		}
	}
	if out.Raw[offDelayD1Level] != 40 || out.Raw[offDelayD2Level] != 40 {
		t.Fatalf("tap levels = %d/%d, want 40/40", out.Raw[offDelayD1Level], out.Raw[offDelayD2Level])
	}
}

// TestTypeIndexMaps guards the amp type mapping against accidental edits; the
// first three values are read back from real backups.
func TestTypeIndexMaps(t *testing.T) {
	if ampTypeIndex["FLAT"] != 1 || ampTypeIndex["CLEAN"] != 8 || ampTypeIndex["CRUNCH"] != 11 ||
		ampTypeIndex["LEAD"] != 24 || ampTypeIndex["BROWN"] != 23 {
		t.Fatalf("amp type indices wrong: %v", ampTypeIndex)
	}
	if boosterTypeIndex["T-SCREAM"] != 12 || boosterTypeIndex["BLUES DRIVE"] != 10 || boosterTypeIndex["MUFF FUZZ"] != 20 {
		t.Fatalf("booster type indices wrong: %v", boosterTypeIndex)
	}
	if modFXTypeIndex["CHORUS"] != 29 || modFXTypeIndex["FLANGER"] != 20 || modFXTypeIndex["COMP"] != 3 {
		t.Fatalf("mod/fx type indices wrong: %v", modFXTypeIndex)
	}
	if delayTypeIndex["DIGITAL DELAY"] != 0 || delayTypeIndex["ANALOG DELAY"] != 7 || delayTypeIndex["TAPE ECHO"] != 8 {
		t.Fatalf("delay type indices wrong: %v", delayTypeIndex)
	}
	if reverbTypeIndex["PLATE REVERB"] != 4 || reverbTypeIndex["SPRING REVERB"] != 5 || reverbTypeIndex["HALL REVERB"] != 3 {
		t.Fatalf("reverb type indices wrong: %v", reverbTypeIndex)
	}
}

// TestSetupCardShowsValues ensures the card renders the new knob values.
func TestSetupCardShowsValues(t *testing.T) {
	d := Default()
	s, err := d.Resolve(Spec{
		Name:          "Brown Practice",
		Amp:           "BROWN",
		Booster:       "T-SCREAM",
		Delay:         "TAPE ECHO",
		Reverb:        "HALL REVERB",
		Gain:          55,
		Volume:        68,
		Bass:          42,
		Middle:        50,
		Treble:        60,
		Presence:      75,
		BoosterDrive:  30,
		DelayTime:     380,
		DelayFeedback: 32,
		ReverbLevel:   45,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	html := d.SetupCardHTML(s)
	for _, want := range []string{
		"BROWN", "T-SCREAM", "TAPE ECHO", "HALL REVERB",
		"AMP GAIN", "55", "AMP VOLUME", "68", "AMP BASS", "42",
		"AMP MIDDLE", "50", "AMP TREBLE", "60", "AMP PRESENCE", "75",
		"BOOSTER DRIVE", "30", "DELAY TIME", "380 ms", "DELAY FEEDBACK", "32", "REVERB LEVEL", "45",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("setup card missing %q:\n%s", want, html)
		}
	}
	// Unspecified knobs must not leak zeros.
	if strings.Contains(html, "BOOSTER TONE") || strings.Contains(html, "DELAY LEVEL") {
		t.Fatalf("setup card shows unspecified knobs:\n%s", html)
	}
}
