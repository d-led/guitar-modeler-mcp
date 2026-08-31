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
	if p.BoosterType != "" || p.BoosterDrive != 0 || p.BoosterLevel != 0 {
		t.Fatalf("booster = %q/drive %d/level %d, want off", p.BoosterType, p.BoosterDrive, p.BoosterLevel)
	}
	for name, v := range map[string]string{"mod": p.ModType, "fx": p.FXType, "delay": p.DelayType, "reverb": p.ReverbType} {
		if v != "" {
			t.Fatalf("%s type = %q, want empty (off)", name, v)
		}
	}
	// The neutral template keeps the device's spatial defaults.
	if p.Position != "SURROUND" || p.Ambience != "STAGE" || p.Mode != "REVERB" {
		t.Fatalf("spatial defaults = %q/%q/%q, want SURROUND/STAGE/REVERB", p.Position, p.Ambience, p.Mode)
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

// TestWriteParamsTurnsDelay2Off proves a single requested delay switches the
// second delay block off, so the preset never carries a second repeat.
func TestWriteParamsTurnsDelay2Off(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{DelayType: "ANALOG DELAY", DelayTime: 380, DelayLevel: 40})

	if out.Raw[offDelayOnOff] != 1 {
		t.Fatalf("delay on/off = %d, want 1", out.Raw[offDelayOnOff])
	}
	if got := int(out.Raw[offDelayTimeHi])<<7 | int(out.Raw[offDelayTimeLo]); got != 380 {
		t.Fatalf("delay time = %d, want 380", got)
	}
	if out.Raw[offDelay2OnOff] != 0 {
		t.Fatalf("delay2 on/off = %d, want 0 (off)", out.Raw[offDelay2OnOff])
	}
}

// TestWriteParamsAmpGainScaled proves the amp gain knob is stored with the
// Katana scaling (stored = 20 + 0.8*gain) rather than the raw knob value.
func TestWriteParamsAmpGainScaled(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{AmpType: "CLEAN", AmpGain: 50})
	if got := out.Raw[offPreampGain]; got != 60 {
		t.Fatalf("gain byte = %d, want 60 (20 + 0.8*50)", got)
	}
	// Reading back recovers the knob value.
	if got := out.ReadParams().AmpGain; got != 50 {
		t.Fatalf("gain round-trip = %d, want 50", got)
	}
}

// TestWriteParamsBoosterOnOff proves the booster block is switched on only
// when a booster is selected.
func TestWriteParamsBoosterOnOff(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	on := tmpl.WriteParams(Params{BoosterType: "T-SCREAM", BoosterDrive: 40})
	if on.Raw[offBoosterOnOff] != 1 || on.Raw[offBoosterType] != boosterTypeIndex["T-SCREAM"] {
		t.Fatalf("booster on/type = %d/%d, want 1/%d", on.Raw[offBoosterOnOff], on.Raw[offBoosterType], boosterTypeIndex["T-SCREAM"])
	}

	off := tmpl.WriteParams(Params{})
	if off.Raw[offBoosterOnOff] != 0 {
		t.Fatalf("booster on/off = %d, want 0 (off)", off.Raw[offBoosterOnOff])
	}
	if off.ReadParams().BoosterType != "" {
		t.Fatalf("booster type = %q, want empty (off)", off.ReadParams().BoosterType)
	}
}

// TestWriteParamsSpatial proves POSITION, AMBIENCE and MODE reach the gyro,
// ambience and reverb-mode offsets.
func TestWriteParamsSpatial(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{Position: "STAGE", Ambience: "STUDIO", Mode: "DLY+REV"})
	if out.Raw[offGyroType] != gyroTypeIndex["STAGE"] {
		t.Fatalf("gyro type = %d, want %d", out.Raw[offGyroType], gyroTypeIndex["STAGE"])
	}
	if out.Raw[offAmbType] != ambienceTypeIndex["STUDIO"] {
		t.Fatalf("ambience type = %d, want %d", out.Raw[offAmbType], ambienceTypeIndex["STUDIO"])
	}
	for _, off := range []int{offModeGreen, offModeRed, offModeYellow} {
		if out.Raw[off] != modeIndex["DLY+REV"] {
			t.Fatalf("mode at %d = %d, want %d", off, out.Raw[off], modeIndex["DLY+REV"])
		}
	}
	got := out.ReadParams()
	if got.Position != "STAGE" || got.Ambience != "STUDIO" || got.Mode != "DLY+REV" {
		t.Fatalf("spatial round-trip = %q/%q/%q", got.Position, got.Ambience, got.Mode)
	}
}

// TestWriteParamsEffectKnobs proves the MOD/FX effect sub-parameters (rate,
// depth, effect level) reach the correct per-effect offsets and read back.
func TestWriteParamsEffectKnobs(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{
		ModType:   "CHORUS",
		ModParams: map[string]float64{"rate": 35, "depth": 60, "effect_level": 50},
		FXType:    "FLANGER",
		FXParams:  map[string]float64{"rate": 15, "depth": 25},
	})

	got := out.ReadParams()
	if got.ModType != "CHORUS" || got.FXType != "FLANGER" {
		t.Fatalf("mod/fx = %q/%q", got.ModType, got.FXType)
	}
	if got.ModParams["rate"] != 35 || got.ModParams["depth"] != 60 || got.ModParams["effect_level"] != 50 {
		t.Fatalf("chorus knobs = %v", got.ModParams)
	}
	if got.FXParams["rate"] != 15 || got.FXParams["depth"] != 25 {
		t.Fatalf("flanger knobs = %v", got.FXParams)
	}
}

// TestWriteParamsBlockExtras proves reverb time, delay high cut and the noise
// suppressor are encoded with their documented scalings.
func TestWriteParamsBlockExtras(t *testing.T) {
	tmpl, err := TemplatePatch()
	if err != nil {
		t.Fatalf("TemplatePatch: %v", err)
	}

	out := tmpl.WriteParams(Params{
		ReverbType:       "HALL REVERB",
		ReverbTime:       3.2,
		DelayType:        "DIGITAL DELAY",
		DelayHighCut:     7,
		NSOn:             boolPtr(false),
		BoosterType:      "T-SCREAM",
		BoosterBottom:    -30,
		BoosterSolo:      true,
		BoosterSoloLevel: 80,
	})

	if got := out.Raw[offReverbTime]; got != 31 {
		t.Fatalf("reverb time byte = %d, want 31 (round(-1+10*3.2))", got)
	}
	if got := out.Raw[offDelayHighCut]; got != 7 {
		t.Fatalf("delay high cut byte = %d, want 7", got)
	}
	if out.Raw[offNSOn] != 0 {
		t.Fatalf("NS on/off = %d, want 0 (off)", out.Raw[offNSOn])
	}
	if got := out.Raw[offBoosterBottom]; got != 20 {
		t.Fatalf("booster bottom byte = %d, want 20 (50-30)", got)
	}
	if out.Raw[offBoosterSoloSW] != 1 || out.Raw[offBoosterSoloLv] != 80 {
		t.Fatalf("booster solo = %d/%d, want 1/80", out.Raw[offBoosterSoloSW], out.Raw[offBoosterSoloLv])
	}

	got := out.ReadParams()
	if got.ReverbTime != 3.2 || got.BoosterBottom != -30 || got.DelayHighCut != 7 {
		t.Fatalf("block extras round-trip = reverb_time %v, bottom %d, high_cut %d",
			got.ReverbTime, got.BoosterBottom, got.DelayHighCut)
	}
	if got.NSOn == nil || *got.NSOn {
		t.Fatalf("NS should read as off, got %v", got.NSOn)
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
	if delayTypeIndex["DIGITAL DELAY"] != 0 || delayTypeIndex["REVERSE DELAY"] != 6 ||
		delayTypeIndex["ANALOG DELAY"] != 7 || delayTypeIndex["TAPE ECHO"] != 8 ||
		delayTypeIndex["MODULATE"] != 9 || delayTypeIndex["SDE-3000"] != 10 {
		t.Fatalf("delay type indices wrong: %v", delayTypeIndex)
	}
	if reverbTypeIndex["ROOM REVERB"] != 1 || reverbTypeIndex["HALL REVERB"] != 3 ||
		reverbTypeIndex["PLATE REVERB"] != 4 || reverbTypeIndex["SPRING REVERB"] != 5 ||
		reverbTypeIndex["MODULATE REVERB"] != 6 {
		t.Fatalf("reverb type indices wrong: %v", reverbTypeIndex)
	}
	if gyroTypeIndex["OFF"] != 0 || gyroTypeIndex["SURROUND"] != 1 || gyroTypeIndex["STATIC"] != 2 || gyroTypeIndex["STAGE"] != 3 {
		t.Fatalf("gyro type indices wrong: %v", gyroTypeIndex)
	}
	if ambienceTypeIndex["STUDIO"] != 0 || ambienceTypeIndex["STAGE"] != 1 {
		t.Fatalf("ambience type indices wrong: %v", ambienceTypeIndex)
	}
	if modeIndex["DELAY"] != 0 || modeIndex["DLY+REV"] != 1 || modeIndex["REVERB"] != 2 {
		t.Fatalf("mode indices wrong: %v", modeIndex)
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
		"GAIN", "55", "VOLUME", "68", "BASS", "42",
		"MIDDLE", "50", "TREBLE", "60", "PRESENCE", "75",
		"DRIVE", "30", "TIME", "380 ms", "FEEDBACK", "32", "LEVEL", "45",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("setup card missing %q:\n%s", want, html)
		}
	}
	// The amp knobs sit inside the AMP block, before the delay and reverb blocks.
	if amp, gain := strings.Index(html, "AMP"), strings.Index(html, "GAIN"); amp < 0 || gain < 0 || gain < amp {
		t.Fatalf("expected GAIN grouped after the AMP block:\n%s", html)
	}
	// Unspecified knobs must not leak zeros.
	if strings.Contains(html, "TONE") || strings.Contains(html, "HIGH CUT") {
		t.Fatalf("setup card shows unspecified knobs:\n%s", html)
	}
}
